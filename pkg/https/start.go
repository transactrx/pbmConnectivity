package https

import "log"

type HTTPPBMConnect struct {
	Conf RouteInfo
		
}

type Config struct {
	Routes     []RouteInfo `json:"routes"`
	DbName     string      `json:"dbName"`
	DbUserName string      `json:"dbUserName"`
	DbPassword string      `json:"dbPassword"`
	DbHost     string      `json:"dbHost"`
	DBPort     string      `json:"dbPort"`
	DBSslMode  string      `json:"dbSslMode"`
	IsDebugMode bool
	PbmUrl                string
	PbmPort               string
	PbmReceiveTimeOut     string
	PbmInsecureSkipVerify bool
	TlsSplitHandshake     bool
}
type RouteInfo struct {
	RouteCode string   `json:"routeCode"`
	PbmUrl    string   `json:"pbmUrl"`
	Headers   []Header `json:"headers"`
	Timeout   float64  `json:"timeout"`
//	PbmInsecureSkipVerify bool `json:"certinsecureskipverify"`
}
type Header struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	Base64encode bool   `json:"base64encode"`
	Prefix       string `json:"prefix"`
}

const PBM_DATA_BUFFER = 16384

var Cfg Config

func (pc *HTTPPBMConnect) Start(cfgMap map[string]interface{}) error {

	log.Printf("HTTPPBMConnect::Start")

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

	tmpBool, ok = cfgMap["TlsSplitHandshake"].(bool)
	Cfg.TlsSplitHandshake =true
	if ok1 {
		Cfg.TlsSplitHandshake = tmpBool
	} else {
		log.Printf("TlsSplitHandshake not Provided - default to true")	
	}
	tmpBool, ok1 = cfgMap["debugEnabled"].(bool)
	if ok1 {
		Cfg.IsDebugMode= tmpBool
	} else {
		log.Printf("debugEnabled not Provided failed...")
		Cfg.IsDebugMode = false
	}
//	Cfg.IsDebugMode = true
	CreateGlobalHttpContext()

	return nil
}
