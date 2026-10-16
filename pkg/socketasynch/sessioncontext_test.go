package socketasynch

import (
	"sync"
	"testing"
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
