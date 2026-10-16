package socketasynch

import (
	"sync"
	"testing"
)

func newTestContext(n int) *SessionContext {
	ctx := &SessionContext{}
	for i := 0; i < n; i++ {
		s := &SocketSession{name: "test", chnl: i}
		s.connected.Store(true)
		ctx.sessions = append(ctx.sessions, s)
	}
	return ctx
}

func TestFindLeastBusyChnl(t *testing.T) {
	t.Run("spreads outstanding claims across chnls", func(t *testing.T) {
		ctx := newTestContext(4)
		used := map[int]int{}
		for i := 0; i < 8; i++ {
			_, idx, err := ctx.FindLeastBusyChnl()
			if err != nil {
				t.Fatal(err)
			}
			used[idx]++
		}
		for i := 0; i < 4; i++ {
			if used[i] != 2 {
				t.Errorf("chnl %d got %d claims, want 2 (%v)", i, used[i], used)
			}
		}
	})

	t.Run("released chnl is picked again", func(t *testing.T) {
		ctx := newTestContext(3)
		for i := 0; i < 3; i++ {
			ctx.FindLeastBusyChnl()
		}
		ctx.sessions[1].releaseClaim()
		_, idx, _ := ctx.FindLeastBusyChnl()
		if idx != 1 {
			t.Errorf("got chnl %d, want 1", idx)
		}
	})

	t.Run("skips paused and disconnected chnls", func(t *testing.T) {
		ctx := newTestContext(3)
		ctx.sessions[0].paused.Store(true)
		ctx.sessions[1].connected.Store(false)
		for i := 0; i < 3; i++ {
			_, idx, err := ctx.FindLeastBusyChnl()
			if err != nil || idx != 2 {
				t.Errorf("got chnl %d err %v, want 2", idx, err)
			}
		}
	})

	t.Run("no chnls available", func(t *testing.T) {
		ctx := newTestContext(1)
		ctx.sessions[0].connected.Store(false)
		if _, _, err := ctx.FindLeastBusyChnl(); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("concurrent posts are balanced", func(t *testing.T) {
		ctx := newTestContext(4)
		var wg sync.WaitGroup
		for i := 0; i < 400; i++ {
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
	})
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
