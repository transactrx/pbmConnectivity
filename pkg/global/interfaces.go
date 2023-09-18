package pbmconnectivitylib


import (
	"time"
)

type PBM interface {
	Start() error
	Post(clm Claim, header map[string][]string, timeout time.Duration) ([]byte, map[string][]string, ErrorInfo)
	Test(claim []byte) ([]byte, ErrorInfo)
	Shutdown() error
}