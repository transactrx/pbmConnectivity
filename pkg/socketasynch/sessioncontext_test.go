package socketasynch

import (
	"sync"
	"testing"
	"time"

	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

func newTestContext(n int) *SessionContext {
	ctx := &SessionContext{}
	for i := range n {
		s := &SocketSession{name: "test", chnl: i}
		s.connected.Store(true)
		ctx.sessions = append(ctx.sessions, s)
	}
	return ctx
}

// pickN calls FindLeastBusyChnl n times and returns how many claims each chnl got
func pickN(t *testing.T, ctx *SessionContext, n int) map[int]int {
	t.Helper()
	used := map[int]int{}
	for range n {
		_, idx, err := ctx.FindLeastBusyChnl()
		if err != nil {
			t.Fatal(err)
		}
		used[idx]++
	}
	return used
}

func TestFindLeastBusyChnlSpreadsClaims(t *testing.T) {
	used := pickN(t, newTestContext(4), 8)
	for i := range 4 {
		if used[i] != 2 {
			t.Errorf("chnl %d got %d claims, want 2 (%v)", i, used[i], used)
		}
	}
}

func TestFindLeastBusyChnlReusesReleasedChnl(t *testing.T) {
	ctx := newTestContext(3)
	pickN(t, ctx, 3)
	ctx.sessions[1].releaseClaim()
	if _, idx, _ := ctx.FindLeastBusyChnl(); idx != 1 {
		t.Errorf("got chnl %d, want 1", idx)
	}
}

func TestFindLeastBusyChnlSkipsPausedAndDisconnected(t *testing.T) {
	ctx := newTestContext(3)
	ctx.sessions[0].paused.Store(true)
	ctx.sessions[1].connected.Store(false)
	if used := pickN(t, ctx, 3); used[2] != 3 {
		t.Errorf("want all 3 claims on chnl 2, got %v", used)
	}
}

func TestFindLeastBusyChnlNoneAvailable(t *testing.T) {
	ctx := newTestContext(1)
	ctx.sessions[0].connected.Store(false)
	if _, _, err := ctx.FindLeastBusyChnl(); err == nil {
		t.Error("expected error")
	}
}

func TestFindLeastBusyChnlConcurrent(t *testing.T) {
	ctx := newTestContext(4)
	var wg sync.WaitGroup
	for range 400 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx.FindLeastBusyChnl()
		}()
	}
	wg.Wait()
	for _, s := range ctx.sessions {
		if got := s.activeClaims.Load(); got != 100 {
			t.Errorf("chnl %d has %d outstanding, want 100", s.chnl, got)
		}
	}
}

func TestReleaseClaimDoesNotGoNegative(t *testing.T) {
	s := &SocketSession{}
	s.releaseClaim()
	if got := s.activeClaims.Load(); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
	s.activeClaims.Store(2)
	s.releaseClaim()
	if got := s.activeClaims.Load(); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
}

func TestFindLeastBusyChnlRotatesWhenIdle(t *testing.T) {
	ctx := newTestContext(4)
	used := map[int]int{}
	for range 8 {
		s, idx, err := ctx.FindLeastBusyChnl()
		if err != nil {
			t.Fatal(err)
		}
		s.releaseClaim() // claim finished before the next one arrives
		used[idx]++
	}
	for i := range 4 {
		if used[i] != 2 {
			t.Errorf("chnl %d got %d claims, want 2 (%v)", i, used[i], used)
		}
	}
}

// newTestConnect returns a connector whose single chnl accepts writes without a socket
func newTestConnect(timeoutSecs string) (*SocketAsyncConnect, *SocketSession) {
	ctx := newTestContext(1)
	s := ctx.sessions[0]
	s.writeCh = make(chan []byte, 10)
	pc := &SocketAsyncConnect{Ctx: ctx, Cfg: Config{PbmReceiveTimeOut: timeoutSecs, DisconnectFailedCount: 10}}
	return pc, s
}

func testHeader(tid, tracking string) map[string][]string {
	return map[string][]string{"transmissionId": {tid}, "headerValueToCheck": {tracking}}
}

func TestPostReleasesClaimOnTimeout(t *testing.T) {
	pc, s := newTestConnect("1")
	_, _, rc := pc.Post([]byte("claim"), testHeader("tid-timeout", "Q00000001"))
	if rc.Code != pbmlib.ErrorCode.TRX05.Code {
		t.Fatalf("got %s, want TRX05", rc.Code)
	}
	if got := s.activeClaims.Load(); got != 0 {
		t.Errorf("activeClaims after timeout = %d, want 0", got)
	}
}

func TestPostReleasesClaimOnResponse(t *testing.T) {
	pc, s := newTestConnect("5")
	go deliverResponse(t, s, "tid-ok")
	resp, _, rc := pc.Post([]byte("claim"), testHeader("tid-ok", "Q00000002"))
	if rc.Code != pbmlib.ErrorCode.TRX00.Code || string(resp) != "response" {
		t.Fatalf("got %s %q, want TRX00 \"response\"", rc.Code, resp)
	}
	if got := s.activeClaims.Load(); got != 0 {
		t.Errorf("activeClaims after response = %d, want 0", got)
	}
}

// deliverResponse waits for post to register tid, checks the claim is counted, then answers it
func deliverResponse(t *testing.T, s *SocketSession, tid string) {
	for {
		if ch, ok := responseChans.Load(tid); ok {
			if got := s.activeClaims.Load(); got != 1 {
				t.Errorf("activeClaims in flight = %d, want 1", got)
			}
			ch.(chan Response) <- Response{data: []byte("response"), status: TransactionFound}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTimedOutChnlStillGetsClaims(t *testing.T) {
	pc, _ := newTestConnect("1")
	pc.Post([]byte("claim"), testHeader("tid-1", "Q00000003")) // times out
	if _, idx, err := pc.Ctx.FindLeastBusyChnl(); err != nil || idx != 0 {
		t.Errorf("got chnl %d err %v, want chnl 0 available", idx, err)
	}
}
