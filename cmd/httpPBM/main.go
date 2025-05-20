package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"
	"github.com/transactrx/pbmConnectivity/pkg/global"
	"github.com/transactrx/pbmConnectivity/pkg/https"
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("####################################################")
	log.Printf("### PBMConnect Interface Example Using https ###")
	log.Printf("####################################################")
	config := make(map[string]interface{})
	const HEADER_CHECK_OFFSET = 19
	const HEADER_CHECK_LEN = 20
	config["pbmUrl"] = "https://messaging2.qs1.com/erx/v2017071/eMar"
	config["pbmPort"] = "20000"
	config["pbmReceiveTimeOut"] = "10"
	config["pbmQueueTimeOut"] = "10"
	config["pbmInsecureSkipVerify"] = true
	config["pbmOutboundChnls"] = "1"
	config["pbmActiveSites"] = "true"
	config["headerCheck"] = true
	config["headerCheckOffset"] = strconv.Itoa(HEADER_CHECK_OFFSET)
	config["HeaderCheckLen"] = strconv.Itoa(HEADER_CHECK_LEN)
	config["endOfRecordChar"] = "LEN"
	config["msgLenOffset"] = strconv.Itoa(6) // zero based offset
	config["msgLenWidth"] = strconv.Itoa(5)  // ASCII right justified len
	config["debugEnabled"] = true

	routeInfo := https.RouteInfo{
		RouteCode: "301",
		//PbmUrl:    "https://messaging2.qs1.com/erx/v2017071/eMar",
		PbmUrl:    "https://72.34.195.16/erxtest/v2017071/eMar",		 		
		Headers:   nil,
		Timeout:   5, // in seconds 
	}

	tlsSync := https.HTTPPBMConnect{
		Conf: routeInfo,		
		
	}
	//var tlsCon global.PBMConnectWithStats = &https.HTTPPBMConnect{}
	var tlsCon global.PBMConnectWithStats = &tlsSync

	tlsCon.Start(config)
	header := map[string][]string{
		"transmissionId": {"123456789"},
		"Content-Type": {"text/xml"},
		//"Authorization": {""},
	}

	time.Sleep(time.Second * 2)

	claim := "004336D0B1BCTX      1076000100        20220418FAMULUS   AM04C2123456789C61C90CCTESTCDTESTAM01C419000501C52C701CATESTCBCLAIMCM123 ANY STREETCNFORT WORTHCOTXCP76102CX01CY0000000004X01AM07EM1D27418529E103D768727010001U701C800D300D5030D61D81DE20210210DF06DI00DJ4E70000540000EU0028MLAM11D90183936{DN01DQ0183936{DU0183936{AM032JDIANE2K2160 TEST ADDY2MFORT WORTH2NTX2P76107EZ01"
	//claim := "12344343"

	go func() {
		response, _, err := tlsCon.Post([]byte(claim), header)
		if err != pbmlib.ErrorCode.TRX00 {
			log.Printf("tlsCon.post failed: '%v'", err)
		} else {
			log.Printf("examplePBM response: '%s'", response)
		}
	}()

	stats := make(map[string]interface{})
	stats["sessionCount"] = 0

	for {
		log.Printf("Main...")
		time.Sleep(time.Second * 10)
		err1 := tlsCon.GetStats(stats)
		if err1 != nil {
			log.Printf("GetStats failed")
		} else {
			log.Printf("Getstats count: %s", stats["sessionCount"].(string))
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
