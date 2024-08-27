package global

import (
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)
type PBMConnect interface {
	Start(config map[string]interface{}) error
	Post(claim []byte, header map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) 
	//Test(claim []byte) ([]byte, transaction.ErrorInfo)
	Close() error
}

type PBMConnectWithStats interface {
    PBMConnect
    GetStats(map[string]interface{}) error
}