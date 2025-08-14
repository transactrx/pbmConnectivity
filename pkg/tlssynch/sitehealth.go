package tlssynch

import (
	"fmt"
	"log"
	"math"
	"time"
)

func EvaluateSiteHealth() {
	activeCount := 0
	// Count active and unpaused sites
	for i := range Sites {
		if Sites[i].Active && !Sites[i].Paused {
			activeCount++
		}
	}
	var pausable []*Site
	// Identify pausable sites based on failure rate
	for i := range Sites {
		site := &Sites[i]
		claims := site.activeClaims.Load()
		failures := site.failedClaims.Load()

		if !site.Paused {
			if claims >= 10 {
				failureRate := (float64(failures) / float64(claims)) * 100
				if failureRate > float64(Cfg.PauseSiteIfFailureHigherThan) {
					site.failureRate = failureRate
					pausable = append(pausable, site)
				} else {
					// Leave site.failureRate as-is or maybe keep the latest computed value:
					site.failureRate = failureRate
				}
			} else {
				// Too few claims to trust failure rate, maybe clear or keep it undefined
				site.failureRate = -1 // or just skip setting it
				if int(failures) > Cfg.PauseSiteIfFailureHigherThan {
					pausable = append(pausable, site)
				}
			}
		}

	}
	// Pause sites ensuring at least one remains active
	for _, site := range pausable {
		if activeCount <= 1 {
			break
		}
		site.Paused = true
		site.pauseCount++
		site.lastPausedTime = time.Now()
		activeCount--
		log.Printf("Pausing site %s due to high failure rate (%.2f%%), backoff level %d.\n", site.URL, site.failureRate*100, site.pauseCount)
	}
}

func GetNextUrl() (string, *Site) {
	var (
		url          string
		selectedSite *Site
	)

	if len(Sites) == 0 {
		log.Printf("GetNextUrl failed Site number is zero")
		return "", nil
	}

	if IsSiteHealthCheckEnabled() {
		EvaluateSiteHealth()
	}

	var (
		bestActiveClaims = int32(math.MaxInt32)
		bestFailures     = int32(math.MaxInt32)
		bestFailPct      = float64(1.0) // 100%
	)

	for i := 0; i < len(Sites); i++ {
		site := &Sites[i]
		if !site.Active || site.Paused {
			continue
		}

		total := site.totalClaims.Load()
		claims := site.activeClaims.Load()
		failures := site.failedClaims.Load()
		failPct := 0.0
		if total > 0 {
			failPct = float64(failures) / float64(total)
		}
		// Select site with:
		// 1. Least activeClaims (load balancing)
		// 2. Lowest failure percentage
		// 3. Lowest raw failure count (tie-breaker)

		if claims < bestActiveClaims ||
			(claims == bestActiveClaims && failPct < bestFailPct) ||
			(claims == bestActiveClaims && failPct == bestFailPct && failures < bestFailures) {

			bestActiveClaims = claims
			bestFailures = failures
			bestFailPct = failPct
			selectedSite = site
		}

	}

	if selectedSite != nil {
		url = selectedSite.URL
		log.Printf("GetNextUrl bestSite: %s bestActiveClaims: %d bestFailPct: %.2f bestFailures: %d", url, bestActiveClaims, bestFailPct, bestFailures)
		selectedSite.activeClaims.Add(1)
		selectedSite.totalClaims.Add(1)
	}

	return url, selectedSite
}

func StartSiteResetMonitor() {
	go func() {
		baseBackoff := 2 * time.Minute
		maxBackoff := 30 * time.Minute

		for {
			time.Sleep(1 * time.Minute) // check more frequently
			now := time.Now()
			for i := range Sites {
				site := &Sites[i]
				if Cfg.DebugEnabled {
					log.Printf("%s", site.PrintStats())
				}

				if !site.Paused {
					site.failedClaims.Store(0)
					site.activeClaims.Store(0)
					site.totalClaims.Store(0)
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
