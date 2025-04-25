package socketpersistedsynch

import (
	"log"
	"reflect"

	"github.com/transactrx/pbmConnectivity/pkg/helpers"
)

type SocketPersistedSyncConnect struct {
	test string
}

var Ctx *SessionContext

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

func (pc *SocketPersistedSyncConnect) Start(cfgMap map[string]interface{}) error {

	var err error

	Cfg.PbmUrl = helpers.GetStringSlice(cfgMap, "pbmUrl")
	Cfg.PbmPort = helpers.GetString(cfgMap, "pbmPort")
	Cfg.PbmReceiveTimeOut = helpers.GetString(cfgMap, "pbmReceiveTimeOut")
	Cfg.PbmInsecureSkipVerify = helpers.GetBool(cfgMap, "pbmInsecureSkipVerify", false)
	Cfg.PbmOutboundChnls = helpers.GetInt(cfgMap, "pbmOutboundChnls", 2)
	Cfg.PbmQueueTimeOut = helpers.GetString(cfgMap, "pbmQueueTimeOut")
	Cfg.PbmActiveSites = helpers.GetBoolSlice(cfgMap, "pbmActiveSites")
	Cfg.HeaderCheck = helpers.GetBool(cfgMap, "headerCheck", false)
	Cfg.HeaderCheckOffset = helpers.GetInt(cfgMap, "headerCheckOffset", 0)
	Cfg.HeaderCheckLen = helpers.GetInt(cfgMap, "HeaderCheckLen", 0)
	Cfg.EndOfRecordChar = helpers.ParseEndOfRecordChar(cfgMap)
	Cfg.MessageLenOffset = helpers.GetInt(cfgMap, "msgLenOffset", 0)
	Cfg.MessageLenWidth = helpers.GetInt(cfgMap, "msgLenWidth", 0)
	Cfg.DebugEnabled = helpers.GetBool(cfgMap, "debugEnabled", false)
	Cfg.MessageLenType = helpers.ParseMessageLenType(cfgMap)
	Cfg.DisconnectFailedCount = helpers.GetInt(cfgMap, "DisconnectFailedCount", 10)
	Cfg.PauseSiteIfFailureHigherThan = helpers.GetInt(cfgMap, "PauseSiteIfFailureHigherThan", 0)

	PrintStructFieldsAndValues(Cfg)

	Ctx, err = NewTlsContext(Cfg)
	if err != nil {
		log.Printf("Start NewTlsContext failed error: %s - critical", err)
		panic(err)
	}

	return nil
}

func IsSiteHealthCheckEnabled()bool{
	retValue := false
	if(len(Cfg.PbmUrl)> 1 && Cfg.PauseSiteIfFailureHigherThan > 0){
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
