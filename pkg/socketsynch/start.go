package socketsynch

import (
	"github.com/transactrx/pbmConnectivity/pkg/helpers"
	"log"
	"sync/atomic"
	"time"
)

type SocketSynchConnect struct {
	Cfg Config
}
type Config struct {
	PbmUrl                       string
	PbmPort                      string
	PbmReceiveTimeOut            string
	PbmInsecureSkipVerify        bool
	TlsSplitHandshake            bool
	PbmUrls                      []string
	PbmPorts                     []string
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

// var Cfg Config
var SiteHealthEnabled = false
var PauseSiteIfFailureHigherThan = 0
var Sites []Site

func (pc *SocketSynchConnect) Start(cfgMap map[string]interface{}) error {
	log.Printf("SocketSynchConnect::Start")

	pc.Cfg.PbmUrls = helpers.GetStringSlice(cfgMap, "pbmUrl")
	pc.Cfg.PbmActiveSites = helpers.GetBoolSlice(cfgMap, "pbmActiveSites")
	pc.Cfg.PbmUrl = helpers.GetString(cfgMap, "pbmUrl")
	pc.Cfg.PbmPort = helpers.GetString(cfgMap, "pbmPort")
	pc.Cfg.PbmPorts = helpers.GetStringSlice(cfgMap, "pbmPort")
	pc.Cfg.PbmReceiveTimeOut = helpers.GetString(cfgMap, "pbmReceiveTimeOut")
	pc.Cfg.PauseSiteIfFailureHigherThan = helpers.GetIntWithDefault(cfgMap, "PauseSiteIfFailureHigherThan", 0)

	//log.Printf("Site information %v len(sites) %d", Sites, len(Sites))
	log.Printf("Start: configuration: %v", pc.Cfg)
	if len(Sites) <= 0 {
		SetupSites(pc.Cfg)
		SiteHealthEnabled = IsSiteHealthCheckEnabled(pc.Cfg)
		PauseSiteIfFailureHigherThan = pc.Cfg.PauseSiteIfFailureHigherThan
		if SiteHealthEnabled {
			StartSiteResetMonitor()
		}
	}

	return nil
}

func IsSiteHealthCheckEnabled(Cfg Config) bool {
	retValue := false
	if len(Cfg.PbmUrls) > 1 && Cfg.PauseSiteIfFailureHigherThan > 0 {
		retValue = true
	}
	return retValue
}

func SetupSites(Cfg Config) {

	Sites = make([]Site, len(Cfg.PbmUrls)*len(Cfg.PbmPorts))
	// Initialize sites based on parsed URLs & ports
	idx := 0
	active := false
	for _, url := range Cfg.PbmUrls {
		for _, port := range Cfg.PbmPorts {
			active = Cfg.PbmActiveSites[idx]
			tmp := url + ":" + port
			Sites[idx] = Site{URL: tmp, Active: active}
			idx++
		}
	}

}
