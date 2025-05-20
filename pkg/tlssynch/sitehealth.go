package tlssynch

import (
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

	// MRG / CB ->>> if if absolute need to refator this code,,, its left there... 
	// If too few active sites, unpause the oldest paused site
	// if activeCount < 1 {
	// 	var oldestPaused *Site
	// 	var oldestTime time.Time

	// 	for i := range Sites {
	// 		site := &Sites[i]
	// 		if site.Paused {
	// 			if oldestPaused == nil || site.lastPausedTime.Before(oldestTime) {
	// 				oldestPaused = site
	// 				oldestTime = site.lastPausedTime
	// 			}
	// 		}
	// 	}

	// 	if oldestPaused != nil {
	// 		log.Printf("Unpausing site %s as only one site is available.\n", oldestPaused.URL)
	// 		oldestPaused.Paused = false
	// 		oldestPaused.pauseCount = 0
	// 		oldestPaused.failedClaims.Store(0)
	// 		activeCount++
	// 	}
	// }
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

		log.Printf("GetNextUrl site: %s activeClaims: %d bestActiveClaims: %d failPct: %.2f bestFailPct: %.2f failures: %d",
			site.URL, claims, bestActiveClaims, failPct, bestFailPct, failures)

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
				log.Printf("Site[%d]: %v", i, site)

				site.failedClaims.Store(0)
				site.activeClaims.Store(0)
				site.totalClaims.Store(0)
				site.failureRate = 0

				if site.Paused {
					backoff := baseBackoff * time.Duration(1<<site.pauseCount)
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
	}()
}
