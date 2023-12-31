package tlssynch

import "log"

type TLSSyncConnect struct {
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

func (pc *TLSSyncConnect) Start(cfgMap map[string]interface{}) error {

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
	return nil
}
