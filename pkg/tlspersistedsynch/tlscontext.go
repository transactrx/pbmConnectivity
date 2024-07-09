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
}

// TlsContext manages multiple TLS sessions.
type TlsContext struct {
	sessions []*TlsSession
	bitmap   []bool
	mu       sync.Mutex
	lastUsed int
}

// NewTlsContext creates a new TlsContext with predefined sessions.
func NewTlsContext(appCfg Config) (*TlsContext, error) {
	ctx := &TlsContext{
		sessions: make([]*TlsSession, appCfg.PbmOutboundChnls),
		bitmap:   make([]bool, appCfg.PbmOutboundChnls),
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: appCfg.PbmInsecureSkipVerify, // You might want to set this to false in production
		ServerName:         appCfg.PbmUrl,
	}

	for i := 0; i < appCfg.PbmOutboundChnls; i++ {
		addr := appCfg.PbmUrl + ":" + appCfg.PbmPort
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
		go session.handleConnection()
	}

	return ctx, nil
}

// handleConnection handles reading and writing for a TLS session.
func (s *TlsSession) handleConnection() {
	readBuffer := make([]byte, PBM_DATA_BUFFER)

	// Goroutine to handle reading from the connection
	go func() {
		for {
			if s.IsConnected() {
				log.Printf("TlsSession[%d] reading...", s.chnl)
				bytes, err := s.conn.Read(readBuffer)
				if err != nil {
					log.Printf("TlsSession[%d] Read failed: %s", s.chnl, err)
					s.setConnected(false)
					continue
				}
				log.Printf("TlsSession[%d] Rcvd %d bytes", s.chnl, bytes)
				s.readCh <- readBuffer[:bytes]
			} else {
				// If not connected, just yield the CPU to avoid busy waiting
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	for {
		if !s.IsConnected() {
			if err := s.reconnect(); err != nil {
				log.Printf("TlsSession[%d] Reconnection failed: %s", s.chnl, err)
				time.Sleep(5 * time.Second)
				continue
			}
		}

		select {
		case data := <-s.writeCh:

			bytes, err := s.conn.Write(data)
			if err != nil {
				log.Printf("TlsSession[%d] Write failed: %s", s.chnl, err)
				s.setConnected(false)
				continue
			} else {
				log.Printf("TlsSession[%d] Snd %d bytes", s.chnl, bytes)
			}
		case <-s.closeCh:
			return
		default:
			// Optional: Add a short sleep to prevent busy waiting in the select loop
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// reconnect attempts to reconnect the TLS session.
func (s *TlsSession) reconnect() error {
	//s.mu.Lock()
	//defer s.mu.Unlock()	
	log.Printf("TlsSession[%d] connect connecting to '%s' Pbm Certificate Insecure Skip Verify: %t", s.chnl, s.address, s.appConfig.PbmInsecureSkipVerify)
	conn, err := tls.Dial("tcp", s.address, s.tlsConfig)
	if err != nil {
		return err
	}
	log.Printf("TlsSession[%d] connect connected to '%s'", s.chnl, s.address)
	s.conn = conn
	s.setConnected(true)
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

func (ctx *TlsContext) FindConnection() (*TlsSession, int ,error) {

	tmp,_ := strconv.Atoi(Cfg.PbmQueueTimeOut)
	maxTime := time.Duration(tmp) 
	waitDuration := maxTime  * time.Second
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
        //log.Printf("Elapsed time: %v waitDuration: %v", elapsed,waitDuration)

        // If waitDuration seconds have passed, return an error
        if elapsed > waitDuration {
            log.Printf("TlsContext FindConnection failed to find chnl - timer expired after %v", elapsed)
            return nil, -1, fmt.Errorf("no available connection after waiting for %v seconds", maxTime)
        }

		// Wait before trying again
		time.Sleep(retryInterval)
	}
} // FindConnection finds an available connection and marks it as used.

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

	//session.mu.Lock()
	//defer session.mu.Unlock()

	session.writeCh <- data
	return nil
}

func (ctx *TlsContext) Read(appCtx context.Context, index int) ([]byte, error) {
    session := ctx.sessions[index]
    session.mu.Lock()
    defer session.mu.Unlock()

    select {
    case data := <-session.readCh:
        return data, nil
    case <-appCtx.Done():
        return nil, appCtx.Err() // Return the context error, typically context.DeadlineExceeded
    }
}

// // Read receives the response from a connection.
// func (ctx *TlsContext) Read(index int) ([]byte, error) {
// 	session := ctx.sessions[index]
// 	session.mu.Lock()
// 	defer session.mu.Unlock()

// 	select {
// 	case data := <-session.readCh:
// 		return data, nil
// 	}
// }

// Close closes all TLS sessions.
func (ctx *TlsContext) Close() {
	for _, session := range ctx.sessions {
		session.closeCh <- true
	}
}
