package https

import (
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
	//"log"
	//"time"
)

func PrintStats(p *HTTPPBMConnect) {
	for {
		//log.Printf("%s", p.stats.WriteStats())
		//time.Sleep(5 * time.Minute)
	}
}

// MapHTTPStatusToTRXCode maps an HTTP status code to a TRX code and returns the corresponding ErrorInfo and a boolean
// indicating if it's considered an error
func MapHTTPStatusToTRXCode(httpCode int) (pbmlib.ErrorInfo, bool) {
	var isError bool

	// By default, consider anything outside 2xx as an error
	isError = !(httpCode >= 200 && httpCode < 300)

	switch {
	case httpCode >= 200 && httpCode < 300:
		return pbmlib.ErrorCode.TRX00, false
	case httpCode == 401:
		return pbmlib.ErrorCode.TRX03, isError
	case httpCode == 403:
		return pbmlib.ErrorCode.TRX09, isError
	case httpCode >= 300 && httpCode < 400:
		return pbmlib.ErrorCode.TRX02, isError
	case httpCode >= 400 && httpCode < 500:
		return pbmlib.ErrorCode.TRX04, isError
	case httpCode >= 500 && httpCode < 600:
		return pbmlib.ErrorCode.TRX02, isError
	default:
		return pbmlib.ErrorCode.TRX9999, isError
	}
}
