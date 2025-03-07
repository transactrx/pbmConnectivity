package https

import (
	"log"

	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
//	"github.com/transactrx/pbmHTTP/pkg/config"
	//"github.com/transactrx/https/pkg/https"
)
func IsDebugMode() bool {
	return Cfg.IsDebugMode
}

func (hpc HTTPPBMConnect) Post(claim []byte, headers map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {

	for pKey, pVal := range headers {
		// Create a new Header with the value and append it to the Headers slice.
		newHeader := Header{
			Key:   pKey,
			Value: pVal[0],
		}
		hpc.Conf.Headers = append(hpc.Conf.Headers, newHeader)
	}

	if IsDebugMode() {
		log.Printf("post dump - headers: %#v  request: %v", headers, string(claim))
	}

	resp, httpCode, err := FastPost(claim, hpc.Conf)
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

