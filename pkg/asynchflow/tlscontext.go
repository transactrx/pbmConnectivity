package asynchflow

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Status int

const (
	NoData           Status = iota // Indicates that there is no data yet
	MoreDataPending                // More data is needed
	TransactionFound               // A full transaction has been found
	ParseError                     // Indicates a parsing error
)

type Site struct {
	URL    string
	Active bool
}

type Response struct {
	data   []byte
	err    error
	status Status
}

const CLAIM_QUEUE_LEN = 100

// TlsSession represents a single TLS session.
type TlsSession struct {
	name      string
	address   string
	tlsConn   *tls.Conn
	readCh    chan []byte
	readCh1   chan Response
	writeCh   chan []byte
	closeCh   chan bool
	connected atomic.Bool
	mu        sync.Mutex
	tlsConfig *tls.Config
	appConfig Config
	chnl      int
	//errors    int

	activeClaims atomic.Int32 // Tracks # of claims awaiting responses
	errorCount   atomic.Int32 // Tracks total errors
	paused       atomic.Bool  // Flag to pause connection if error threshold exceeded

}

// RegisterError increments the error count and checks for pause/disconnect.
func (s *TlsSession) RegisterError(maxErrors int, maxPause int) {
	count := s.errorCount.Add(1)
	if int32(maxPause) > 0 && count >= int32(maxPause) {
		s.paused.Store(true) // Pause if errors exceed threshold
	}
	if count >= int32(maxErrors) {
		s.Disconnect()
	}
}

// Disconnect closes the session.
func (s *TlsSession) Disconnect() {
	if s.connected.CompareAndSwap(true, false) {
		s.closeCh <- true
	}
}

// ResetErrors resets the error count when session stabilizes.
func (s *TlsSession) ResetErrors() {
	s.errorCount.Store(0)
	s.paused.Store(false)
}

// TlsContext manages multiple TLS sessions.
type TlsContext struct {
	sessions  []*TlsSession
	bitmap    []bool
	mu        sync.Mutex
	lastUsed  uint32
	sites     []*Site
	tcpDialer *net.Dialer
}

func isIPAddress(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil
}
func extractLastOctetFromHostname(hostName string) string {
	// Check if the hostname starts with "ip-" and is in the expected format
	if strings.HasPrefix(hostName, "ip-") {
		// Split by hyphen to get the components (ip-10-103-4-90 will split into parts)
		parts := strings.Split(hostName, ".")

		if len(parts) > 0 {
			ipParts := strings.Split(parts[0], "-")
			// Ensure there are enough parts to represent an IP address (ip-x-x-x-x)
			if len(ipParts) >= 5 {
				return ipParts[4] // Get the last octet (e.g., 90)
			}
		}
	}

	// If not in the expected format, return the full hostname as fallback
	return hostName
}
func createSessionName(i int, siteURL string) string {
	// Get the local machine's hostname
	hostName, _ := os.Hostname()
	hostLastOctet := extractLastOctetFromHostname(hostName)

	// Check if the siteURL is an IP address
	var targetIdentifier string
	if isIPAddress(siteURL) {
		// Extract last octet of the site URL IP
		urlParts := strings.Split(siteURL, ".")
		targetIdentifier = urlParts[len(urlParts)-1]
	} else {
		// Use full site URL if it's not an IP address
		targetIdentifier = siteURL
	}

	// Construct the name using the last octets or full strings
	tmpName := fmt.Sprintf("tls[ch:%d;f:%s;t:%s]", i, hostLastOctet, targetIdentifier)

	return tmpName
}

