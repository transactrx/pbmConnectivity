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
	DebugEnabled     bool
	MessageLenType   int // 0 - default - includes header itself
	// 1 - skipheader - excludes header
	DisconnectFailedCount        int
	PauseSiteIfFailureHigherThan int
}

const PBM_DATA_BUFFER = 16384

var Cfg Config

func (pc *TLSPersistedSyncConnect) Start(cfgMap map[string]interface{}) error {

	var err error

	Cfg.PbmUrl = getStringSlice(cfgMap, "pbmUrl")
	Cfg.PbmPort = getString(cfgMap, "pbmPort")
	Cfg.PbmReceiveTimeOut = getString(cfgMap, "pbmReceiveTimeOut")
	Cfg.PbmInsecureSkipVerify = getBool(cfgMap, "pbmInsecureSkipVerify", false)
	Cfg.PbmOutboundChnls = getInt(cfgMap, "pbmOutboundChnls", 2)
	Cfg.PbmQueueTimeOut = getString(cfgMap, "pbmQueueTimeOut")
	Cfg.PbmActiveSites = getBoolSlice(cfgMap, "pbmActiveSites")
	Cfg.HeaderCheck = getBool(cfgMap, "headerCheck", false)
	Cfg.HeaderCheckOffset = getInt(cfgMap, "headerCheckOffset", 0)
	Cfg.HeaderCheckLen = getInt(cfgMap, "HeaderCheckLen", 0)
	Cfg.EndOfRecordChar = parseEndOfRecordChar(cfgMap)
	Cfg.MessageLenOffset = getInt(cfgMap, "msgLenOffset", 0)
	Cfg.MessageLenWidth = getInt(cfgMap, "msgLenWidth", 0)
	Cfg.DebugEnabled = getBool(cfgMap, "debugEnabled", false)
	Cfg.MessageLenType = parseMessageLenType(cfgMap)
	Cfg.DisconnectFailedCount = getInt(cfgMap, "DisconnectFailedCount", 10)
	Cfg.PauseSiteIfFailureHigherThan = getInt(cfgMap, "PauseSiteIfFailureHigherThan", 0)

	PrintStructFieldsAndValues(Cfg)

	Ctx, err = NewTlsContext(Cfg)
	if err != nil {
		log.Printf("Start NewTlsContext failed error: %s - critical", err)
		panic(err)
	}

	return nil
}

func getString(cfg map[string]interface{}, key string) string {
	if val, ok := cfg[key].(string); ok {
		return val
	}
	log.Printf("%s not provided or not a string", key)
	return ""
}

func getInt(cfg map[string]interface{}, key string, def int) int {
	if val, ok := cfg[key].(string); ok {
		if num, err := strconv.Atoi(val); err == nil {
			return num
		}
		log.Printf("Invalid int for %s: %v", key, val)
	}
	return def
}

func getBool(cfg map[string]interface{}, key string, def bool) bool {
	if val, ok := cfg[key].(bool); ok {
		return val
	}
	log.Printf("%s not provided or not a bool", key)
	return def
}

