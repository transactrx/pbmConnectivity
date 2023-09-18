package global

import (
	"time"
	"github.com/transactrx/rxtransactionmodels/pkg/transaction"
)

type Claim struct{
	tid string
}


type PBMConnect interface {
	Start(config map[string]interface{}) error
	Post(claim []byte, header map[string][]string, timeOut time.Duration) ([]byte, map[string][]string, transaction.ErrorInfo) 
	//Test(claim []byte) ([]byte, transaction.ErrorInfo)
	Close() error
}
