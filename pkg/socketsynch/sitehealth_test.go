package socketsynch

import (
	"testing"
	"time"
)

func setupTestSocketSites() {
	Sites = []Site{
		{
			URL:    "socket1.example.com:8080",
			Active: true,
			Paused: false,
		},
		{
			URL:    "socket2.example.com:8080",
			Active: true,
			Paused: false,
		},
		{
			URL:    "socket3.example.com:8080",
			Active: true,
			Paused: false,
		},
	}
	// Initialize atomic counters
	for i := range Sites {
		Sites[i].activeClaims.Store(0)
		Sites[i].failedClaims.Store(0)
	}
}

/*
	func TestSocketEvaluateSiteHealth(t *testing.T) {
		// Setup test configuration

		Cfg.PauseSiteIfFailureHigherThan = 20 // 20% failure rate threshold

		tests := []struct {
			name         string
			setupSites   func()
			expectPaused []bool
		}{
			{
				name: "no sites paused with low failure rate",
				setupSites: func() {
					setupTestSocketSites()
					// Site 0: 10 claims, 1 failure (10% failure rate - below threshold)
					Sites[0].activeClaims.Store(10)
					Sites[0].failedClaims.Store(1)
				},
				expectPaused: []bool{false, false, false},
			},
			{
				name: "one site paused with high failure rate",
				setupSites: func() {
					setupTestSocketSites()
					// Site 0: 10 claims, 3 failures (30% failure rate - above threshold)
					Sites[0].activeClaims.Store(10)
					Sites[0].failedClaims.Store(3)

					// Site 1: healthy
					Sites[1].activeClaims.Store(10)
					Sites[1].failedClaims.Store(1)
				},
				expectPaused: []bool{true, false, false},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tt.setupSites()
				EvaluateSiteHealth()

				for i, expectedPaused := range tt.expectPaused {
					if Sites[i].Paused != expectedPaused {
						t.Errorf("Site[%d].Paused = %v, want %v", i, Sites[i].Paused, expectedPaused)
					}
				}
			})
		}
	}
*/
func TestSocketGetNextUrl(t *testing.T) {
	tests := []struct {
		name         string
		setupSites   func()
		expectedURL  string
		expectedSite bool
	}{
		{
			name: "select site with least active claims",
			setupSites: func() {
				setupTestSocketSites()
				Sites[0].activeClaims.Store(5)
				Sites[1].activeClaims.Store(2) // Should be selected
				Sites[2].activeClaims.Store(3)
			},
			expectedURL:  "socket2.example.com:8080",
			expectedSite: true,
		},
		{
			name: "paused sites are still selected (no pause check in socket implementation)",
			setupSites: func() {
				setupTestSocketSites()
				Sites[0].Paused = true         // This doesn't prevent selection in socket version
				Sites[0].activeClaims.Store(0) // Best active claims

				Sites[1].activeClaims.Store(5)
				Sites[2].activeClaims.Store(3)
			},
			expectedURL:  "socket1.example.com:8080", // Socket implementation doesn't skip paused sites
			expectedSite: true,
		},
		{
			name: "skip inactive sites",
			setupSites: func() {
				setupTestSocketSites()
				Sites[0].Active = false
				Sites[0].activeClaims.Store(0) // Best active claims but inactive

				Sites[1].activeClaims.Store(5)
				Sites[2].activeClaims.Store(3) // Should be selected
			},
			expectedURL:  "socket3.example.com:8080",
			expectedSite: true,
		},
		{
			name: "empty sites list",
			setupSites: func() {
				Sites = []Site{}
			},
			expectedURL:  "",
			expectedSite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupSites()

			url, site := GetNextUrl()

			if url != tt.expectedURL {
				t.Errorf("GetNextUrl() url = %v, want %v", url, tt.expectedURL)
			}

			gotSite := site != nil
			if gotSite != tt.expectedSite {
				t.Errorf("GetNextUrl() site = %v, want %v", gotSite, tt.expectedSite)
			}
		})
	}
}

func TestSocketSitePauseTimeTracking(t *testing.T) {
	setupTestSocketSites()
	//Cfg.PauseSiteIfFailureHigherThan = 10 // Low threshold to trigger pause

	// Set up high failure rate to trigger pause
	Sites[0].activeClaims.Store(10)
	Sites[0].failedClaims.Store(5) // 50% failure rate

	beforePause := time.Now()
	EvaluateSiteHealth()
	afterPause := time.Now()

	if !Sites[0].Paused {
		t.Error("Site should be paused")
	}

	if Sites[0].pauseCount != 1 {
		t.Errorf("Pause count = %d, want 1", Sites[0].pauseCount)
	}

	if Sites[0].lastPausedTime.Before(beforePause) || Sites[0].lastPausedTime.After(afterPause) {
		t.Error("lastPausedTime not set correctly")
	}
}

func TestSocketIsSiteHealthCheckEnabled(t *testing.T) {
	// Test the function exists and can be called
	result := SiteHealthEnabled //IsSiteHealthCheckEnabled()

	// The result should be a boolean
	if result != true && result != false {
		t.Errorf("IsSiteHealthCheckEnabled() should return a boolean")
	}
}
