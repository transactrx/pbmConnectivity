package tlspersistedsynch

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

func (pc *TLSPersistedSyncConnect) Post(claim []byte, header map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {

	//var responseBuffer []byte
	readTimeOut, _ := strconv.Atoi(Cfg.PbmReceiveTimeOut)
	tid := "Unknown-TID"
	if values, ok := header["transmissionId"]; ok && len(values) > 0 {
		tid = values[0]
	}
	log.Printf("tlspersistedsynch.post tid: %s running", tid)

	_, index, err := Ctx.FindConnection()
	if err != nil {
		log.Printf("tlspersistedsynch.post tid: %s no channel found", tid)
		return nil, nil, pbmlib.ErrorCode.TRX10
	}
	log.Printf("tlspersistedsynch.post tid: %s chnl: %d", tid, index)
	err = Ctx.Write(index, claim)
	if err != nil {
		log.Printf("tlspersistedsynch.post tid: %s write failed", tid)
		return nil, nil, pbmlib.ErrorCode.TRX10
	}
	// Create a context with a timeout
	appCtx, cancel := context.WithTimeout(context.Background(), time.Duration(readTimeOut)*time.Second) // adjust the timeout as needed
	defer cancel()

	response, err := Ctx.Read(appCtx, index)
	if err != nil {
		if err == context.DeadlineExceeded {
			log.Printf("tlspersistedsynch.post tid: %s read failed error: timeout", tid)
			Ctx.ReleaseConnection(index)
			return nil, nil, pbmlib.ErrorCode.TRX05
		} else {
			log.Printf("tlspersistedsynch.post tid: %s read failed error: %v",tid ,err)
			Ctx.ReleaseConnection(index)
			return nil, nil, pbmlib.ErrorCode.TRX10
		}

	}
	Ctx.ReleaseConnection(index)

	//log.Printf("Response: %s", string(response))

	return response, nil, pbmlib.ErrorCode.TRX00
}
