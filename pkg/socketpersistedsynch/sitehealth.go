package socketpersistedsynch

import (
	"log"
	"time"
	"fmt"
)

func (ctx *SessionContext) EvaluateSiteHealth() {
	activeCount := 0
	var pausable []*Site
	var failureRate float64

	// First pass: count active sites and identify pausable ones
	for i := range ctx.sites {
		site := ctx.sites[i]

		claims := site.activeClaims.Load()
		failures := site.failedClaims.Load()

		if site.Active && !site.Paused {
			activeCount++
		}

		if Cfg.DebugEnabled {
			log.Printf("url: %s claims: %d paused: %t failures: %d Cfg.PauseSiteIfFailureHigherThan: %d", site.URL, claims, site.Paused, failures, Cfg.PauseSiteIfFailureHigherThan)
		}

		if claims >= 10 && !site.Paused {
			failureRate = (float64(failures) / float64(claims)) * 100
			if failureRate > float64(Cfg.PauseSiteIfFailureHigherThan) {
				site.failureRate = failureRate // Optional: store for log clarity
				pausable = append(pausable, site)
			}
		}
		if claims < 10 && !site.Paused {
			if int(failures) > Cfg.PauseSiteIfFailureHigherThan {
				pausable = append(pausable, site)
			}
		}

	}
	CheckPausableSites(pausable, &activeCount)
}

func (s *Site) IsPaused() bool {
	if s == nil {
		return false // default to "not paused"
	}
	return s.Paused
}

func (s *Site) PrintStats() string {
	line := ""
	if s == nil {
		return ""
	}

	activeInt := 0
	pauseInt := 0
	if s.Active {
		activeInt = 1
	}
	if s.Paused {
		pauseInt = 1
	}
	if !s.lastPausedTime.IsZero() {
		line = fmt.Sprintf("pbmsitestats url: %s active: %d inprocess: %d failed: %d paused: %d pausecount: %d failurerate: %f lastpaused: %s", s.URL, activeInt, s.activeClaims.Load(), s.failedClaims.Load(), pauseInt, s.pauseCount, s.failureRate, s.lastPausedTime)
	} else {
		line = fmt.Sprintf("pbmsitestats url: %s active: %d inprocess: %d failed: %d paused: %d pausecount: %d failurerate: %f lastpaused: never", s.URL, activeInt, s.activeClaims.Load(), s.failedClaims.Load(), pauseInt, s.pauseCount, s.failureRate)
	}

	return line
}

func (ctx *SessionContext) StartSiteResetMonitor(frequency time.Duration) {
	go func() {
		for {
			time.Sleep(frequency * time.Minute) // check more frequently
			ctx.CheckSites()
		}
	}()
}

func (ctx *SessionContext) CheckSites() {
	baseBackoff := 2 * time.Minute
	maxBackoff := 30 * time.Minute
	now := time.Now()
	for i := range ctx.sites {
		site := ctx.sites[i]
		if Cfg.DebugEnabled {
			log.Printf("%s", site.PrintStats())
		}

		if !site.Paused {
			site.failedClaims.Store(0)
			site.activeClaims.Store(0)
			site.failureRate = 0
		}

		if site.Paused {
			backoff := baseBackoff * time.Duration(1<<(site.pauseCount-1))
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			if now.Sub(site.lastPausedTime) >= backoff {
				log.Printf("Auto-unpausing site %s after backoff (%v).\n", site.URL, backoff)
				site.Paused = false
				//site.pauseCount = 0
			}
		}
	}

}

func CheckPausableSites(pausable []*Site, activeCount *int) {

	if len(pausable) <= 0 {
		return
	}
	if Cfg.DebugEnabled {
		log.Printf("CheckPausableSites activeCount:%d pausable sites: %d", *activeCount, len(pausable))
	}
	for _, site := range pausable {
		if *activeCount <= 1 {
			break
		}
		site.Paused = true
		site.pauseCount++
		site.lastPausedTime = time.Now()
		*activeCount-- // Decrement as we pause
		log.Printf("Pausing site %s due to high failure rate (%.2f%%) or max.site failures: %d, backoff level %d.\n", site.URL, site.failureRate*100, site.failedClaims.Load(), site.pauseCount)
	}
}