// NewTlsContext creates a new TlsContext with predefined sessions.
func NewTlsContext(appCfg Config) (*TlsContext, error) {
	// Parse the PbmUrl string into a slice of URLs
	//urls := strings.Split(appCfg.PbmUrl, ",")

	ctx := &TlsContext{
		sessions: make([]*TlsSession, appCfg.PbmOutboundChnls),
		bitmap:   make([]bool, appCfg.PbmOutboundChnls),
		sites:    make([]*Site, len(appCfg.PbmUrl)), // Create sites based on the number of URLs
	}

	activeSite := false
	// Initialize sites based on parsed URLs
	for i, url := range appCfg.PbmUrl {
		activeSite = false
		activeSite = appCfg.PbmActiveSites[i]
		ctx.sites[i] = &Site{URL: url, Active: activeSite}
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: appCfg.PbmInsecureSkipVerify, // You might want to set this to false in production
	}

	// Assign sessions to sites
	for i := 0; i < appCfg.PbmOutboundChnls; i++ {
		site := ctx.sites[i%len(ctx.sites)] // Round-robin assignment of sites
		addr := site.URL + ":" + appCfg.PbmPort

		session := &TlsSession{
			name:    createSessionName(i, site.URL),
			address: addr,
			readCh:  make(chan []byte),
			readCh1: make(chan Response, CLAIM_QUEUE_LEN),
			writeCh: make(chan []byte, CLAIM_QUEUE_LEN),
			closeCh: make(chan bool),
			//connected: connected.Store(false),
			tlsConfig: tlsConfig,
			appConfig: appCfg,
			chnl:      i,
		}
		session.connected.Store(false)
		ctx.sessions[i] = session
		go session.handleConnection(ctx) // Pass ctx to handleConnection
		go session.ProcessResponseWorker()
	}

	// Start monitoring with a threshold of 5 errors and a check interval of 10 seconds
	ctx.StartMonitoring(appCfg.DisconnectFailedCount, 10*time.Second)

	return ctx, nil
}

func (ctx *TlsContext) SetSiteStatus(index int, active bool) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if index >= 0 && index < len(ctx.sites) {
		ctx.sites[index].Active = active
	}
}

func (ctx *TlsContext) IncrementError(index int) {
	// ctx.sessions[index].mu.Lock()
	// ctx.sessions[index].errors++
	// ctx.sessions[index].mu.Unlock()
	ctx.sessions[index].errorCount.Add(int32(1))
}

func (ctx *TlsContext) ClearError(index int) {
	// ctx.sessions[index].mu.Lock()
	// ctx.sessions[index].errors = 0
	// ctx.sessions[index].mu.Unlock()
	ctx.sessions[index].errorCount.Store(0)
}

func (ctx *TlsContext) DisconnectSession(index int) {
	//ctx.sessions[index].mu.Lock()
	//defer ctx.sessions[index].mu.Unlock()

	if ctx.sessions[index].connected.CompareAndSwap(true, false) {
		//ctx.sessions[index].connected = false
		ctx.sessions[index].tlsConn.Close()
		ctx.sessions[index].errorCount.Store(0)
	}
}

// StartMonitoring checks session error counts at a fixed interval
func (ctx *TlsContext) StartMonitoring(threshold int, interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)

			for i, session := range ctx.sessions {
				if session.errorCount.Load() > int32(threshold) {
					log.Printf("%s monitor thread threshold reached current: %d threshold: %d",
						session.name, session.errorCount.Load(), threshold)
					ctx.DisconnectSession(i)
				}
			}
		}
	}()
}

// func (ctx *TlsContext) StartMonitoring(threshold int, interval time.Duration) {
// 	go func() {
// 		for {
// 			time.Sleep(interval)
// 			ctx.mu.Lock()
// 			for i, session := range ctx.sessions {
// 				session.mu.Lock()
// 				if session.errors > threshold {
// 					session.mu.Unlock()
// 					log.Printf("%s monitor thread threshold reached current: %d threshold: %d", session.name, session.errors, threshold)
// 					ctx.DisconnectSession(i)
// 				} else {
// 					session.mu.Unlock()
// 				}
// 			}
// 			ctx.mu.Unlock()
// 		}
// 	}()
// }

