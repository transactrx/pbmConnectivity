package socketsynch

import (
	"log"
	"math"
	"time"
	"fmt"
)

func EvaluateSiteHealth() {
	activeCount := 0
	var pausable []*Site
	var failureRate float64

	// First pass: count active sites and identify pausable ones
	for i := range Sites {
		site := &Sites[i]
		claims := site.activeClaims.Load()
		failures := site.failedClaims.Load()

		if site.Active && !site.Paused {
			activeCount++
		}

		if claims >= 10 && !site.Paused {
			failureRate = (float64(failures) / float64(claims)) * 100
			if failureRate > float64(PauseSiteIfFailureHigherThan) {
				site.failureRate = failureRate // Optional: store for log clarity
				pausable = append(pausable, site)
			}
		}
		if claims < 10 && !site.Paused {
			if int(failures) > PauseSiteIfFailureHigherThan {
				pausable = append(pausable, site)
			}
		}
	}
	// Only pause sites if we’ll still have at least one active site remaining
	for _, site := range pausable {
		if activeCount <= 1 {
			break
		}
		site.Paused = true
		site.pauseCount++
		site.lastPausedTime = time.Now()
		activeCount-- // Decrement as we pause
		log.Printf("Pausing site %s due to high failure rate (%.2f%%), backoff level %d.\n", site.URL, site.failureRate*100, site.pauseCount)
	}
}

func GetNextUrl() (string, *Site) {
	var (
		url          string
		selectedSite *Site
	)

	if len(Sites) == 0 {
		return "", nil
	}

	if SiteHealthEnabled {
		EvaluateSiteHealth()
	}

	var (
		bestClaims   = int32(math.MaxInt32)
		bestFailures = int32(math.MaxInt32)
		bestFailPct  = float64(1.0) // 100%
	)

	for i := 0; i < len(Sites); i++ {
		site := &Sites[i]
		if !site.Active {
			continue
		}
		claims := site.activeClaims.Load()
		failures := site.failedClaims.Load()
		total := claims + failures
		var failPct float64
		if total > 0 {
			failPct = float64(failures) / float64(total)
		} else {
			failPct = 0.0
		}
		// Primary: least claims, then failure pct, then raw failures
		if claims < bestClaims ||
			(claims == bestClaims && failPct < bestFailPct) ||
			(claims == bestClaims && failPct == bestFailPct && failures < bestFailures) {

			bestClaims = claims
			bestFailures = failures
			bestFailPct = failPct
			selectedSite = site
		}
	}

	if selectedSite != nil {
		url = selectedSite.URL
		log.Printf("GetNextUrl bestSite: %s bestActiveClaims: %d bestFailPct: %.2f bestFailures: %d", url, bestClaims, bestFailPct, bestFailures)
		selectedSite.activeClaims.Add(1)
	}

	return url, selectedSite
}

func (pc *SocketSynchConnect) StartSiteResetMonitor() {
	go func() {
		baseBackoff := 2 * time.Minute
		maxBackoff := 30 * time.Minute

		for {
			time.Sleep(1 * time.Minute) // check more frequently
			now := time.Now()
			for i := range Sites {
				site := &Sites[i]
				if pc.Cfg.DebugEnabled {
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
					}
				}
			}
		}
	}()
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