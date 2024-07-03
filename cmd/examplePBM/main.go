package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/transactrx/pbmConnectivity/pkg/global"
	//"github.com/transactrx/pbmConnectivity/pkg/tlssynch"
	"github.com/transactrx/pbmConnectivity/pkg/tlspersistedsynch"
	//"github.com/transactrx/rxtransactionmodels/pkg/transaction"
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

func main() {

	log.Printf("####################################################")
	log.Printf("### PBMConnect Interface Example Using TLS Synch ###")
	log.Printf("####################################################")
	var tlsCon global.PBMConnect = &tlspersistedsynch.TLSPersistedSyncConnect{}
	config := make(map[string]interface{})

	// prime testing over tls 10.0.205.1:26301

	config["pbmUrl"] = "10.0.205.1"
	config["pbmPort"] = "26301"
	config["pbmReceiveTimeOut"] = "20"
	config["pbmInsecureSkipVerify"] = true
	config["pbmTotalChnls"] = "1"
	tlsCon.Start(config)
	header := map[string][]string{
		"transmissionId":{"123456789"},
	}

	time.Sleep(time.Second * 5)


    claim := "0210032   000019                       D0B11A076000100        20230316AM25C2123456789AM29CABUBOCHKACBFOMINC419690609AM21ANRF3241025545163007968FA03FB05FBCYFB25AM22EM1D21"


	go func(){
		response,_,err := tlsCon.Post([]byte(claim),header)
		if(err!=pbmlib.ErrorCode.TRX00){
			log.Printf("tlsCon.post failed: '%v'",err)
		}else{
			log.Printf("examplePBM response: '%s'",response)
		}
	}()


	for{
		log.Printf("Main...")
		time.Sleep(time.Second * 5)
	}
	
	func(){
		response,_,err := tlsCon.Post([]byte("<HEADER><DATA>"),header)
		if(err!=pbmlib.ErrorCode.TRX00){
			log.Printf("tlsCon.post failed: '%v'",err)
		}else{
			log.Printf("examplePBM response: '%s'",response)
		}
	}()
	c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt)
    go func() {
        select {
        case sig := <-c:
            log.Printf("PBMConnect shutdown %s signal. Aborting...\n", sig)
            os.Exit(1)
        }
    }()
}