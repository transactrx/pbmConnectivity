package asynchflow

import (
	"log"
	"strconv"
	"sync"
	"time"
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

var responseChans sync.Map       // tid vs responsechnl 
var responsePbmHeader sync.Map  // headertoPBM vs tid 

type Claim struct {
	Tid    string
	Claim  []byte
	Header map[string][]string
}

func (pc *AsynchFlow) Post(claim []byte, header map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {

	readTimeOut, _ := strconv.Atoi(Cfg.PbmReceiveTimeOut)
	tid := "Unknown-TID"
	requestHeader := "nodata"
	if values, ok := header["transmissionId"]; ok && len(values) > 0 {
		tid = values[0]
	}
	if values, ok := header["headerValueToCheck"]; ok && len(values) > 0 {
		requestHeader = values[0]
	}
	log.Printf("asynch.post tid: %s headerValue: %s FindChnl... readtimeout: %d", tid, requestHeader, readTimeOut)
	session, index, err := Ctx.FindConnection()
	if err != nil || index == -1 {
		log.Printf("asynch.post tid: %s no channel found", tid)
		return nil, nil, pbmlib.ErrorCode.TRX08 
	}
	log.Printf("asynch.post[%d]  tid: %s", index, tid)
	respCh := make(chan Response, 1) // Buffered to avoid goroutine leaks
	responseChans.Store(tid, respCh)
	defer responseChans.Delete(tid)	
	responsePbmHeader.Store(requestHeader,tid)
	defer responsePbmHeader.Delete(requestHeader)
	
	err = session.Write(index, claim)
	if err != nil {
		log.Printf("asynch.post[%d]  tid: %s write failed", index, tid)
		return nil, nil, pbmlib.ErrorCode.TRX10
	}
	var resp Response
	// Wait for response with timeout
	select {
	case resp = <-respCh:
		log.Printf("Received response: %s", resp.status)
	case <-time.After(5 * time.Second):
		log.Println("Timed out waiting for response")
		return nil, nil, pbmlib.ErrorCode.TRX05
	}
	log.Printf("Response: %s", string(resp.data))
	return resp.data, nil, pbmlib.ErrorCode.TRX00
}