func getStringSlice(cfg map[string]interface{}, key string) []string {
	if raw, ok := cfg[key].(string); ok {
		parts := strings.Split(raw, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	log.Printf("%s not provided or not a string", key)
	return nil
}

func getBoolSlice(cfg map[string]interface{}, key string) []bool {
	raw, ok := cfg[key].(string)
	if !ok {
		log.Printf("%s not provided or not a string", key)
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]bool, len(parts))
	for i, v := range parts {
		result[i] = strings.ToLower(strings.TrimSpace(v)) == "true"
	}
	return result
}

func parseEndOfRecordChar(cfg map[string]interface{}) byte {
	val := getString(cfg, "endOfRecordChar")
	switch strings.ToUpper(val) {
	case "EOT":
		return 0x04
	case "LEN":
		return 0x00
	default:
		return 0x03
	}
}

func parseMessageLenType(cfg map[string]interface{}) int {
	val := getString(cfg, "messageLenType")
	switch strings.ToUpper(val) {
	case "EXCLUDELEN":
		return 1
	default:
		return 0 // default is INCLUDELEN
	}
}

// func (pc *TLSPersistedSyncConnect) Start(cfgMap map[string]interface{}) error {

// 	var err error
// 	tmp, ok := cfgMap["pbmUrl"].(string)
// 	if ok {

// 		urlSites := strings.Split(tmp, ",")
// 		Cfg.PbmUrl = make([]string, len(urlSites))
// 		for i, v := range urlSites {
// 			if v == "true" {
// 				Cfg.PbmUrl[i] = v
// 			} else {
// 				Cfg.PbmUrl[i] = v
// 			}
// 		}
// 	} else {
// 		log.Printf("Start Url(s) not Provided failed")
// 	}
// 	tmp, ok = cfgMap["pbmPort"].(string)
// 	if ok {
// 		Cfg.PbmPort = tmp
// 	} else {
// 		log.Printf("Start port not Provided failed")
// 	}
// 	tmp, ok = cfgMap["pbmReceiveTimeOut"].(string)

// 	if ok {
// 		Cfg.PbmReceiveTimeOut = tmp
// 	} else {
// 		log.Printf("Start receive time-out not Provided failed")
// 	}

// 	tmpBool, ok1 := cfgMap["pbmInsecureSkipVerify"].(bool)

// 	if ok1 {
// 		Cfg.PbmInsecureSkipVerify = tmpBool
// 	} else {
// 		log.Printf("PbmInsecureSkipVerify not Provided failed")
// 		Cfg.PbmInsecureSkipVerify = false
// 	}
// 	tmp, ok = cfgMap["pbmOutboundChnls"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err != nil {
// 			log.Printf("Start strconv.Atoi failed error:  %s", err)
// 			Cfg.PbmOutboundChnls = 2 // set default to 2
// 		} else {
// 			Cfg.PbmOutboundChnls = num
// 		}
// 	} else {
// 		log.Printf("Total number of chnls not Provided failed")
// 	}
// 	tmp, ok = cfgMap["pbmQueueTimeOut"].(string)

// 	if ok {
// 		Cfg.PbmQueueTimeOut = tmp
// 	} else {
// 		log.Printf("Start queue time-out not Provided failed")
// 	}
// 	tmp, ok = cfgMap["pbmActiveSites"].(string) // idea is to provide a comma delimitted boolean values (e.g true,false,true,false,.... site-n
// 	if ok {
// 		activeSites := strings.Split(tmp, ",")
// 		Cfg.PbmActiveSites = make([]bool, len(activeSites))
// 		for i, v := range activeSites {
// 			if v == "true" {
// 				Cfg.PbmActiveSites[i] = true
// 			} else {
// 				Cfg.PbmActiveSites[i] = false
// 			}
// 		}

// 		log.Printf("values are %v", Cfg.PbmActiveSites)

// 		//Cfg.PbmQueueTimeOut = tmp
// 	} else {
// 		log.Printf("Start site(s) status not Provided failed")
// 	}

// 	// TODO MRG 10.8.24 - make sure this is done thru config
// 	// for now hardcoding it

// 	tmpBool, ok1 = cfgMap["headerCheck"].(bool)

// 	if ok1 {
// 		Cfg.HeaderCheck = tmpBool
// 	} else {
// 		log.Printf("HeaderCheck not Provided failed")
// 		Cfg.HeaderCheck = false
// 	}

// 	tmp, ok = cfgMap["headerCheckOffset"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			Cfg.HeaderCheckOffset = num
// 		}
// 	} else {
// 		log.Printf("HeaderCheckOffset not Provided failed")
// 	}
// 	tmp, ok = cfgMap["HeaderCheckLen"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			Cfg.HeaderCheckLen = num
// 		}
// 	} else {
// 		log.Printf("HeaderCheckLen not Provided failed")
// 	}

// 	tmp, ok = cfgMap["endOfRecordChar"].(string)

// 	if ok {
// 		log.Printf("tmp: %v",tmp)
// 		if tmp == "EOT" {
// 			Cfg.EndOfRecordChar = 0x04
// 		} else if tmp == "LEN" {
// 			Cfg.EndOfRecordChar = 0x00 // use ASCII length vs delimiter
// 		} else {
// 			Cfg.EndOfRecordChar = 0x03
// 		}
// 	} else {
// 		log.Printf("endOfRecordChar not Provided failed")
// 	}
// 	tmp, ok = cfgMap["msgLenOffset"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			Cfg.MessageLenOffset = num
// 		}
// 	} else {
// 		log.Printf("msgLenOffset not Provided failed")
// 	}

// 	tmp, ok = cfgMap["msgLenWidth"].(string)

// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			Cfg.MessageLenWidth = num
// 		}
// 	} else {
// 		log.Printf("msgLenWidth not Provided failed")
// 	}

// 	tmpBool, ok1 = cfgMap["debugEnabled"].(bool)

// 	if ok1 {
// 		Cfg.DebugEnabled= tmpBool
// 	} else {
// 		log.Printf("debugEnabled not Provided failed")
// 		Cfg.DebugEnabled = false
// 	}

// 	tmp, ok = cfgMap["messageLenType"].(string)

// 	if ok {
// 		Cfg.MessageLenType = 0
// 		log.Printf("tmp: %v",tmp)
// 		if tmp == "INCLUDELEN" {
// 			Cfg.MessageLenType = 0
// 		} else if tmp == "EXCLUDELEN" {
// 			Cfg.MessageLenType = 1
// 		}
// 	} else {
// 		log.Printf("messageLenType not Provided failed")
// 	}

// 	tmp, ok = cfgMap["DisconnectFailedCount"].(string)

// 	Cfg.DisconnectFailedCount = 10 // 10 failure disconnect default value
// 	if ok {
// 		num, err := strconv.Atoi(tmp)
// 		if err == nil {
// 			Cfg.DisconnectFailedCount = num
// 		}
// 	} else {
// 		log.Printf("DisconnectFailedCount not Provided failed")
// 	}

// 	tmp, ok = cfgMap["PauseSiteIfFailureHigherThan"].(string)
// 	Cfg.PauseSiteIfFailureHigherThan = 0
// 	if ok {
// 		Cfg.PauseSiteIfFailureHigherThan, _ = strconv.Atoi(tmp)
// 	} else {
// 		log.Printf("Start PauseSiteIfFailureHigherThan not Provided failed")
// 	}

// 	PrintStructFieldsAndValues(Cfg)

// 	// run TlsContext
// 	Ctx, err = NewTlsContext(Cfg)
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
