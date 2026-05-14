package https

import (
	"fmt"
	"github.com/transactrx/pbmConnectivity/pkg/helpers"
	"log"
	"os"
	"time"
)

type HTTPPBMConnect struct {
	Conf     RouteInfo
	TokenMgr *TokenManager
}

type Config struct {
	Routes                   []RouteInfo `json:"routes"`
	DbName                   string      `json:"dbName"`
	DbUserName               string      `json:"dbUserName"`
	DbPassword               string      `json:"dbPassword"`
	DbHost                   string      `json:"dbHost"`
	DBPort                   string      `json:"dbPort"`
	DBSslMode                string      `json:"dbSslMode"`
	IsDebugMode              bool
	PbmUrl                   string
	PbmPort                  string
	PbmReceiveTimeOut        string
	PbmInsecureSkipVerify    bool
	TlsSplitHandshake        bool
	DisableConnectionPooling bool
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

// var TokenMgr *TokenManager
var TokenCfg TokenConfig

func (pc *HTTPPBMConnect) Start(cfgMap map[string]interface{}) error {
	log.Println("HTTPPBMConnect::Start")
	Cfg.PbmUrl = helpers.GetString(cfgMap, "pbmUrl")
	Cfg.PbmPort = helpers.GetString(cfgMap, "pbmPort")
	Cfg.PbmReceiveTimeOut = helpers.GetString(cfgMap, "pbmReceiveTimeOut")
	Cfg.PbmInsecureSkipVerify = helpers.GetBool(cfgMap, "pbmInsecureSkipVerify", false)
	Cfg.TlsSplitHandshake = helpers.GetBool(cfgMap, "TlsSplitHandshake", true)
	Cfg.IsDebugMode = helpers.GetBool(cfgMap, "debugEnabled", false)
	Cfg.DisableConnectionPooling = helpers.GetBool(cfgMap, "disableConnectionPooling", false)
	TokenCfg.ClientID = helpers.GetString(cfgMap, "clientId")
	TokenCfg.ClientSecret = helpers.GetString(cfgMap, "clientSecret")
	TokenCfg.TokenURL = helpers.GetString(cfgMap, "tokenUrl")
	stringTokenType := helpers.GetString(cfgMap, "tokenType")
	TokenCfg.TokenScope = helpers.GetString(cfgMap, "tokenScope")
	TokenCfg.TokenGrantType, _ = ParseTokenType(stringTokenType)
	TokenCfg.HTTPTimeout = 5 * time.Second
	TokenMgr := NewTokenManagerWithConfig(TokenCfg)
	pc.TokenMgr = TokenMgr
	hostName, _ := os.Hostname()
	TokenMgr.host = hostName

	if pc.TokenMgr.IsValidTokenSettings() {
		//go GenerateTokens(pc.TokenMgr)
		go TokenMgr.AutoRefreshToken()
	}
	CreateGlobalHttpContext()
	return nil
}

func GenerateTokens(TokenMgr *TokenManager) {
	TokenMgr.GetToken()
}

func ParseTokenType(value string) (TokenType, error) {
	switch value {
	case "ClientCredentials":
		return ClientCredentials, nil
	case "AuthorizationCode":
		return AuthorizationCode, nil
	default:
		return "", fmt.Errorf("invalid token type: %s", value)
	}
}
