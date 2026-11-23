package tlspersistedsynch

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestRecycleLimit(t *testing.T) {
	tests := []struct {
		name  string
		max   time.Duration
		chnl  int
		count int
		want  time.Duration
	}{
		{"single channel gets full max", 50 * time.Minute, 0, 1, 50 * time.Minute},
		{"chnl 0 of 5", 50 * time.Minute, 0, 5, 50 * time.Minute},
		{"chnl 1 of 5", 50 * time.Minute, 1, 5, 45 * time.Minute},
		{"chnl 4 of 5", 50 * time.Minute, 4, 5, 30 * time.Minute},
		{"chnl 1 of 2", 60 * time.Minute, 1, 2, 45 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recycleLimit(tt.max, tt.chnl, tt.count); got != tt.want {
				t.Errorf("recycleLimit(%v,%d,%d) = %v, want %v", tt.max, tt.chnl, tt.count, got, tt.want)
			}
		})
	}
}

// startTLSServer accepts TLS connections and counts them; it never writes.
func startTLSServer(t *testing.T) (addr string, accepted *atomic.Int32) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	accepted = &atomic.Int32{}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			go func() {
				buf := make([]byte, 1024)
				for {
					if _, err := c.Read(buf); err != nil { // drives the handshake, holds the conn open
						c.Close()
						return
					}
				}
			}()
		}
	}()
	return ln.Addr().String(), accepted
}

func waitFor(t *testing.T, d time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return cond()
}

// TestRecycleOldSessions runs real sessions against a local TLS server and
// checks that only idle sessions past their limit are recycled and reconnect.
func TestRecycleOldSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: needs ~10s for reconnect")
	}
	addr, accepted := startTLSServer(t)
	host, port, _ := net.SplitHostPort(addr)

	Cfg = Config{
		PbmUrl:                []string{host},
		PbmPort:               port,
		PbmInsecureSkipVerify: true,
		PbmOutboundChnls:      2,
		PbmActiveSites:        []bool{true},
		PbmQueueTimeOut:       "1",
		DisconnectFailedCount: 10,
		EndOfRecordChar:       0x03,
	}
	ctx, err := NewTlsContext(Cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, s := range ctx.sessions {
			ctx.DisconnectSession(s.chnl)
		}
	})

	// Both sessions connect (6s LB pause before they're marked connected).
	bothUp := func() bool { return ctx.sessions[0].IsConnected() && ctx.sessions[1].IsConnected() }
	if !waitFor(t, 15*time.Second, bothUp) {
		t.Fatalf("sessions did not connect")
	}
	if got := accepted.Load(); got != 2 {
		t.Fatalf("accepted = %d, want 2", got)
	}

	// chnl 1 is busy (claimed by a request) and must not be recycled.
	ctx.mu.Lock()
	ctx.bitmap[1] = true
	ctx.mu.Unlock()

	ctx.recycleOldSessions(time.Nanosecond) // every session is past its limit

	if ctx.sessions[0].IsConnected() {
		t.Errorf("idle chnl 0 should have been disconnected for recycling")
	}
	if !ctx.sessions[1].IsConnected() {
		t.Errorf("busy chnl 1 must not be recycled")
	}
	ctx.mu.Lock()
	reserved0, reserved1 := ctx.bitmap[0], ctx.bitmap[1]
	ctx.mu.Unlock()
	if reserved0 {
		t.Errorf("chnl 0 should be released after recycling")
	}
	if !reserved1 {
		t.Errorf("chnl 1 reservation must be untouched")
	}

	// chnl 0 reconnects through the normal read-loop path with a fresh age.
	if !waitFor(t, 15*time.Second, func() bool { return ctx.sessions[0].IsConnected() }) {
		t.Fatalf("chnl 0 did not reconnect")
	}
	if got := accepted.Load(); got != 3 {
		t.Errorf("accepted = %d, want 3 (one reconnect)", got)
	}
	if age, ok := ctx.sessions[0].connectionAge(); !ok || age > 5*time.Second {
		t.Errorf("chnl 0 age after reconnect = %v (connected=%v), want fresh", age, ok)
	}
}
