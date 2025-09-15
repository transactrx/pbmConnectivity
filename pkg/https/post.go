package https

import (
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
	"log"
	"strconv"
	"strings"
	//	"github.com/transactrx/pbmHTTP/pkg/config"
	//"github.com/transactrx/https/pkg/https"
)

func IsDebugMode() bool {
	return Cfg.IsDebugMode
}

func (hpc HTTPPBMConnect) Post(claim []byte, headers map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {

	//log.Printf("Post %v", headers)
	_transmissionId := ""
	_appendTidToUrl := false
	// Inject bearer token if not already present
	if hpc.TokenMgr != nil && hpc.TokenMgr.IsValidTokenSettings() {
		token, err := hpc.TokenMgr.GetToken() // Ensure this returns a valid/refreshed token
		if IsDebugMode() {
			log.Printf("post dynamic Token ON case token value: %s error: %v", token, err)
		}
		if err == nil && token != "" {
			authHeader := Header{
				Key:   "Authorization",
				Value: token,

				Base64encode: false,
				Prefix:       "Bearer",
			}
			hpc.Conf.Headers = append(hpc.Conf.Headers, authHeader)
		} else {
			log.Printf("post dynamic token emtpy or error")
			return []byte("error in http post"), nil, pbmlib.ErrorCode.TRX15
		}
	} else {
		log.Printf("post dynamic token OFF or Invalid Settings")
	}

	for pKey, pVal := range headers {
		// Skip any header that contains "_key"
		if strings.HasPrefix(pKey, "_") {
			if pKey == "_transmissionId" {
				_transmissionId = pVal[0]
			}
			if pKey == "_appendToUrl" {
				val, err := strconv.ParseBool(pVal[0])
				if err != nil {
					// fall back or log if it's not a valid bool string
					log.Printf("invalid _appendTidToUrl value: %s", pVal[0])
					continue
				} else {
					_appendTidToUrl = val
				}
			}
			continue
		}

		//log.Printf("Header key:%s value: %s", pKey, pVal[0])
		newHeader := Header{
			Key:   pKey,
			Value: pVal[0],
		}
		hpc.Conf.Headers = append(hpc.Conf.Headers, newHeader)
	}
	url := BuildUrl(hpc.Conf, _appendTidToUrl, _transmissionId)
	if IsDebugMode() {
		log.Printf("post dump - headers: %#v  request: %v", hpc.Conf.Headers, string(claim))
	}

	resp, httpCode, err := FastPost(claim, hpc.Conf, url)
	if err != nil {
		// Should not return 200 for errors.
		if httpCode == 200 {
			httpCode = 500
		}
		errorInfo, _ := MapHTTPStatusToTRXCode(httpCode)
		return []byte("error in http post"), nil, errorInfo
	}

	errorInfo, _ := MapHTTPStatusToTRXCode(httpCode)

	if errorInfo != pbmlib.ErrorCode.TRX00 && Cfg.IsDebugMode {
		log.Printf("error dump - httpCode: %v   headers: %#v   response: %v", httpCode, headers, resp)
	}

	return []byte(resp), nil, errorInfo
}

func BuildUrl(conf RouteInfo, _appendToUrl bool, _transmissionId string) string {

	url := conf.PbmUrl
	if _appendToUrl == true && _transmissionId != "" {
		base := strings.TrimRight(conf.PbmUrl, "/")
		url = base + "/" + _transmissionId
	}
	return url
}
