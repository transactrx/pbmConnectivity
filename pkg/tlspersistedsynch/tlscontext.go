package tlspersistedsynch

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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

// TlsSession represents a single TLS session.
type TlsSession struct {
	name      string 
	address   string
	tlsConn   *tls.Conn
	readCh    chan []byte
	readCh1   chan Response
	writeCh   chan []byte
	closeCh   chan bool
	connected bool
	mu        sync.Mutex
	tlsConfig *tls.Config
	appConfig Config
	chnl      int
	errors    int
}

// TlsContext manages multiple TLS sessions.
type TlsContext struct {
	sessions  []*TlsSession
	bitmap    []bool
	mu        sync.Mutex
	lastUsed  int
	sites     []*Site
	tcpDialer *net.Dialer
}
func isIPAddress(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil
}
func createSessionName(i int, siteURL string) string {
	// Get the local machine's hostname
	hostName, _ := os.Hostname()

	// Check if the hostname is an IP address
	var hostIdentifier string
	if isIPAddress(hostName) {
		// Extract last octet of the hostname IP
		hostParts := strings.Split(hostName, ".")
		hostIdentifier = hostParts[len(hostParts)-1]
	} else {
		// Use full hostname if it's not an IP address
		hostIdentifier = hostName
	}

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
	tmpName := fmt.Sprintf("tls[ch:%d;f:%s;t:%s]", i, hostIdentifier, targetIdentifier)

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
		activeSite = Cfg.PbmActiveSites[i]
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
			name:  createSessionName(i,site.URL),
			address:   addr,
			readCh:    make(chan []byte),
			readCh1:   make(chan Response),
			writeCh:   make(chan []byte),
			closeCh:   make(chan bool),
			connected: false,
			tlsConfig: tlsConfig,
			appConfig: appCfg,
			chnl:      i,
		}
		ctx.sessions[i] = session
		go session.handleConnection(ctx) // Pass ctx to handleConnection
	}

	// Start monitoring with a threshold of 2 errors and a check interval of 10 seconds
	ctx.StartMonitoring(2, 10*time.Second)

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
	ctx.sessions[index].mu.Lock()
	ctx.sessions[index].errors++
	ctx.sessions[index].mu.Unlock()
}

func (ctx *TlsContext) ClearError(index int) {
	ctx.sessions[index].mu.Lock()
	ctx.sessions[index].errors = 0
	ctx.sessions[index].mu.Unlock()
}

func (ctx *TlsContext) DisconnectSession(index int) {
	ctx.sessions[index].mu.Lock()
	defer ctx.sessions[index].mu.Unlock()

	if ctx.sessions[index].connected {
		ctx.sessions[index].connected = false
		ctx.sessions[index].tlsConn.Close()
		ctx.sessions[index].errors = 0 // Reset error count
		//		close(ctx.sessions[index].closeCh) // Signal close
	}
}

func (ctx *TlsContext) StartMonitoring(threshold int, interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			ctx.mu.Lock()
			for i, session := range ctx.sessions {
				session.mu.Lock()
				if session.errors > threshold {
					session.mu.Unlock()
					ctx.DisconnectSession(i)
				} else {
					session.mu.Unlock()
				}
			}
			ctx.mu.Unlock()
		}
	}()
}

