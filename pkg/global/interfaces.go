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
	//Connect(host string, port string, timeOut string, disconnectAfterResponse bool) (transaction.ErrorInfo) 
	Post(host string, port string, timeOut string, disconnectAfterResponse bool,claim []byte, header map[string][]string, timeout time.Duration) ([]byte, map[string][]string, transaction.ErrorInfo)
	//Test(claim []byte) ([]byte, transaction.ErrorInfo)
	Close() error
}

