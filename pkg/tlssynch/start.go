package tlssynch

import (
	"log"
	"sync/atomic"
	"time"	
	"github.com/transactrx/pbmoptumsxc/helpers"
)

type TLSSyncConnect struct {
	test string
}
type Config struct {
	PbmUrl                       string
	PbmPort                      string
	PbmReceiveTimeOut            string
	PbmInsecureSkipVerify        bool
	TlsSplitHandshake            bool
	PbmUrls                      []string
	PbmActiveSites               []bool
	PauseSiteIfFailureHigherThan int
}
type Site struct {
	URL            string
	Active         bool
	activeClaims   atomic.Int32 // Tracks # of claims awaiting responses
	failedClaims   atomic.Int32
	Paused         bool
	pauseCount     int       // number of consecutive pauses
	lastPausedTime time.Time // timestamp of the last pause
	failureRate    float64
}

const PBM_DATA_BUFFER = 16384
var Cfg Config
var Sites []Site

func (pc *TLSSyncConnect) Start(cfgMap map[string]interface{}) error {
	log.Printf("TLSSyncConnect::Start")

	Cfg.PbmUrls = helpers.getStringSlice(cfgMap, "pbmUrl")
	Cfg.PbmActiveSites = helpers.getBoolSlice(cfgMap, "pbmActiveSites")
	Cfg.PbmUrl = helpers.getString(cfgMap, "pbmUrl")
	Cfg.PbmPort = helpers.getString(cfgMap, "pbmPort")
	Cfg.PbmReceiveTimeOut = helpers.getString(cfgMap, "pbmReceiveTimeOut")
	Cfg.PbmInsecureSkipVerify = helpers.getBoolWithDefault(cfgMap, "pbmInsecureSkipVerify", false)
	Cfg.TlsSplitHandshake = helpers.getBoolWithDefault(cfgMap, "TlsSplitHandshake", true)
	Cfg.PauseSiteIfFailureHigherThan = helpers.getIntWithDefault(cfgMap, "PauseSiteIfFailureHigherThan", 0)

	SetupSites()

	log.Printf("Start: configuration: %v", Cfg)

	if Cfg.PauseSiteIfFailureHigherThan > 0 {
		StartSiteResetMonitor()
	}

	return nil
}


func SetupSites() {
	activeSite := false
	Sites = make([]Site, len(Cfg.PbmUrls))
	// Initialize sites based on parsed URLs
	for i, url := range Cfg.PbmUrls {
		activeSite = false
		activeSite = Cfg.PbmActiveSites[i]
		Sites[i] = Site{URL: url, Active: activeSite}
	}
}
