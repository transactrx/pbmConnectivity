package tlspersistedsynch

import (
	"log"

	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

func (pc *TLSPersistedSyncConnect) Post(claim []byte, header map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {

	//var responseBuffer []byte
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
	log.Printf("tlspersistedsynch.post tid: %s chnl: %d", tid,index)
	err = Ctx.Write(index,claim)
	if err != nil {
		log.Printf("tlspersistedsynch.post tid: %s write failed", tid)	
		return nil, nil, pbmlib.ErrorCode.TRX10
	}	
	response, err := Ctx.Read(index)
	if err != nil {
		log.Printf("tlspersistedsynch.post tid: %s read failed", tid)	
		return nil, nil, pbmlib.ErrorCode.TRX10
	}
	Ctx.ReleaseConnection(index)

	//log.Printf("Response: %s", string(response))

	return response, nil, pbmlib.ErrorCode.TRX00
}
