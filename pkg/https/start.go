package https

import (
	"github.com/transactrx/pbmConnectivity/pkg/helpers"
	"log"
)

type HTTPPBMConnect struct {
	Conf RouteInfo
}

type Config struct {
	Routes                []RouteInfo `json:"routes"`
	DbName                string      `json:"dbName"`
	DbUserName            string      `json:"dbUserName"`
	DbPassword            string      `json:"dbPassword"`
	DbHost                string      `json:"dbHost"`
	DBPort                string      `json:"dbPort"`
	DBSslMode             string      `json:"dbSslMode"`
	IsDebugMode           bool
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
	log.Println("HTTPPBMConnect::Start")

	Cfg.PbmUrl = helpers.GetString(cfgMap, "pbmUrl")
	Cfg.PbmPort = helpers.GetString(cfgMap, "pbmPort")
	Cfg.PbmReceiveTimeOut = helpers.GetString(cfgMap, "pbmReceiveTimeOut")
	Cfg.PbmInsecureSkipVerify = helpers.GetBool(cfgMap, "pbmInsecureSkipVerify", false)
	Cfg.TlsSplitHandshake = helpers.GetBool(cfgMap, "TlsSplitHandshake", true)
	Cfg.IsDebugMode = helpers.GetBool(cfgMap, "debugEnabled", false)

	CreateGlobalHttpContext()

	return nil
}
