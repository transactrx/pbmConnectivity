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
	log.Printf("tlspersynch.post tid: %s finding chnl...", tid)

	session , index, err := Ctx.FindConnection()
	if err != nil {
		log.Printf("tlspersynch.post tid: %s no channel found", tid)
		return nil, nil, pbmlib.ErrorCode.TRX10
	}
	log.Printf("tlspersynch.post[%d]  tid: %s",index,tid)
	err = session.Write(index, claim)
	if err != nil {
		log.Printf("tlspersynch.post[%d]  tid: %s write failed", index,tid)
		return nil, nil, pbmlib.ErrorCode.TRX10
	}
	//log.Printf("READING.....")
	// Create a context with a timeout
	appCtx, cancel := context.WithTimeout(context.Background(), time.Duration(readTimeOut)*time.Second) // adjust the timeout as needed
	defer cancel()

	response, err := session.Read(appCtx, index)
	if err != nil {
		Ctx.IncrementError(index)
		if err == context.DeadlineExceeded {
			log.Printf("tlspersynch.post[%d]  tid: %s read failed error: timeout disconnecting",index, tid)
			Ctx.ReleaseConnection(index)
			Ctx.DisconnectSession(index)			 
			return nil, nil, pbmlib.ErrorCode.TRX05
		} else {
			log.Printf("tlspersynch.post[%d]  tid: %s read failed error: %v",index,tid ,err)
			Ctx.ReleaseConnection(index)
			return nil, nil, pbmlib.ErrorCode.TRX10
		}

	}
	Ctx.ClearError(index)
	Ctx.ReleaseConnection(index)

	//log.Printf("Response: %s", string(response))

	return response, nil, pbmlib.ErrorCode.TRX00
}
