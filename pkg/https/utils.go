package https

import (
	"context"
	"errors"
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
	"github.com/valyala/fasthttp"
	"net"
	"os"
	"syscall"
	//"log"
	//"time"
)

func PrintStats(p *HTTPPBMConnect) {
	for {
		//log.Printf("%s", p.stats.WriteStats())
		//time.Sleep(5 * time.Minute)
	}
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}

	// 1) Common sentinels
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, os.ErrDeadlineExceeded) ||
		errors.Is(err, fasthttp.ErrTimeout) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	// 2) Anything implementing net.Error with Timeout()==true
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}

	// 3) net.OpError often wraps dial/read/write timeouts
	var op *net.OpError
	if errors.As(err, &op) {
		// op.Timeout() checks underlying error too
		if op.Timeout() {
			return true
		}
		// belt & suspenders: direct syscall check on the wrapped error
		if errors.Is(op.Err, syscall.ETIMEDOUT) {
			return true
		}
	}

	return false
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
		return pbmlib.ErrorCode.TRX07, isError
	default:
		return pbmlib.ErrorCode.TRX9999, isError
	}
}
