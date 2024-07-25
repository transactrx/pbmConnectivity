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

	config["pbmUrl"] = "10.0.205.1,10.0.2.32"
	config["pbmPort"] = "26301"
	config["pbmReceiveTimeOut"] = "25"
	config["pbmQueueTimeOut"] = "20"
	config["pbmInsecureSkipVerify"] = true
	config["pbmOutboundChnls"] = "1"
	config["pbmActiveSites"] = "true,false"
	
	tlsCon.Start(config)
	header := map[string][]string{
		"transmissionId":{"123456789"},
	}

	time.Sleep(time.Second * 2)


    claim := "011552D0B1BCTX      1076000100        20220418FAMULUS   AM04C2123456789C61C90CCTESTCDTESTAM01C419000501C52C701CATESTCBCLAIMCM123 ANY STREETCNFORT WORTHCOTXCP76102CX01CY0000000004X01AM07EM1D27418529E103D768727010001U701C800D300D5030D61D81DE20210210DF06DI00DJ4E70000540000EU0028MLAM11D90183936{DN01DQ0183936{DU0183936{AM032JDIANE2K2160 TEST ADDY2MFORT WORTH2NTX2P76107EZ01DB1234567891DRTESTPHY"


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