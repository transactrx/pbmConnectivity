package tlspersistedsynch

import (
	"log"
	"strconv"
)

type TLSPersistedSyncConnect struct {
	test string
}

var Ctx *TlsContext

type Config struct {
	PbmUrl                string
	PbmPort               string
	PbmReceiveTimeOut     string
	PbmInsecureSkipVerify bool
	PbmOutboundChnls      int
}

const PBM_DATA_BUFFER = 16384

var Cfg Config

func (pc *TLSPersistedSyncConnect) Start(cfgMap map[string]interface{}) error {

	var err error
	tmp, ok := cfgMap["pbmUrl"].(string)
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
	// run TlsContext
	Ctx, err = NewTlsContext(Cfg)
	if err != nil {
		log.Printf("Start NewTlsContext failed error: %s - critical", err)
		panic(err)
	}

	return nil
}
