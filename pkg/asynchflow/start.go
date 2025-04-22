package asynchflow

import (
	"log"
	"reflect"
	"github.com/transactrx/pbmConnectivity/pkg/helpers"
)

type AsynchFlow struct {	
	Cfg    Config 
	Ctx    *TlsContext
}

//var Ctx *TlsContext

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
	WaitAfterConnectSeconds  int 
}

const PBM_DATA_BUFFER = 16384

//var Cfg Config
func (pc *AsynchFlow) Start(cfgMap map[string]interface{}) error {
	var err error

	pc.Cfg.PbmUrl = helpers.GetStringSlice(cfgMap, "pbmUrl")
	pc.Cfg.PbmPort = helpers.GetString(cfgMap, "pbmPort")
	pc.Cfg.PbmReceiveTimeOut = helpers.GetString(cfgMap, "pbmReceiveTimeOut")
	pc.Cfg.PbmInsecureSkipVerify = helpers.GetBool(cfgMap, "pbmInsecureSkipVerify", false)
	pc.Cfg.PbmOutboundChnls = helpers.GetInt(cfgMap, "pbmOutboundChnls", 2)
	pc.Cfg.PbmQueueTimeOut = helpers.GetString(cfgMap, "pbmQueueTimeOut")
	pc.Cfg.PbmActiveSites = helpers.GetBoolSlice(cfgMap, "pbmActiveSites")
	pc.Cfg.HeaderCheck = helpers.GetBool(cfgMap, "headerCheck", false)
	pc.Cfg.HeaderCheckOffset = helpers.GetInt(cfgMap, "headerCheckOffset", 0)
	pc.Cfg.HeaderCheckLen = helpers.GetInt(cfgMap, "HeaderCheckLen", 0)

	pc.Cfg.EndOfRecordChar = helpers.ParseEndOfRecordChar(cfgMap)
	pc.Cfg.MessageLenOffset = helpers.GetInt(cfgMap, "msgLenOffset", 0)
	pc.Cfg.MessageLenWidth = helpers.GetInt(cfgMap, "msgLenWidth", 0)
	pc.Cfg.DebugEnabled = helpers.GetBool(cfgMap, "debugEnabled", false)
	pc.Cfg.MessageLenType = helpers.ParseMessageLenType(cfgMap)
	pc.Cfg.DisconnectFailedCount = helpers.GetInt(cfgMap, "DisconnectFailedCount", 10)
	pc.Cfg.WaitAfterConnectSeconds = helpers.GetInt(cfgMap, "WaitAfterConnectSeconds", 6)

	PrintStructFieldsAndValues(pc.Cfg)

	pc.Ctx, err = NewTlsContext(pc.Cfg)
	if err != nil {
		log.Fatalf("Start NewTlsContext failed error: %s - critical", err)
	}

	return nil
}

// func (pc *AsynchFlow) Start(cfgMap map[string]interface{}) error {

// 	var err error
// 	tmp, ok := cfgMap["pbmUrl"].(string)
// 	if ok {

// 		urlSites := strings.Split(tmp, ",")
// 		pc.Cfg.PbmUrl = make([]string, len(urlSites))
// 		for i, v := range urlSites {
// 			if v == "true" {
// 				pc.Cfg.PbmUrl[i] = v
// 			} else {
// 				pc.Cfg.PbmUrl[i] = v
// 			}
// 		}
// 	} else {
// 		log.Printf("Start Url(s) not Provided failed")
// 	}
// 	tmp, ok = cfgMap["pbmPort"].(string)
// 	if ok {
// 		pc.Cfg.PbmPort = tmp
// 	} else {
// 		log.Printf("Start port not Provided failed")
// 	}
// 	tmp, ok = cfgMap["pbmReceiveTimeOut"].(string)

// 	if ok {
// 		pc.Cfg.PbmReceiveTimeOut = tmp
// 	} else {
// 		log.Printf("Start receive time-out not Provided failed")
// 	}

// 	tmpBool, ok1 := cfgMap["pbmInsecureSkipVerify"].(bool)

// 	if ok1 {
// 		pc.Cfg.PbmInsecureSkipVerify = tmpBool
// 	} else {
// 		log.Printf("PbmInsecureSkipVerify not Provided failed")
// 		pc.Cfg.PbmInsecureSkipVerify = false
// 	}
// 	tmp, ok = cfgMap["pbmOutboundChnls"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err != nil {
// 			log.Printf("Start strconv.Atoi failed error:  %s", err)
// 			pc.Cfg.PbmOutboundChnls = 2 // set default to 2
// 		} else {
// 			pc.Cfg.PbmOutboundChnls = num
// 		}
// 	} else {
// 		log.Printf("Total number of chnls not Provided failed")
// 	}
// 	tmp, ok = cfgMap["pbmQueueTimeOut"].(string)