// handleConnection handles reading and writing for a TLS session.
func (s *TlsSession) handleConnection(ctx *TlsContext) {
	readBuffer := make([]byte, PBM_DATA_BUFFER)
	tmpBuffer := make([]byte, PBM_DATA_BUFFER)
	zeroSlice := make([]byte, len(readBuffer)) // Create a zeroed slice of the same length
	// MRG 8/13/24 handle connection then the 'read' data to ensure both are in synched
	tranFoundState := NoData
	outputLen := 0      // Current number of valid bytes in output
	expectedMsgLen := 0 // if ASCII len

	go func() {
		for {
			if s.IsConnected() {
				log.Printf("%s reading... status: %s", s.name, tranFoundState)
				copy(readBuffer, zeroSlice) // Copy the zeroed slice into the buffer
				bytes, err := s.tlsConn.Read(readBuffer)
				if err != nil || bytes <= 0 {
					s.activeClaims.Store(0)
					// MRG 8.21.24 let the monitor routine disconnect after error count
					ctx.DisconnectSession(s.chnl)
					log.Printf("%s Read failed: %s", s.name, err)
					time.Sleep(1 * time.Second)
					continue
				}
				if s.appConfig.DebugEnabled {
					log.Printf("%s Rcvd %d bytes data: %s", s.name, bytes, readBuffer)	
					log.Printf("%s Rcvd %d bytes data: %v", s.name, bytes, readBuffer)	
				}				
				log.Printf("%s Rcvd %d bytes", s.name, bytes)
				retVal, state, err := FindFullTransaction(readBuffer, bytes, &tmpBuffer, &outputLen, tranFoundState, &expectedMsgLen, s.appConfig)
				tranFoundState = state
				if err != nil {
					log.Printf("%s FindFullTransaction failed err: %s status: %s", s.name, err, state)
					s.readCh1 <- Response{nil, err, state}
					tranFoundState = NoData
					outputLen = 0
					expectedMsgLen = 0
					copy(tmpBuffer, zeroSlice) // Copy the zeroed slice into the buffer
				} else {
					if retVal && state == TransactionFound {
						s.activeClaims.Add(-1)
						// Create a new slice with the received data
						dataToSend := make([]byte, outputLen)
						copy(dataToSend, tmpBuffer[:outputLen])
						//s.readCh <- dataToSend // Send the new slice
						s.readCh1 <- Response{dataToSend, nil, state}
						tranFoundState = NoData
						outputLen = 0
						expectedMsgLen = 0
						copy(tmpBuffer, zeroSlice) // Copy the zeroed slice into the buffer
					} else {
						log.Printf("%s Rcvd outputLen: %d status: %s Read again", s.name, outputLen, state)
					}
				}
			} else {
				// Check if the site is active
				siteIndex := s.chnl % len(ctx.sites)
				if !ctx.sites[siteIndex].Active {
					time.Sleep(1 * time.Second) // Wait before retrying
					continue
				}

				if err := s.reconnect(true); err != nil {
					log.Printf("%s Reconnection failed: %s", s.name, err)
					time.Sleep(5 * time.Second)
					continue
				}
				log.Printf("%s Pausing to ensure LB is connected to vendor", s.name)
				if(s.appConfig.WaitAfterConnectSeconds>0){
					time.Sleep(time.Duration(s.appConfig.WaitAfterConnectSeconds) * time.Second)
				}				
				s.setConnected(true)
			}
		}
	}()

	for {
		select {
		case data := <-s.writeCh:
			if s.IsConnected() && s.tlsConn != nil {
				total := 0
				for total < len(data) {
					n, err := s.tlsConn.Write(data[total:])
					if err != nil {
						log.Printf("%s Write failed: %s", s.name, err)
						s.setConnected(false)
					}
					total += n
				}
				log.Printf("%s Snd %d bytes", s.name, total)
				s.activeClaims.Add(1)
			} else {
				log.Printf("%s Write failed: connection is nil/not connected", s.name)
			}

		case <-s.closeCh:
			if s.tlsConn != nil {
				log.Printf("%s closing connection...", s.name)
				s.tlsConn.Close()
			}
		}
	}
}

// Implement the String() method for the Status type
func (s Status) String() string {
	switch s {
	case NoData:
		return "NoData"
	case MoreDataPending:
		return "MoreDataPending"
	case TransactionFound:
		return "TransactionFound"
	case ParseError:
		return "ParseError"
	default:
		return "Unknown"
	}
}

