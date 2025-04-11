package tlssynch

import (
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
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
func (pc *TLSSyncConnect) Start(cfgMap map[string]interface{}) error {

	log.Printf("TLSSyncConnect::Start")
	tmp, ok := cfgMap["pbmUrl"].(string)
	if ok {

		urlSites := strings.Split(tmp, ",")
		Cfg.PbmUrls = make([]string, len(urlSites))
		for i, v := range urlSites {
			if v == "true" {
				Cfg.PbmUrls[i] = v
			} else {
				Cfg.PbmUrls[i] = v
			}
		}
	} else {
		log.Printf("Start Url(s) not Provided failed")
	}
	tmp, ok = cfgMap["pbmActiveSites"].(string) // idea is to provide a comma delimitted boolean values (e.g true,false,true,false,.... site-n
	if ok {
		activeSites := strings.Split(tmp, ",")
		Cfg.PbmActiveSites = make([]bool, len(activeSites))
		for i, v := range activeSites {
			if v == "true" {
				Cfg.PbmActiveSites[i] = true
			} else {
				Cfg.PbmActiveSites[i] = false
			}
		}

		log.Printf("values are %v", Cfg.PbmActiveSites)

		//pc.Cfg.PbmQueueTimeOut = tmp
	} else {
		log.Printf("Start site(s) status not Provided failed")
	}

	SetupSites()

	tmp, ok = cfgMap["PauseSiteIfFailureHigherThan"].(string)
	Cfg.PauseSiteIfFailureHigherThan = 80
	if ok {
		Cfg.PauseSiteIfFailureHigherThan, _ = strconv.Atoi(tmp)
	} else {
		log.Printf("Start PauseSiteIfFailureHigherThan not Provided failed")
	}

	tmp, ok = cfgMap["pbmUrl"].(string)
	if ok {
		Cfg.PbmUrl = tmp
	} else {
		log.Printf("Start Url not Provided failed")
	}
	tmp, ok = cfgMap["pbmPort"].(string)
	if ok {
		Cfg.PbmPort = tmp
	} else {
		log.Printf("Start port not Provided failed")
	}
	tmp, ok = cfgMap["pbmReceiveTimeOut"].(string)

	if ok {
		Cfg.PbmReceiveTimeOut = tmp
	} else {
		log.Printf("Start receive time-out not Provided failed")
	}

	tmpBool, ok1 := cfgMap["pbmInsecureSkipVerify"].(bool)

	if ok1 {
		Cfg.PbmInsecureSkipVerify = tmpBool
	} else {
		log.Printf("PbmInsecureSkipVerify not Provided failed")
		Cfg.PbmInsecureSkipVerify = false
	}

	tmpBool, ok = cfgMap["TlsSplitHandshake"].(bool)
	Cfg.TlsSplitHandshake = true

	if ok1 {
		Cfg.TlsSplitHandshake = tmpBool

	} else {
		log.Printf("TlsSplitHandshake not Provided - default to true")
	}
	log.Printf("Start: configuration: %v", Cfg)
	if Cfg.PauseSiteIfFailureHigherThan > 0 {
		StartSiteResetMonitor()
	}

	return nil

}