// handleConnection handles reading and writing for a TLS session.
func (s *TlsSession) handleConnection(ctx *TlsContext) {
	readBuffer := make([]byte, PBM_DATA_BUFFER)
	tmpBuffer := make([]byte, PBM_DATA_BUFFER)
	zeroSlice := make([]byte, len(readBuffer)) // Create a zeroed slice of the same length
	// MRG 8/13/24 handle connection then the 'read' data to ensure both are in synched
	tranFoundState := NoData
	outputLen := 0 // Current number of valid bytes in output
	

	go func() {
		for {
			if s.IsConnected() {
				log.Printf("%s reading... status: %s",s.name ,tranFoundState)
				copy(readBuffer, zeroSlice) // Copy the zeroed slice into the buffer
				bytes, err := s.tlsConn.Read(readBuffer)
				if err != nil || bytes <= 0 {
					// MRG 8.21.24 let the monitor routine disconnect after error count
					ctx.DisconnectSession(s.chnl)
					log.Printf("%s Read failed: %s", s.name, err)
					time.Sleep(1 * time.Second)
					continue
				}
				log.Printf("%s Rcvd %d bytes", s.name, bytes)
				retVal, state, err := FindFullTransaction(readBuffer, bytes, &tmpBuffer, &outputLen, tranFoundState)
				tranFoundState = state
				if err != nil {
					log.Printf("%s FindFullTransaction failed err: %s status: %s", s.name, err,state)
					s.readCh1 <- Response{nil,err,state}
					tranFoundState = NoData
					outputLen = 0
					copy(tmpBuffer, zeroSlice) // Copy the zeroed slice into the buffer
				} else {
					if retVal && state == TransactionFound {
						// Create a new slice with the received data
						dataToSend := make([]byte, outputLen)
						copy(dataToSend, tmpBuffer[:outputLen])
						//s.readCh <- dataToSend // Send the new slice
						s.readCh1 <- Response{dataToSend, nil, state}
						tranFoundState = NoData
						outputLen = 0
						copy(tmpBuffer, zeroSlice) // Copy the zeroed slice into the buffer
					} else {
						log.Printf("%s Rcvd outputLen: %d status: %s Read again", s.name, outputLen,state)
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
				log.Printf("%s Pausing to ensure LB is connected to vendor", s.chnl)
				time.Sleep(6 * time.Second)
				s.setConnected(true)
			}
		}
	}()

	for {

		select {
		case data := <-s.writeCh:
			if s.IsConnected() && s.tlsConn != nil {
				bytes, err := s.tlsConn.Write(data)
				if err != nil {
					log.Printf("%s Write failed: %s", s.name, err)
					s.setConnected(false)
					continue
				} else {
					log.Printf("%s Snd %d bytes", s.name, bytes)
				}
			} else {
				log.Printf("%s Write failed connection object is nil", s.chnl)
			}

		case <-s.closeCh:

			if s.tlsConn != nil {
				log.Printf("%s closing connection...", s.chnl)
				s.tlsConn.Close()
			} else {
				log.Printf("%s s.conn.close - conn was null", s.chnl)
			}

			return
		default:
			// Optional: Add a short sleep to prevent busy waiting in the select loop
			time.Sleep(100 * time.Millisecond)
		}
	}
}

type Status int

const (
	NoData           Status = iota // Indicates that there is no data yet
	MoreDataPending                // More data is needed
	TransactionFound               // A full transaction has been found
	ParseError                     // Indicates a parsing error
)
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

// FindFullTransaction processes input bytes and updates the output with complete transactions.
func FindFullTransaction(input []byte, inputLen int, output *[]byte, outputLen *int, state Status) (bool, Status, error) {
	// Ensure the input length is valid
	if inputLen < 0 || inputLen > len(input) {
		return false, ParseError, errors.New("invalid input length")
	}

	// Calculate how many bytes we can safely append
	availableSpace := PBM_DATA_BUFFER - *outputLen
	if availableSpace <= 0 {
		return false, ParseError, errors.New("output buffer overflow")
	}

	// Determine how much input we can append
	bytesToAppend := inputLen
	if bytesToAppend > availableSpace {
		bytesToAppend = availableSpace
	}

	// Check for ETX (0x03) in the input data
	if idx := bytes.IndexByte(input[:bytesToAppend], Cfg.EndOfRecordChar); idx != -1 {
		// Found ETX, append up to and including the ETX
		copy((*output)[*outputLen:], input[:idx+1]) // Copy the valid portion to output
		*outputLen += idx + 1                       // Update the output length
		return true, TransactionFound, nil
	}

	// No ETX found, append the input data to output
	copy((*output)[*outputLen:], input[:bytesToAppend]) // Copy to output
	*outputLen += bytesToAppend                         // Update the output length

	return false, MoreDataPending, nil
}

func (s *TlsSession) reconnect(explicitHandshake bool) error {
	log.Printf("%s connect connecting to '%s' Pbm Certificate Insecure Skip Verify: %t splitHandshake: %t", s.name, s.address, s.appConfig.PbmInsecureSkipVerify, explicitHandshake)
	if explicitHandshake { // split call using tcp then tls - in order to configure keep-alive
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
		log.Printf("%s connect connecting to '%s' handshake success", s.name, s.address)
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
	log.Printf("%s connect connected to '%s'", s.name, s.address)
	return nil
}

// setConnected sets the connection status of the session.
func (s *TlsSession) setConnected(status bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = status
}

// IsConnected returns whether the session is connected.
func (s *TlsSession) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected
}

func (s *TlsSession) Read(appCtx context.Context, index int, requestHeader string) ([]byte, error) {

	select {
	case response := <-s.readCh1:
		log.Printf("%s %d bytes received status: %s err: %v", s.name, len(response.data),response.status,response.err)
		if(response.status != ParseError){
			validResponse := IsValidResponse(response.data, requestHeader)
			if !validResponse {
				return nil, errors.New("Mismatch request/response")
			}else{
				return response.data, nil		
			}			
		}else{
			return nil, errors.New("Parse error")
		}
		
	case <-appCtx.Done():
		//ctx.IncrementError(index)
		return nil, appCtx.Err() // Return the context error, typically context.DeadlineExceeded
	}
}

// MRG 9/23/24 compare response header vs request header
// true - valid response
// false -- issue with incoming header (potential swapped responses)
func IsValidResponse(response []byte, requestHeader string) bool {

	//log.Printf("PBM response data(ALL) '%s'", string(response))
	result := false

	if len(response) > Cfg.HeaderCheckOffset+Cfg.HeaderCheckLen {
		if len(requestHeader) > Cfg.HeaderCheckLen {
			requestHeader = requestHeader[:Cfg.HeaderCheckLen] // truncate to 23 characters if longer
		}
		responseHeader := make([]byte, Cfg.HeaderCheckLen)
		copy(responseHeader, response[Cfg.HeaderCheckOffset:Cfg.HeaderCheckOffset+Cfg.HeaderCheckLen])
		// Compare response hdr vs claim header

		reqHdrString := fmt.Sprintf("%-*s", Cfg.HeaderCheckLen, requestHeader)
		if string(responseHeader) == reqHdrString {
			result = true
		} else {
			log.Printf("ValidateResponse failed mismatch FULLresp: '%s' requestHdr: '%s'", string(response), requestHeader)
		}
	}
	return result
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

func (ctx *TlsContext) FindConnection() (*TlsSession, int, error) {
	tmp, _ := strconv.Atoi(Cfg.PbmQueueTimeOut)
	maxTime := time.Duration(tmp)
	waitDuration := maxTime * time.Second
	const retryInterval = 100 * time.Millisecond

	startTime := time.Now()

	for {
		ctx.mu.Lock()
		for i := 0; i < len(ctx.sessions); i++ {
			index := (ctx.lastUsed + i) % len(ctx.sessions)
			if !ctx.bitmap[index] && ctx.sessions[index].IsConnected() {
				ctx.bitmap[index] = true
				ctx.lastUsed = index + 1 // Update the last used index
				ctx.mu.Unlock()
				return ctx.sessions[index], index, nil
			}
		}
		ctx.mu.Unlock()

		elapsed := time.Since(startTime)
		if elapsed > waitDuration {
			log.Printf("TlsContext FindConnection failed to find chnl - timer expired after %v", elapsed)
			return nil, -1, fmt.Errorf("no available connection after waiting for %v seconds", maxTime)
		}

		// Wait before trying again
		time.Sleep(retryInterval)
	}
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
		if session.connected {
			count++
		}
		//session.mu.Unlock() // Unlock the session mutex after checking the status
	}

	return count
}