func FindFullTransactionUseASCIILen(input []byte, inputLen int, output *[]byte, outputLen *int, state Status, expectedMsgLen *int, appCfg Config) (bool, Status, error) {
	headerLen := appCfg.MessageLenWidth
	headerOffset := appCfg.MessageLenOffset
	tmpLen := *outputLen + inputLen

	if appCfg.DebugEnabled {
		log.Printf("FindFullTransactionUseASCIILen - expected: %d, outputLen: %d, headerLen: %d, headerOffset: %d, tmpLen: %d", *expectedMsgLen, *outputLen, headerLen, headerOffset, tmpLen)
	}

	// Append the input data to the output buffer
	if tmpLen > cap(*output) {
		return false, ParseError, errors.New("output buffer capacity exceeded")
	}
	copy((*output)[*outputLen:], input[:inputLen])
	*outputLen += inputLen

	// Step 1: Ensure the header is fully available
	if *outputLen < headerOffset+headerLen {
		if appCfg.DebugEnabled {
			log.Printf("Not enough data for header - outputLen: %d, required: %d", *outputLen, headerOffset+headerLen)
		}
		return false, MoreDataPending, nil
	}

	// Step 2: Parse header to determine expected message length
	if *expectedMsgLen == 0 {
		asciiHeader := (*output)[headerOffset : headerOffset+headerLen]
		expectedLen, err := strconv.Atoi(strings.TrimSpace(string(asciiHeader)))
		if err != nil || expectedLen <= 0 {
			return false, ParseError, errors.New("invalid ASCII header length")
		}
		if appCfg.MessageLenType == 0 {
			*expectedMsgLen = expectedLen //  cvs case includes full buffer
		} else if appCfg.MessageLenType == 1 {
			*expectedMsgLen = expectedLen + headerLen //  optumrxsolutions excludes header so need to add to incoming
		}

		if appCfg.DebugEnabled {
			log.Printf("Parsed header: expectedMsgLen = %d outputLen: %d", *expectedMsgLen, *outputLen)
		}
	}

	// Step 3: Check if full message has been received
	if *outputLen >= *expectedMsgLen {
		if *outputLen > *expectedMsgLen {
			return false, ParseError, errors.New("extra bytes detected beyond expected length")
		}
		return true, TransactionFound, nil
	}

	// Step 4: Wait for more data
	remaining := *expectedMsgLen - *outputLen
	if remaining > 0 {
		if appCfg.DebugEnabled {
			log.Printf("Waiting for more data - remaining: %d, outputLen: %d, expectedMsgLen: %d", remaining, *outputLen, *expectedMsgLen)
		}
		return false, MoreDataPending, nil
	}

	return false, ParseError, errors.New("unexpected condition encountered")
}

// FindFullTransaction processes input bytes and updates the output with complete transactions.
func FindFullTransaction(input []byte, inputLen int, output *[]byte, outputLen *int, state Status, expectedMsgLen *int, appCfg Config) (bool, Status, error) {
	// Ensure the input length is valid
	var tranFound bool = false
	var tranStatus Status = MoreDataPending
	var err error

	if appCfg.DebugEnabled {
		log.Printf("FindFullTransaction endofchar: %v  (Hex): %x", appCfg.EndOfRecordChar, input[:inputLen])
	}

	if inputLen < 0 || inputLen > len(input) {
		return false, ParseError, errors.New("invalid input length")
	}

	// Calculate how many bytes we can safely append
	availableSpace := PBM_DATA_BUFFER - *outputLen
	if availableSpace <= 0 {
		return false, ParseError, errors.New("output buffer overflow")
	}

	if appCfg.EndOfRecordChar == 0x00 { // TODO: write code to find end of transaction using ASCII Len
		tranFound, tranStatus, err = FindFullTransactionUseASCIILen(input, inputLen, output, outputLen, state, expectedMsgLen, appCfg)
		return tranFound, tranStatus, err
	} else {
		// Determine how much input we can append
		bytesToAppend := inputLen
		if bytesToAppend > availableSpace {
			bytesToAppend = availableSpace
		}
		// Check for ETX (0x03) in the input data
		if idx := bytes.IndexByte(input[:bytesToAppend], appCfg.EndOfRecordChar); idx != -1 {
			// Found ETX, append up to and including the ETX
			copy((*output)[*outputLen:], input[:idx+1]) // Copy the valid portion to output
			*outputLen += idx + 1                       // Update the output length
			return true, TransactionFound, nil
		}
		// No ETX found, append the input data to output
		copy((*output)[*outputLen:], input[:bytesToAppend]) // Copy to output
		*outputLen += bytesToAppend                         // Update the output length
	}

	return false, MoreDataPending, nil
}

