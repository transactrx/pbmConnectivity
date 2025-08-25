package tlssynch

import (
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/transactrx/pbmConnectivity/pkg/helpers"
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
	PbmActiveSites               []bool
	PauseSiteIfFailureHigherThan int
	DebugEnabled                 bool
}
type Site struct {
	URL            string
	Active         bool
	activeClaims   atomic.Int32 // Tracks # of claims awaiting responses
	failedClaims   atomic.Int32
	totalClaims    atomic.Int32
	Paused         bool
	pauseCount     int       // number of consecutive pauses
	lastPausedTime time.Time // timestamp of the last pause
	failureRate    float64
}

const PBM_DATA_BUFFER = 16384

var Cfg Config
var Sites []Site

func (pc *TLSSyncConnect) Start(cfgMap map[string]interface{}) error {
	log.Printf("TLSSynchConnect::Start")

	Cfg.PbmActiveSites = helpers.GetBoolSlice(cfgMap, "pbmActiveSites")
	Cfg.PbmUrl = helpers.GetString(cfgMap, "pbmUrl")
	Cfg.PbmPort = helpers.GetString(cfgMap, "pbmPort")
	Cfg.PbmReceiveTimeOut = helpers.GetString(cfgMap, "pbmReceiveTimeOut")
	Cfg.PbmInsecureSkipVerify = helpers.GetBoolWithDefault(cfgMap, "pbmInsecureSkipVerify", false)
	Cfg.TlsSplitHandshake = helpers.GetBoolWithDefault(cfgMap, "TlsSplitHandshake", true)
	Cfg.PauseSiteIfFailureHigherThan = helpers.GetIntWithDefault(cfgMap, "PauseSiteIfFailureHigherThan", 0)
	Cfg.DebugEnabled = helpers.GetBool(cfgMap, "debugEnabled", false)

	SetupSites()

	log.Printf("Start: configuration: %v", Cfg)

	if IsSiteHealthCheckEnabled() {
		StartSiteResetMonitor()
	}

	return nil
}

func IsSiteHealthCheckEnabled() bool {
	retValue := false
	if len(Cfg.PbmUrl) > 1 && Cfg.PauseSiteIfFailureHigherThan > 0 {
		retValue = true
	}
	return retValue
}

func SetupSites() {
	activeSite := false
	// pbmurl has this syntax pbmurl:  site1,site2,sitex
	urlSites := strings.Split(Cfg.PbmUrl, ",")
	Sites = make([]Site, len(urlSites))
	// Initialize sites based on parsed URLs
	for i, url := range urlSites {
		activeSite = false
		activeSite = Cfg.PbmActiveSites[i]
		Sites[i] = Site{URL: url, Active: activeSite}
	}
}
