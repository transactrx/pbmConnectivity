package tlspersistedsynch

import (
	"log"
	"time"
)

func (ctx *TlsContext) EvaluateSiteHealth() {
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

	// If only 1 or 0 sites are active, unpause all to ensure traffic can continue
	if activeCount <= 1 {
		for i := range ctx.sites {
			site := ctx.sites[i]
			if site.Paused {
				log.Printf("Unpausing site %s as only one site is available.\n", site.URL)
				site.Paused = false
				site.pauseCount = 0
				site.failedClaims.Store(0)
			}
		}
	}
}

func (ctx *TlsContext) StartSiteResetMonitor() {
	go func() {
		baseBackoff := 2 * time.Minute
		maxBackoff := 30 * time.Minute

		for {
			time.Sleep(1 * time.Minute) // check more frequently
			now := time.Now()
			for i := range ctx.sites {
				site := ctx.sites[i]
				log.Printf("Site[%d]: %v", i, site)

				site.failedClaims.Store(0)
				site.activeClaims.Store(0)
				site.failureRate = 0

				if site.Paused {
					backoff := baseBackoff * time.Duration(1<<site.pauseCount)
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					if now.Sub(site.lastPausedTime) >= backoff {
						log.Printf("Auto-unpausing site %s after backoff (%v).\n", site.URL, backoff)
						site.Paused = false
						site.pauseCount = 0
					}
				}
			}
		}
	}()
}
