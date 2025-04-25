package socketsynch

import (
	"log"
	"sync/atomic"
	"time"
	"github.com/transactrx/pbmConnectivity/pkg/helpers"
)

type SocketSynchConnect struct {
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

func (pc *SocketSynchConnect) Start(cfgMap map[string]interface{}) error {
	log.Printf("SocketSynchConnect::Start")

	Cfg.PbmUrls = helpers.GetStringSlice(cfgMap, "pbmUrl")
	Cfg.PbmActiveSites = helpers.GetBoolSlice(cfgMap, "pbmActiveSites")
	Cfg.PbmUrl = helpers.GetString(cfgMap, "pbmUrl")
	Cfg.PbmPort = helpers.GetString(cfgMap, "pbmPort")
	Cfg.PbmReceiveTimeOut = helpers.GetString(cfgMap, "pbmReceiveTimeOut")
	Cfg.PauseSiteIfFailureHigherThan = helpers.GetIntWithDefault(cfgMap, "PauseSiteIfFailureHigherThan", 0)

	SetupSites()

	log.Printf("Start: configuration: %v", Cfg)

	if IsSiteHealthCheckEnabled() {
		StartSiteResetMonitor()
	}

	return nil
}

func IsSiteHealthCheckEnabled()bool{
	retValue := false
	if(len(Cfg.PbmUrls)> 1 && Cfg.PauseSiteIfFailureHigherThan > 0){
		retValue = true
	}
	return retValue
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
