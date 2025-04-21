package tlspersistedsynch

import (
	"log"
	"reflect"
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
	// Data validation
	HeaderCheck       bool
	HeaderCheckOffset int
	HeaderCheckLen    int
	EndOfRecordChar   byte
	// Find End of Message Using ASCII Len in message 
	MessageLenOffset int 
	MessageLenWidth  int 
	DebugEnabled bool
	MessageLenType int 	// 0 - default - includes header itself 
						// 1 - skipheader - excludes header 
	DisconnectFailedCount int 						
	PauseSiteIfFailureHigherThan int
}

const PBM_DATA_BUFFER = 16384

var Cfg Config

func (pc *TLSPersistedSyncConnect) Start(cfgMap map[string]interface{}) error {

	var err error
	tmp, ok := cfgMap["pbmUrl"].(string)
	if ok {

		urlSites := strings.Split(tmp, ",")
		Cfg.PbmUrl = make([]string, len(urlSites))
		for i, v := range urlSites {
			if v == "true" {
				Cfg.PbmUrl[i] = v
			} else {
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
		Cfg.PbmActiveSites = make([]bool, len(activeSites))
		for i, v := range activeSites {
			if v == "true" {
				Cfg.PbmActiveSites[i] = true
			} else {
				Cfg.PbmActiveSites[i] = false
			}
		}

		log.Printf("values are %v", Cfg.PbmActiveSites)

		//Cfg.PbmQueueTimeOut = tmp
	} else {
		log.Printf("Start site(s) status not Provided failed")
	}

	// TODO MRG 10.8.24 - make sure this is done thru config
	// for now hardcoding it

	tmpBool, ok1 = cfgMap["headerCheck"].(bool)

	if ok1 {
		Cfg.HeaderCheck = tmpBool
	} else {
		log.Printf("HeaderCheck not Provided failed")
		Cfg.HeaderCheck = false
	}

	tmp, ok = cfgMap["headerCheckOffset"].(string)

	if ok {
		num, err := strconv.Atoi(tmp)
		if err == nil {
			Cfg.HeaderCheckOffset = num
		}
	} else {
		log.Printf("HeaderCheckOffset not Provided failed")
	}
	tmp, ok = cfgMap["HeaderCheckLen"].(string)

	if ok {
		num, err := strconv.Atoi(tmp)
		if err == nil {
			Cfg.HeaderCheckLen = num
		}
	} else {
		log.Printf("HeaderCheckLen not Provided failed")
	}

	tmp, ok = cfgMap["endOfRecordChar"].(string)

	if ok {
		log.Printf("tmp: %v",tmp)
		if tmp == "EOT" {
			Cfg.EndOfRecordChar = 0x04
		} else if tmp == "LEN" {
			Cfg.EndOfRecordChar = 0x00 // use ASCII length vs delimiter
		} else {
			Cfg.EndOfRecordChar = 0x03
		}
	} else {
		log.Printf("endOfRecordChar not Provided failed")
	}
	tmp, ok = cfgMap["msgLenOffset"].(string)

	if ok {
		num, err := strconv.Atoi(tmp)
		if err == nil {
			Cfg.MessageLenOffset = num
		}
	} else {
		log.Printf("msgLenOffset not Provided failed")
	}

	tmp, ok = cfgMap["msgLenWidth"].(string)

	if ok {
		num, err := strconv.Atoi(tmp)
		if err == nil {
			Cfg.MessageLenWidth = num
		}
	} else {
		log.Printf("msgLenWidth not Provided failed")
	}

	tmpBool, ok1 = cfgMap["debugEnabled"].(bool)

	if ok1 {
		Cfg.DebugEnabled= tmpBool
	} else {
		log.Printf("debugEnabled not Provided failed")
		Cfg.DebugEnabled = false
	}

	tmp, ok = cfgMap["messageLenType"].(string)

	if ok {
		Cfg.MessageLenType = 0
		log.Printf("tmp: %v",tmp)
		if tmp == "INCLUDELEN" {
			Cfg.MessageLenType = 0
		} else if tmp == "EXCLUDELEN" {
			Cfg.MessageLenType = 1			
		} 
	} else {
		log.Printf("messageLenType not Provided failed")
	}	

	tmp, ok = cfgMap["DisconnectFailedCount"].(string)

	Cfg.DisconnectFailedCount = 10 // 10 failure disconnect default value 
	if ok {
		num, err := strconv.Atoi(tmp)
		if err == nil {
			Cfg.DisconnectFailedCount = num
		}
	} else {
		log.Printf("DisconnectFailedCount not Provided failed")
	}

	tmp, ok = cfgMap["PauseSiteIfFailureHigherThan"].(string)
	Cfg.PauseSiteIfFailureHigherThan = 0
	if ok {
		Cfg.PauseSiteIfFailureHigherThan, _ = strconv.Atoi(tmp)
	} else {
		log.Printf("Start PauseSiteIfFailureHigherThan not Provided failed")
	}
	
	PrintStructFieldsAndValues(Cfg)
	
	// run TlsContext
	Ctx, err = NewTlsContext(Cfg)
	if err != nil {
		log.Printf("Start NewTlsContext failed error: %s - critical", err)
		panic(err)
	}

	return nil
}



func PrintStructFieldsAndValues(data interface{}) {
	val := reflect.ValueOf(data)
	typ := val.Type()
	
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i).Interface()
		log.Printf("%s: %v\n", field.Name, fieldValue)
	}
}