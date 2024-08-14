package tlspersistedsynch

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"
)

type Site struct {
	URL    string
	Active bool
}

// TlsSession represents a single TLS session.
type TlsSession struct {
	address   string
	conn      *tls.Conn
	readCh    chan []byte
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
	sessions []*TlsSession
	bitmap   []bool
	mu       sync.Mutex
	lastUsed int
	sites    []*Site
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
			address:   addr,
			readCh:    make(chan []byte),
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

	// Start monitoring with a threshold of 5 errors and a check interval of 10 seconds
	ctx.StartMonitoring(5, 10*time.Second)

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
		ctx.sessions[index].conn.Close()
		ctx.sessions[index].errors = 0     // Reset error count
		close(ctx.sessions[index].closeCh) // Signal close
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

	// MRG 8/13/24 handle connection then the 'read' data to ensure both are in synched

	go func() {
		for {
			if s.IsConnected() {
				log.Printf("TlsSession[%d] reading...", s.chnl)
				bytes, err := s.conn.Read(readBuffer)
				if err != nil {
					s.setConnected(false)
					s.mu.Lock()
					s.conn = nil
					s.mu.Unlock()
					log.Printf("TlsSession[%d] Read failed: %s", s.chnl, err)
					continue
				}
				log.Printf("TlsSession[%d] Rcvd %d bytes", s.chnl, bytes)
				s.readCh <- readBuffer[:bytes]
			} else {
				// Check if the site is active
				siteIndex := s.chnl % len(ctx.sites)
				if !ctx.sites[siteIndex].Active {
					time.Sleep(1 * time.Second) // Wait before retrying
					continue
				}

				if err := s.reconnect(); err != nil {
					log.Printf("TlsSession[%d] Reconnection failed: %s", s.chnl, err)
					time.Sleep(5 * time.Second)
					continue
				}
				log.Printf("TlsSession[%d] Pausing to ensure LB is connected to vendor", s.chnl)
				time.Sleep(6 * time.Second)
				s.setConnected(true)
			}
		}
	}()

	for {

		select {
		case data := <-s.writeCh:
			if s.IsConnected() && s.conn != nil {
				bytes, err := s.conn.Write(data)
				if err != nil {
					log.Printf("TlsSession[%d] Write failed: %s", s.chnl, err)
					s.setConnected(false)
					continue
				} else {
					log.Printf("TlsSession[%d] Snd %d bytes", s.chnl, bytes)
				}
			} else {
				log.Printf("TlsSession[%d] Write failed connection object is nil", s.chnl)
			}

		case <-s.closeCh:
			log.Printf("TlsSession[%d] before closing connection", s.chnl)
			//if s.conn != nil {
			log.Printf("TlsSession[%d] closing connection", s.chnl)
			s.conn.Close()
			//}

			return
		default:
			// Optional: Add a short sleep to prevent busy waiting in the select loop
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// reconnect attempts to reconnect the TLS session.
func (s *TlsSession) reconnect() error {
	log.Printf("TlsSession[%d] connect connecting to '%s' Pbm Certificate Insecure Skip Verify: %t", s.chnl, s.address, s.appConfig.PbmInsecureSkipVerify)
	conn, err := tls.Dial("tcp", s.address, s.tlsConfig)
	if err != nil {
		return err
	}
	log.Printf("TlsSession[%d] connect connected to '%s'", s.chnl, s.address)
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	//s.setConnected(true)
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

func (session *TlsSession) Read(appCtx context.Context, index int) ([]byte, error) {
	//session := s
	//session.mu.Lock()
	//defer session.mu.Unlock()

	select {
	case data := <-session.readCh:
		log.Printf("TlsSession[%d] %d bytes received", index, len(data))
		//ctx.ClearError(index)
		return data, nil
	case <-appCtx.Done():
		//ctx.IncrementError(index)
		return nil, appCtx.Err() // Return the context error, typically context.DeadlineExceeded
	}
}

// Write sends data through a connection.
func (session *TlsSession) Write(index int, data []byte) error {
	log.Printf("TlsSession[%d] Snding %d bytes", index, len(data))
	//session := s
	session.writeCh <- data
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

func (ctx *TlsContext) Read(appCtx context.Context, index int) ([]byte, error) {
	session := ctx.sessions[index]
	//session.mu.Lock()
	//defer session.mu.Unlock()

	select {
	case data := <-session.readCh:
		log.Printf("read some data... data len: %d", len(data))
		ctx.ClearError(index)
		return data, nil
	case <-appCtx.Done():
		ctx.IncrementError(index)
		return nil, appCtx.Err() // Return the context error, typically context.DeadlineExceeded
	}
}

// Close closes all TLS sessions.
func (ctx *TlsContext) Close() {
	log.Printf("TlsContext Close running...")
	for _, session := range ctx.sessions {
		log.Printf("sending signal to chnl %d", session.chnl)
		session.closeCh <- true
	}
}
