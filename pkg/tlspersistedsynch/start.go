package tlspersistedsynch

import (
	"log"
	"strconv"
	"strings"
)

type TLSPersistedSyncConnect struct {
	test string
}

var Ctx *TlsContext

type Config struct {
	PbmUrl                []string
	PbmPort               string
	PbmReceiveTimeOut     string
	PbmQueueTimeOut       string 
	PbmInsecureSkipVerify bool
	PbmOutboundChnls      int
	PbmActiveSites        []bool

}

const PBM_DATA_BUFFER = 16384

var Cfg Config

func (pc *TLSPersistedSyncConnect) Start(cfgMap map[string]interface{}) error {

	var err error
	tmp, ok := cfgMap["pbmUrl"].(string)
	if ok {
		
		urlSites := strings.Split(tmp, ",")
		Cfg.PbmUrl = make([]string,len(urlSites))
		for i, v := range urlSites {
			if v == "true" {
				Cfg.PbmUrl[i] = v
			}else{
				Cfg.PbmUrl[i] = v
			}			
		}
	} else {
		log.Printf("Start Url(s) not Provided failed")
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
	tmp, ok = cfgMap["pbmOutboundChnls"].(string)

	if ok {
		num, err := strconv.Atoi(tmp)
		if err != nil {
			log.Printf("Start strconv.Atoi failed error:  %s", err)
			Cfg.PbmOutboundChnls = 2 // set default to 2
		} else {
			Cfg.PbmOutboundChnls = num
		}
	} else {
		log.Printf("Total number of chnls not Provided failed")
	}
	tmp, ok = cfgMap["pbmQueueTimeOut"].(string)

	if ok {
		Cfg.PbmQueueTimeOut = tmp
	} else {
		log.Printf("Start queue time-out not Provided failed")
	}
	tmp, ok = cfgMap["pbmActiveSites"].(string) // idea is to provide a comma delimitted boolean values (e.g true,false,true,false,.... site-n 
	if ok {		
		activeSites := strings.Split(tmp, ",")
		Cfg.PbmActiveSites = make([]bool,len(activeSites))
		for i, v := range activeSites {
			if v == "true" {
				Cfg.PbmActiveSites[i] = true
			}else{
				Cfg.PbmActiveSites[i] = false
			}			
		}

		log.Printf("values are %v",Cfg.PbmActiveSites)


		//Cfg.PbmQueueTimeOut = tmp
	} else {
		log.Printf("Start site(s) status not Provided failed")
	}

	// run TlsContext
	Ctx, err = NewTlsContext(Cfg)
	if err != nil {
		log.Printf("Start NewTlsContext failed error: %s - critical", err)
		panic(err)
	}

	return nil
}