func (s *TlsSession) reconnect(splitHandshake bool) error {
	log.Printf("%s connect connecting to '%s' Pbm Certificate Insecure Skip Verify: %t splitHandshake: %t", s.name, s.address, s.appConfig.PbmInsecureSkipVerify, splitHandshake)
		start := time.Now() // Capture the start time
	if splitHandshake { // split call using tcp then tls - in order to configure keep-alive
		// create dialer with keep-alive and connect time-out
		timeout := 5 * time.Second
		keepAliveInterval := 5 * time.Minute
		dialer := &net.Dialer{
			Timeout:   timeout,
			KeepAlive: keepAliveInterval,
		}
		tcpConn, err := dialer.Dial("tcp", s.address)
		if err != nil {
			return err
		}
		// Wrap the TCP connection in a TLS connection
		conn := tls.Client(tcpConn, s.tlsConfig)
		// Perform the TLS handshake using a time out
		conn.SetReadDeadline(time.Now().Add(timeout))
		err = conn.Handshake()
		if err != nil {
			log.Printf("%s connect connecting to '%s' handshake failed err: %v", s.name, s.address, err)
			tcpConn.Close()
			return err
		}
		remoteAddr := conn.RemoteAddr().(*net.TCPAddr)
		log.Printf("%s connect connecting to '%s'(%s:%d)  handshake success", s.name, s.address,remoteAddr.IP,remoteAddr.Port)
		// After a successful handshake, set the read deadline to "never"
		conn.SetReadDeadline(time.Time{})
		s.mu.Lock()
		s.tlsConn = conn
		s.mu.Unlock()

	} else {
		conn, err := tls.Dial("tcp", s.address, s.tlsConfig)
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.tlsConn = conn
		s.mu.Unlock()
	}
	elapsed := time.Since(start) // Calculate elapsed time
	log.Printf("%s connect ok tls handshake duration: %d ms url: %s", s.name,elapsed.Milliseconds(), s.address)
	return nil
}

// setConnected sets the connection status of the session.
func (s *TlsSession) setConnected(status bool) {
	//s.mu.Lock()
	//defer s.mu.Unlock()
	s.connected.Store(status)
}

// IsConnected returns whether the session is connected.
func (s *TlsSession) IsConnected() bool {
	//s.mu.Lock()
	//defer s.mu.Unlock()
	return s.connected.Load()
}

func (s *TlsSession) ProcessResponseWorker() {

	for {

		select {
		case response := <-s.readCh1:
			log.Printf("%s %d bytes received status: %s err: %v", s.name, len(response.data), response.status, response.err)
			if response.status != ParseError {
				responseHeader := GetHeader(response.data, s.appConfig)
				if len(responseHeader) > 0 {
					// Load and delete the transaction ID from responsePbmHeader
					tid, ok := responsePbmHeader.LoadAndDelete(responseHeader)
					if !ok {
						log.Printf("%s Transaction ID not found for request header: %s", s.name, responseHeader)
						continue
					}
					tidStr, ok := tid.(string)
					if !ok {
						log.Printf("%s Invalid transaction ID type", s.name)
						continue
					}

					// Load and delete the response channel
					ch, ok := responseChans.LoadAndDelete(tidStr)
					if !ok {
						log.Printf("%s Response channel not found for transaction ID: %s", s.name, tidStr)
						continue
					}

					chTyped, ok := ch.(chan Response)
					if !ok {
						log.Printf("%s Invalid response channel type", s.name)
						continue
					}

					// Send response safely (avoid deadlock)
					select {
					case chTyped <- response:
						log.Printf("%s response sent to waiting goroutine for tid: %s", s.name, tidStr)
					default:
						log.Printf("%s No receiver available, dropping response", s.name)
					}
				}
			} else {
				//return nil, errors.New("Parse error")
			}
		}

	}

}

