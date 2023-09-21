package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/transactrx/pbmConnectivity/pkg/global"
	"github.com/transactrx/pbmConnectivity/pkg/tlssynch"
	//"github.com/transactrx/rxtransactionmodels/pkg/transaction"
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

func main() {

	log.Printf("####################################################")
	log.Printf("### PBMConnect Interface Example Using TLS Synch ###")
	log.Printf("####################################################")
	var tlsCon global.PBMConnect = &tlssynch.TLSSyncConnect{}
	config := make(map[string]interface{})
	config["pbmUrl"] = "10.0.120.250"
	config["pbmPort"] = "5845"
	config["pbmReceiveTimeOut"] = "8"
	tlsCon.Start(config)
	response,_,err := tlsCon.Post([]byte("TESTING LIBRARY..."),nil)
	if(err!=pbmlib.ErrorCode.TRX00){
		log.Printf("tlsCon.Post failed: '%v'",err)
	}else{
		log.Printf("examplePBM response: '%s'",response)
	}
	c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt)
    go func() {
        select {
        case sig := <-c:
            log.Printf("Got %s signal. Aborting...\n", sig)
            os.Exit(1)
        }
    }()
}