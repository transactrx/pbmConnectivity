package asynchflow

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestStatus(t *testing.T) {
	tests := []struct {
		status   Status
		expected int
	}{
		{NoData, 0},
		{MoreDataPending, 1},
		{TransactionFound, 2},
		{ParseError, 3},
	}

	for _, tt := range tests {
		if int(tt.status) != tt.expected {
			t.Errorf("Status %v = %d, want %d", tt.status, int(tt.status), tt.expected)
		}
	}
}

func TestResponse(t *testing.T) {
	data := []byte("test data")
	err := fmt.Errorf("test error")
	
	resp := Response{
		data:   data,
		err:    err,
		status: TransactionFound,
	}
	
	if !bytes.Equal(resp.data, data) {
		t.Errorf("Response.data = %v, want %v", resp.data, data)
	}
	if resp.err.Error() != "test error" {
		t.Errorf("Response.err = %v, want %v", resp.err, err)
	}
	if resp.status != TransactionFound {
		t.Errorf("Response.status = %v, want %v", resp.status, TransactionFound)
	}
}

func TestSite(t *testing.T) {
	site := Site{
		URL:    "example.com:8080",
		Active: true,
	}
	
	if site.URL != "example.com:8080" {
		t.Errorf("Site.URL = %v, want %v", site.URL, "example.com:8080")
	}
	if !site.Active {
		t.Errorf("Site.Active = %v, want %v", site.Active, true)
	}
}

func TestTlsSessionInitialization(t *testing.T) {
	session := &TlsSession{
		name:    "test-session",
		address: "localhost:8080",
	}
	
	if session.name != "test-session" {
		t.Errorf("TlsSession.name = %v, want %v", session.name, "test-session")
	}
	if session.address != "localhost:8080" {
		t.Errorf("TlsSession.address = %v, want %v", session.address, "localhost:8080")
	}
	
	// Test atomic boolean defaults to false
	if session.connected.Load() {
		t.Error("TlsSession.connected should default to false")
	}
}

func TestTlsSessionConnectedState(t *testing.T) {
	session := &TlsSession{}
	
	// Test initial state
	if session.connected.Load() {
		t.Error("Initial connected state should be false")
	}
	
	// Test setting connected to true
	session.connected.Store(true)
	if !session.connected.Load() {
		t.Error("Connected state should be true after Store(true)")
	}
	
	// Test setting connected to false
	session.connected.Store(false)
	if session.connected.Load() {
		t.Error("Connected state should be false after Store(false)")
	}
}

func TestTlsSessionChannelInitialization(t *testing.T) {
	session := &TlsSession{
		readCh:  make(chan []byte, 10),
		readCh1: make(chan Response, 10),
		writeCh: make(chan []byte, 10),
		closeCh: make(chan bool, 1),
	}
	
	// Test that channels are initialized and can be used
	select {
	case session.readCh <- []byte("test"):
	default:
		t.Error("readCh should accept data")
	}
	
	select {
	case session.readCh1 <- Response{data: []byte("test"), status: NoData}:
	default:
		t.Error("readCh1 should accept response")
	}
	
	select {
	case session.writeCh <- []byte("test"):
	default:
		t.Error("writeCh should accept data")
	}
	
	select {
	case session.closeCh <- true:
	default:
		t.Error("closeCh should accept boolean")
	}
}

func TestTlsSessionConcurrentAccess(t *testing.T) {
	session := &TlsSession{}
	
	const numGoroutines = 100
	done := make(chan bool, numGoroutines)
	
	// Test concurrent access to connected state
	for i := 0; i < numGoroutines; i++ {
		go func(val bool) {
			session.connected.Store(val)
			_ = session.connected.Load()
			done <- true
		}(i%2 == 0)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Goroutine timed out")
		}
	}
	
	// Final state should be either true or false
	finalState := session.connected.Load()
	if finalState != true && finalState != false {
		t.Error("Connected state should be boolean")
	}
}

func TestTlsSessionMutexFunctionality(t *testing.T) {
	session := &TlsSession{}
	
	// Test that mutex can be locked and unlocked
	session.mu.Lock()
	session.mu.Unlock()
	
	// Test concurrent access with mutex protection
	counter := 0
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			session.mu.Lock()
			localCounter := counter
			time.Sleep(time.Millisecond) // Simulate work
			counter = localCounter + 1
			session.mu.Unlock()
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	if counter != 10 {
		t.Errorf("Counter = %d, want 10 (mutex should prevent race conditions)", counter)
	}
}

func TestConstants(t *testing.T) {
	if CLAIM_QUEUE_LEN != 100 {
		t.Errorf("CLAIM_QUEUE_LEN = %d, want 100", CLAIM_QUEUE_LEN)
	}
}

func TestResponseChannelUsage(t *testing.T) {
	// Test that Response struct works properly with channels
	ch := make(chan Response, 1)
	
	testResp := Response{
		data:   []byte("channel test"),
		err:    nil,
		status: TransactionFound,
	}
	
	// Send response through channel
	select {
	case ch <- testResp:
	default:
		t.Error("Should be able to send Response through channel")
	}
	
	// Receive response from channel
	select {
	case received := <-ch:
		if !bytes.Equal(received.data, testResp.data) {
			t.Error("Received data doesn't match sent data")
		}
		if received.status != testResp.status {
			t.Error("Received status doesn't match sent status")
		}
		if received.err != testResp.err {
			t.Error("Received error doesn't match sent error")
		}
	default:
		t.Error("Should be able to receive Response from channel")
	}
}

func TestTlsSessionFieldAccess(t *testing.T) {
	session := &TlsSession{
		name:    "field-test",
		address: "test.example.com:443",
	}
	
	// Test that fields are accessible
	if session.name != "field-test" {
		t.Errorf("name field = %v, want %v", session.name, "field-test")
	}
	
	if session.address != "test.example.com:443" {
		t.Errorf("address field = %v, want %v", session.address, "test.example.com:443")
	}
	
	// Test that tlsConn can be set to nil without issues
	session.tlsConn = nil
	if session.tlsConn != nil {
		t.Error("tlsConn should be nil")
	}
}

// Test that demonstrates the usage pattern
func TestTlsSessionUsagePattern(t *testing.T) {
	session := &TlsSession{
		name:    "usage-test",
		address: "localhost:8080",
		readCh:  make(chan []byte, 1),
		readCh1: make(chan Response, 1),
		writeCh: make(chan []byte, 1),
		closeCh: make(chan bool, 1),
	}
	
	// Simulate connecting
	session.connected.Store(true)
	
	// Simulate writing data
	testData := []byte("test message")
	select {
	case session.writeCh <- testData:
	case <-time.After(100 * time.Millisecond):
		t.Error("Write channel should accept data quickly")
	}
	
	// Simulate reading response
	testResponse := Response{
		data:   []byte("response message"),
		status: TransactionFound,
		err:    nil,
	}
	select {
	case session.readCh1 <- testResponse:
	case <-time.After(100 * time.Millisecond):
		t.Error("Response channel should accept response quickly")
	}
	
	// Simulate closing
	select {
	case session.closeCh <- true:
	case <-time.After(100 * time.Millisecond):
		t.Error("Close channel should accept signal quickly")
	}
	
	// Verify state
	if !session.connected.Load() {
		t.Error("Session should still be marked as connected")
	}
}