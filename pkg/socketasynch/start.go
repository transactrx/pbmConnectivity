package socketasynch

import (
	"log"
	"reflect"

	"github.com/transactrx/pbmConnectivity/pkg/helpers"
)

type SocketAsyncConnect struct {
	Ctx *SessionContext
	Cfg Config
}

type Config struct {
	PbmUrl                []string
	PbmPorts              []string
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


func (pc *SocketAsyncConnect) Start(cfgMap map[string]interface{}) error {
	log.Printf("SocketAsynchConnect::Start")
	var err error

	pc.Cfg.PbmUrl = helpers.GetStringSlice(cfgMap, "pbmUrl")
	pc.Cfg.PbmPorts = helpers.GetStringSlice(cfgMap, "pbmPort")
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
	pc.Cfg.PauseSiteIfFailureHigherThan = helpers.GetInt(cfgMap, "PauseSiteIfFailureHigherThan", 0)

	PrintStructFieldsAndValues(pc.Cfg)

	pc.Ctx, err = pc.NewSessionContext(pc.Cfg)
	if err != nil {
		log.Printf("Start NewSessionContext failed error: %s - critical", err)
		panic(err)
	}

	return nil
}

func (pc *SocketAsyncConnect) IsSiteHealthCheckEnabled() bool {
	retValue := false
	if len(pc.Cfg.PbmUrl) > 1 && pc.Cfg.PauseSiteIfFailureHigherThan > 0 {
		retValue = true
	}
	return retValue
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
