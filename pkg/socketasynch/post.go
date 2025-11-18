package socketasynch

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

var responseChans sync.Map     // tid vs responsechnl
var responsePbmHeader sync.Map // headertoPBM vs tid

func (pc *SocketAsyncConnect) Post(claim []byte, header map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {

	readTimeOut, _ := strconv.Atoi(pc.Cfg.PbmReceiveTimeOut)
	tid := "Unknown-TID"
	requestHeader := "nodata"

	if values, ok := header["transmissionId"]; ok && len(values) > 0 {
		tid = values[0]
	}

	if values, ok := header["headerValueToCheck"]; ok && len(values) > 0 {
		requestHeader = values[0]
	}

	log.Printf("socketasynch.post tid: %s headerValue: %s FindChnl...", tid, requestHeader)

	//session, index, err := pc.FindConnection()
	session, index, err := pc.Ctx.FindLeastBusyChnl()
	if err != nil {
		log.Printf("socketasynch.post tid: %s no channel found", tid)
		return nil, nil, pbmlib.ErrorCode.TRX10
	}
	log.Printf("socketasynch.post[%d]  tid: %s", index, tid)

	//////////////////////
	respCh := make(chan Response, 1) // Buffered to avoid goroutine leaks
	responseChans.Store(tid, respCh)
	defer responseChans.Delete(tid)
	responsePbmHeader.Store(requestHeader, tid)
	defer responsePbmHeader.Delete(requestHeader)
	///////////////////////

	err = session.Write(index, claim)
	if err != nil {
		log.Printf("socketasynch.post[%d]  tid: %s write failed", index, tid)
		return nil, nil, pbmlib.ErrorCode.TRX10
	}
	var resp Response
	chnlTimeOut := time.Duration(readTimeOut * int(time.Second))
	// Wait for response with timeout
	select {
	case resp = <-respCh:
		log.Printf("socketasynch.post[%d] tid: %s response received: status: %s len: %d", index, tid, resp.status, len(resp.data))
		session.ResetErrors()
	case <-time.After(chnlTimeOut):
		log.Printf("socketasynch.post[%d] tid: %s timed out waiting for response timeout: %f", index, tid, chnlTimeOut.Seconds())
		session.RegisterError(pc.Cfg.DisconnectFailedCount, 0)
		return nil, nil, pbmlib.ErrorCode.TRX05

	}
	if pc.Cfg.DebugEnabled {
		log.Printf("asynch.post[%d] tid: %s Response: %s", index, tid, string(resp.data))
	}
	return resp.data, nil, pbmlib.ErrorCode.TRX00
}
