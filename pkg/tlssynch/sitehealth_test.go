package tlssynch

import (
	"testing"
	"time"
)

func setupTestTLSSites() {
	Sites = []Site{
		{
			URL:    "tls1.example.com",
			Active: true,
			Paused: false,
		},
		{
			URL:    "tls2.example.com",
			Active: true,
			Paused: false,
		},
		{
			URL:    "tls3.example.com",
			Active: true,
			Paused: false,
		},
	}
	// Initialize atomic counters
	for i := range Sites {
		Sites[i].activeClaims.Store(0)
		Sites[i].failedClaims.Store(0)
		Sites[i].totalClaims.Store(0)
	}
}

func TestTLSEvaluateSiteHealth(t *testing.T) {
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
				setupTestTLSSites()
				// Site 0: 10 claims, 1 failure (10% failure rate - below threshold)
				Sites[0].activeClaims.Store(10)
				Sites[0].failedClaims.Store(1)
			},
			expectPaused: []bool{false, false, false},
		},
		{
			name: "one site paused with high failure rate",
			setupSites: func() {
				setupTestTLSSites()
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

func TestTLSGetNextUrl(t *testing.T) {
	tests := []struct {
		name         string
		setupSites   func()
		expectedURL  string
		expectedSite bool
	}{
		{
			name: "select site with least active claims",
			setupSites: func() {
				setupTestTLSSites()
				Sites[0].activeClaims.Store(5)
				Sites[1].activeClaims.Store(2) // Should be selected
				Sites[2].activeClaims.Store(3)
			},
			expectedURL:  "tls2.example.com",
			expectedSite: true,
		},
		{
			name: "skip paused sites",
			setupSites: func() {
				setupTestTLSSites()
				Sites[0].Paused = true
				Sites[0].activeClaims.Store(0) // Best active claims but paused
				
				Sites[1].activeClaims.Store(5)
				Sites[2].activeClaims.Store(3) // Should be selected
			},
			expectedURL:  "tls3.example.com",
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

func TestTLSIsSiteHealthCheckEnabled(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		expected bool
	}{
		{
			name: "enabled with URL and threshold",
			setup: func() {
				Cfg.PbmUrl = "example.com"
				Cfg.PauseSiteIfFailureHigherThan = 10
			},
			expected: true,
		},
		{
			name: "disabled with short URL",
			setup: func() {
				Cfg.PbmUrl = "x"
				Cfg.PauseSiteIfFailureHigherThan = 10
			},
			expected: false,
		},
		{
			name: "disabled with zero threshold",
			setup: func() {
				Cfg.PbmUrl = "example.com"
				Cfg.PauseSiteIfFailureHigherThan = 0
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := IsSiteHealthCheckEnabled()
			if result != tt.expected {
				t.Errorf("IsSiteHealthCheckEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTLSSitePauseTimeTracking(t *testing.T) {
	setupTestTLSSites()
	Cfg.PauseSiteIfFailureHigherThan = 10 // Low threshold to trigger pause
	
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

func TestTLSSetupSites(t *testing.T) {
	// Test SetupSites function
	Cfg.PbmUrl = "site1.com,site2.com,site3.com"
	Cfg.PbmActiveSites = []bool{true, false, true}
	
	SetupSites()
	
	if len(Sites) != 3 {
		t.Errorf("Expected 3 sites, got %d", len(Sites))
	}
	
	expectedSites := []struct {
		url    string
		active bool
	}{
		{"site1.com", true},
		{"site2.com", false},
		{"site3.com", true},
	}
	
	for i, expected := range expectedSites {
		if Sites[i].URL != expected.url {
			t.Errorf("Site[%d].URL = %v, want %v", i, Sites[i].URL, expected.url)
		}
		if Sites[i].Active != expected.active {
			t.Errorf("Site[%d].Active = %v, want %v", i, Sites[i].Active, expected.active)
		}
	}
}