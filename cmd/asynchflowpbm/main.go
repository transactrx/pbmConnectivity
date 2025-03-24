package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/transactrx/pbmConnectivity/pkg/global"
	"github.com/transactrx/pbmConnectivity/pkg/asynchflow"
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

func main() {

	log.Printf("####################################################")
	log.Printf("### PBMConnect Interface Example Using ASynch Flow ###")
	log.Printf("####################################################")
	var tlsCon global.PBMConnectWithStats = &asynchflow.AsynchFlow{}
	config := make(map[string]interface{})

	// prime testing over tls 10.0.205.1:26301

	const HEADER_CHECK_OFFSET = 11
	const HEADER_CHECK_LEN = 10

	config["pbmUrl"] = "10.0.120.250"
	config["pbmPort"] = "30009"
	config["pbmReceiveTimeOut"] = "10"
	config["pbmQueueTimeOut"] = "10"
	config["pbmInsecureSkipVerify"] = true
	config["pbmOutboundChnls"] = "1"
	config["pbmActiveSites"] = "true"
	config["headerCheck"] = true
	config["headerCheckOffset"] = strconv.Itoa(HEADER_CHECK_OFFSET)
	config["HeaderCheckLen"] = strconv.Itoa(HEADER_CHECK_LEN)
	config["endOfRecordChar"] = "ETX"
	config["msgLenOffset"] = strconv.Itoa(6) // zero based offset 
	config["msgLenWidth"] = strconv.Itoa(5)  // ASCII right justified len
	config["debugEnabled"] = false
	config["MessageLenType"] = 0
	

	tlsCon.Start(config)
	header := map[string][]string{
		"transmissionId": {"123456789"},
		"headerValueToCheck": {"QS12345678"},
	}

	time.Sleep(time.Second * 10)

	claim := "M0000100426QS123456781004336D0B1BCTX      1076000100        20220418FAMULUS   AM04C2123456789C61C90CCTESTCDTESTAM01C419000501C52C701CATESTCBCLAIMCM123 ANY STREETCNFORT WORTHCOTXCP76102CX01CY0000000004X01AM07EM1D27418529E103D768727010001U701C800D300D5030D61D81DE20210210DF06DI00DJ4E70000540000EU0028MLAM11D90183936{DN01DQ0183936{DU0183936{AM032JDIANE2K2160 TEST ADDY2MFORT WORTH2NTX2P76107EZ01"

	go func() {
		response, _, err := tlsCon.Post([]byte(claim), header)
		if err != pbmlib.ErrorCode.TRX00 {
			log.Printf("tlsCon.post failed: '%v'", err)
		} else {
			log.Printf("asynchflow response: '%s'", response)
		}
	}()
	
	// go func() {
	// 	response, _, err := tlsCon.Post([]byte(claim), header)
	// 	if err != pbmlib.ErrorCode.TRX00 {
	// 		log.Printf("tlsCon.post failed: '%v'", err)
	// 	} else {
	// 		log.Printf("asynchflow response: '%s'", response)
	// 	}
	// }()


	stats := make(map[string]interface{})
	stats["sessionCount"] = 0

	for {
		log.Printf("Main...")
		time.Sleep(time.Second * 10)
		err1 := tlsCon.GetStats(stats)
		if err1 != nil {
			log.Printf("GetStats failed")
		}else{
			log.Printf("Getstats count: %s",stats["sessionCount"].(string))
		}
		//tlsCon.Close()

	}
	//select{}

	func() {
		response, _, err := tlsCon.Post([]byte("<HEADER><DATA>"), header)
		if err != pbmlib.ErrorCode.TRX00 {
			log.Printf("tlsCon.post failed: '%v'", err)
		} else {
			log.Printf("examplePBM response: '%s'", response)
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
