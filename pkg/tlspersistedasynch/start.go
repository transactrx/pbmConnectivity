package tlspersistedasynch

import (
	"log"
	"time"
)

type TLSPersistedASyncConnect struct {
	test string
}
type Config struct {
	PbmUrl            string
	PbmPort           string
	PbmReceiveTimeOut string
	PbmInsecureSkipVerify bool
}
const PBM_DATA_BUFFER = 16384
var Cfg Config

func (pc *TLSPersistedASyncConnect) Start(cfgMap map[string]interface{}) error {

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
	go ManagePbmConnections()
	// ALL object creation needed to handle Asynch claims is needed 
	
	// 1. Create go routines to handle 
       
		// connection management with vendor 
		// connect 
		// reconnect 
		// graceful socket closure 

	// 2. Queue - thread safe  (claim queue )
	// 3. Map  - generic thread safe map to store context (claim information)
	// .... 


	return nil
}

func ManagePbmConnections()  {

	for {
		

		log.Printf("ManagePbmConnections....")
		time.Sleep(10 * time.Second)
	}


	
}