func GetHeader(response []byte, appCfg Config) string {

	var responseHeader []byte
	if len(response) > appCfg.HeaderCheckOffset+appCfg.HeaderCheckLen {
		responseHeader = make([]byte, appCfg.HeaderCheckLen)
		copy(responseHeader, response[appCfg.HeaderCheckOffset:appCfg.HeaderCheckOffset+appCfg.HeaderCheckLen])
		if appCfg.DebugEnabled {
			log.Printf("Response header: %s offset: %d len: %d ", string(responseHeader), appCfg.HeaderCheckOffset, appCfg.HeaderCheckLen)
		}
	}
	return string(responseHeader)
}

// Write sends data through a connection.
func (s *TlsSession) Write(index int, data []byte) error {
	log.Printf("%s Snding %d bytes", s.name, len(data))
	//session := s
	s.writeCh <- data
	return nil
}

// ##########################################################
// #################### CONTEXT FUNCTIONS ###################
// ##########################################################

// func (ctx *TlsContext) FindLeastBusyChnl() (*TlsSession, int, error) {
// 	n := len(ctx.sessions)
// 	if n == 0 {
// 		return nil, -1, errors.New("no sessions available")
// 	}

// 	start := int(atomic.LoadUint32(&ctx.lastUsed)) // Read last used index atomically
// 	for i := 0; i < n; i++ {
// 		index := (start + i) % n
// 		if ctx.sessions[index].IsConnected() {
// 			atomic.StoreUint32(&ctx.lastUsed, uint32(index+1)) // Update safely
// 			return ctx.sessions[index], index, nil
// 		}
// 	}

//		return nil, -1, errors.New("no available connections")
//	}
func (ctx *TlsContext) FindLeastBusyChnl() (*TlsSession, int, error) {
	var bestSession *TlsSession = nil
	minClaims := int32(1<<31 - 1) // Set to max int32 value

	for _, s := range ctx.sessions {
		if s.paused.Load() || !s.connected.Load() {
			if s.appConfig.DebugEnabled {
				log.Printf("%s paused or not connected - skipping chnl ", s.name)
			}
			continue // Skip paused or disconnected sessions
		}
		outstanding := s.activeClaims.Load()
		if s.appConfig.DebugEnabled {
			log.Printf("%s active claims chhl: %d activeClaims: %d ", s.name,s.chnl,s.activeClaims)
		}

		if outstanding < minClaims {
			minClaims = outstanding
			bestSession = s
		}
	}
	if bestSession != nil {
		return bestSession, bestSession.chnl, nil
	} else {
		return nil, -1, errors.New("no sessions available")
	}
}

func (ctx *TlsContext) FindConnection() (*TlsSession, int, error) {
	n := len(ctx.sessions)
	if n == 0 {
		return nil, -1, errors.New("no sessions available")
	}

	start := int(atomic.LoadUint32(&ctx.lastUsed)) // Read last used index atomically
	for i := 0; i < n; i++ {
		index := (start + i) % n
		if ctx.sessions[index].IsConnected() {
			atomic.StoreUint32(&ctx.lastUsed, uint32(index+1)) // Update safely
			return ctx.sessions[index], index, nil
		}
	}

	return nil, -1, errors.New("no available connections")
}

// ReleaseConnection releases a connection, making it available again.
func (ctx *TlsContext) ReleaseConnection(index int) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.bitmap[index] = false
}

// Write sends data through a connection.
func (ctx *TlsContext) Write(index int, data []byte) error {
	log.Printf("Writing %d bytes on chnl: %d", len(data), index)
	session := ctx.sessions[index]
	session.writeCh <- data
	return nil
}

// Close closes all TLS sessions.
func (ctx *TlsContext) Close() {
	log.Printf("TlsContext Close running...")
	for _, session := range ctx.sessions {
		log.Printf("sending signal to chnl %d", session.chnl)
		session.closeCh <- true
	}
}
func (ctx *TlsContext) GetConnectionCount() int {
	ctx.mu.Lock()         // Lock the mutex to ensure thread safety
	defer ctx.mu.Unlock() // Unlock the mutex after the function is done

	count := 0
	for _, session := range ctx.sessions {
		//session.mu.Lock() // Lock the session mutex to ensure thread safety for the connected status
		if session.connected.Load() {
			count++
		}
		//session.mu.Unlock() // Unlock the session mutex after checking the status
	}

	return count
}