// 	if ok {
// 		pc.Cfg.PbmQueueTimeOut = tmp
// 	} else {
// 		log.Printf("Start queue time-out not Provided failed")
// 	}
// 	tmp, ok = cfgMap["pbmActiveSites"].(string) // idea is to provide a comma delimitted boolean values (e.g true,false,true,false,.... site-n
// 	if ok {
// 		activeSites := strings.Split(tmp, ",")
// 		pc.Cfg.PbmActiveSites = make([]bool, len(activeSites))
// 		for i, v := range activeSites {
// 			if v == "true" {
// 				pc.Cfg.PbmActiveSites[i] = true
// 			} else {
// 				pc.Cfg.PbmActiveSites[i] = false
// 			}
// 		}

// 		log.Printf("values are %v", pc.Cfg.PbmActiveSites)

// 		//pc.Cfg.PbmQueueTimeOut = tmp
// 	} else {
// 		log.Printf("Start site(s) status not Provided failed")
// 	}	
// 	tmpBool, ok1 = cfgMap["headerCheck"].(bool)
// 	if ok1 {
// 		pc.Cfg.HeaderCheck = tmpBool
// 	} else {
// 		log.Printf("HeaderCheck not Provided failed")
// 		pc.Cfg.HeaderCheck = false
// 	}

// 	tmp, ok = cfgMap["headerCheckOffset"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			pc.Cfg.HeaderCheckOffset = num
// 		}
// 	} else {
// 		log.Printf("HeaderCheckOffset not Provided failed")
// 	}
// 	tmp, ok = cfgMap["HeaderCheckLen"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			pc.Cfg.HeaderCheckLen = num
// 		}
// 	} else {
// 		log.Printf("HeaderCheckLen not Provided failed")
// 	}

// 	tmp, ok = cfgMap["endOfRecordChar"].(string)

// 	if ok {
// 		log.Printf("endOfRecordChar tmp: %v",tmp)
// 		if tmp == "EOT" {
// 			pc.Cfg.EndOfRecordChar = 0x04
// 		} else if tmp == "LEN" {
// 			pc.Cfg.EndOfRecordChar = 0x00 // use ASCII length vs delimiter
// 		} else {
// 			pc.Cfg.EndOfRecordChar = 0x03
// 		}
// 	} else {
// 		log.Printf("endOfRecordChar not Provided failed")
// 	}
// 	tmp, ok = cfgMap["msgLenOffset"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			pc.Cfg.MessageLenOffset = num
// 		}
// 	} else {
// 		log.Printf("msgLenOffset not Provided failed")
// 	}

// 	tmp, ok = cfgMap["msgLenWidth"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			pc.Cfg.MessageLenWidth = num
// 		}
// 	} else {
// 		log.Printf("msgLenWidth not Provided failed")
// 	}

// 	tmpBool, ok1 = cfgMap["debugEnabled"].(bool)

// 	if ok1 {
// 		pc.Cfg.DebugEnabled= tmpBool
// 	} else {
// 		log.Printf("debugEnabled not Provided failed")
// 		pc.Cfg.DebugEnabled = false
// 	}

// 	tmp, ok = cfgMap["messageLenType"].(string)

// 	if ok {
// 		pc.Cfg.MessageLenType = 0
// 		log.Printf("tmp: %v",tmp)
// 		if tmp == "INCLUDELEN" {
// 			pc.Cfg.MessageLenType = 0
// 		} else if tmp == "EXCLUDELEN" {
// 			pc.Cfg.MessageLenType = 1			
// 		} 
// 	} else {
// 		log.Printf("messageLenType not Provided failed")
// 	}	

// 	tmp, ok = cfgMap["DisconnectFailedCount"].(string)

// 	pc.Cfg.DisconnectFailedCount = 10 // 10 failure disconnect default value 
// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			pc.Cfg.DisconnectFailedCount = num
// 		}
// 	} else {
// 		log.Printf("DisconnectFailedCount not Provided failed")
// 	}

// 	tmp, ok = cfgMap["WaitAfterConnectSeconds"].(string)

// 	pc.Cfg.WaitAfterConnectSeconds = 6
// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			pc.Cfg.WaitAfterConnectSeconds = num
// 		}
// 	} else {
// 		log.Printf("WaitAfterConnectSeconds not Provided failed")
// 	}

	

// 	PrintStructFieldsAndValues(pc.Cfg)
	
// 	// run TlsContext
// 	pc.Ctx, err = NewTlsContext(pc.Cfg)
// 	if err != nil {
// 		log.Printf("Start NewTlsContext failed error: %s - critical", err)
// 		panic(err)
// 	}

// 	return nil
// }

func PrintStructFieldsAndValues(data interface{}) {
	val := reflect.ValueOf(data)
	typ := val.Type()
	
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i).Interface()
		log.Printf("%s: %v\n", field.Name, fieldValue)
	}
}