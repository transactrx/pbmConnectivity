package socketsynch

import (
	"log"
	"net"
	"strconv"
	"time"

	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

func (pc *SocketSynchConnect) Post(claim []byte, header map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {

	var responseBuffer []byte
	bytesRead := 0
	tmp, _ := strconv.Atoi(pc.Cfg.PbmReceiveTimeOut)
	timeOut := time.Duration(float64(tmp) * float64(time.Second))
	tid := "Unknown-TID"
	urlOverride := ""
	sessionLogin := "none"
	loginData := ""
	isSessionLoggedIn := false
	skipFirstTwoBytes := false
	var site *Site = nil

	if values, ok := header["transmissionId"]; ok && len(values) > 0 {
		tid = values[0]
	}
	if values, ok := header["urlOverride"]; ok && len(values) > 0 {
		urlOverride = values[0]
	} else {
		urlOverride, site = GetNextUrl()
	}
	if values, ok := header["sessionLogin"]; ok && len(values) > 0 {
		sessionLogin = values[0]
		if sessionLogin == "skipFirstTwoBytes" {
			skipFirstTwoBytes = true
		}
	}
	if values, ok := header["loginData"]; ok && len(values) > 0 {
		loginData = values[0]
	}
	conn, err := Connect(tid, urlOverride, pc.Cfg)
	if err != pbmlib.ErrorCode.TRX00 {

		log.Printf("socketsynch.Post tid: %s Connect failed, error: '%s'", tid, err.Message)
		if site != nil {
			site.failedClaims.Add(1)
		}
		DecreaseActiveClaims(site)
		return nil, nil, err
	} else {
		if sessionLogin == "sendLoginData" {
			log.Printf("socketsynch.Post tid: %s sending session login data len: %d", tid, len(loginData))
			isSessionLoggedIn, bytesRead, err = SubmitLoginData(loginData, tid, conn, time.Duration(5*float64(time.Second)))
			// submit login Data and verify response
			if !isSessionLoggedIn {
				log.Printf("socketsynch.Post tid: %s sending session login data failed", tid)
				DecreaseActiveClaims(site)
				return nil, nil, err
			}
		}
		responseBuffer, bytesRead, err = SubmitRequest(string(claim), tid, conn, timeOut, skipFirstTwoBytes) // TODO read from env variables
		if bytesRead <= 0 {
			log.Printf("socketsynch.post tid: %s SubmitRequest failed, error: %s", tid, err.Message)
			if site != nil {
				site.failedClaims.Add(1)
			}
			DecreaseActiveClaims(site)
			return responseBuffer, nil, err
		}

		if !IsValidResponse(responseBuffer, header) {
			return responseBuffer, nil, pbmlib.ErrorCode.TRX11
		}
	}
	log.Printf("socketsynch.post tid: %s responsedata(16): %.16s", tid, responseBuffer)
	DecreaseActiveClaims(site)
	return responseBuffer, nil, pbmlib.ErrorCode.TRX00
}

func IsValidResponse(response []byte, reqHeader map[string][]string) bool {
	reqTracking := ""
	headerCheckOffset := 0
	headerCheckLen := 0

	if values, ok := reqHeader["headerValueToCheck"]; ok && len(values) > 0 {
		reqTracking = values[0]
	}
	if values, ok := reqHeader["headerCheckOffset"]; ok && len(values) > 0 {
		headerCheckOffset, _ = strconv.Atoi(values[0])
	}
	if values, ok := reqHeader["headerCheckLen"]; ok && len(values) > 0 {
		headerCheckLen, _ = strconv.Atoi(values[0])
	}

	if len(response) < headerCheckOffset+headerCheckLen {
		log.Printf("ValidateResponse failed mismatch FULLresp: '%s' requestTracking: '%s'", string(response), reqTracking)
		return false
	}

	if headerCheckLen == 0 {
		// nothing to check, so move on
		// log.Printf("Tracking Length is zero.")
		return true
	}

	rspTracking := response[headerCheckOffset : headerCheckOffset+headerCheckLen]
	if reqTracking != string(rspTracking) {
		log.Printf("ValidateResponse failed mismatch respTracking: '%s' requestTracking: '%s'", rspTracking, reqTracking)
		return false
	}

	//log.Printf("Validated Tracking: request '%s' & response '%s'", reqTracking, rspTracking)
	
	return true
}

func DecreaseActiveClaims(site *Site) {
	if site != nil && site.activeClaims.Load() > 0 {
		site.activeClaims.Add(-1)
	}

}

func Connect(tid string, urlOverride string, cfg Config) (net.Conn, pbmlib.ErrorInfo) {

	url := cfg.PbmUrl
	var err error
	if len(urlOverride) > 0 {
		url = urlOverride
	}
	address := url // + ":" + cfg.PbmPort
	log.Printf("socketsynch.connect tid: %s connecting to '%s'", tid, address)
	start := time.Now()        // Capture the start time
	timeout := 5 * time.Second // Adjust the timeout duration as needed
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		log.Printf("socketsynch.connect tid: %s failed, error: '%s'", tid, err)
		return nil, pbmlib.ErrorCode.TRX02
	}
	elapsed := time.Since(start) // Calculate elapsed time
	log.Printf("socketsynch.connect tid: %s ok tcp handshake duration: %d ms url: %s", tid, elapsed.Milliseconds(), address)
	//log.Printf("socketsynch.connect tid: %s tcp handshake duration: %d ms url: %s", tid,elapsed.Milliseconds(),address)
	return conn, pbmlib.ErrorCode.TRX00
}

func SubmitRequest(claim string, tid string, conn net.Conn, timeout time.Duration, skipFirstTwoBytes bool) ([]byte, int, pbmlib.ErrorInfo) {

	//var keepAlive = []byte{0x02,0x30}
	peerAddr := conn.RemoteAddr().String()
	defer conn.Close()
	log.Printf("socketsynch.submitRequest tid: %s data(16) %.16s time-out value: %f seconds url: %s skipData: %t", tid, claim, timeout.Seconds(), peerAddr, skipFirstTwoBytes)
	bytes, err := conn.Write([]byte(claim))
	if err != nil {
		log.Printf("socketsynch.submitRequest tid: %s Write data error: '%s'", tid, err)
		return nil, 0, pbmlib.ErrorCode.TRX10
	} else {
		log.Printf("socketsynch.submitRequest tid: %s Write Snd %d bytes OK", tid, bytes)
	}
	buffer := make([]byte, PBM_DATA_BUFFER)
	conn.SetReadDeadline(time.Now().Add(timeout))
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Handle the read timeout error
			log.Printf("socketsynch.submitRequest tid: %s Read conn.Read failed timeout error: %s", tid, err)
			return nil, 0, pbmlib.ErrorCode.TRX05
		}
		if bytesRead > 0 { // check this case in case some good data was received
			log.Printf("socketsynch.submitRequest tid: %s Read.error raised but bytesRead > 0 error: %s bytesRead: %d", tid, err, bytesRead)
		} else {
			log.Printf("socketsynch.submitRequest tid: %s Read failed error: %s url: %s", tid, err, peerAddr)
			return nil, 0, pbmlib.ErrorCode.TRX10
		}
	}
	//	log.Printf("socketsynch.submitRequest data: %x",buffer[:bytesRead])
	log.Printf("socketsynch.submitRequest tid: %s Rcvd: %d bytes", tid, bytesRead)
	responseBuffer := make([]byte, bytesRead)
	copy(responseBuffer, buffer[:bytesRead])
	return responseBuffer, bytesRead, pbmlib.ErrorCode.TRX00
}

func SubmitLoginData(loginData string, tid string, conn net.Conn, timeout time.Duration) (bool, int, pbmlib.ErrorInfo) {

	retValue := false
	peerAddr := conn.RemoteAddr().String()
	log.Printf("socketsynch.SubmitLoginData tid: %s data(16) %.16X time-out value: %f seconds url: %s", tid, loginData, timeout.Seconds(), peerAddr)
	bytes, err := conn.Write([]byte(loginData))
	if err != nil {
		log.Printf("socketsynch.SubmitLoginData tid: %s Write data error: '%s'", tid, err)
		return retValue, 0, pbmlib.ErrorCode.TRX10
	} else {
		log.Printf("socketsynch.SubmitLoginData tid: %s Write Snd %d bytes OK", tid, bytes)
	}
	buffer := make([]byte, PBM_DATA_BUFFER)
	conn.SetReadDeadline(time.Now().Add(timeout))
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("socketsynch.SubmitLoginData tid: %s Read conn.Read failed timeout error: %s", tid, err)
			return retValue, 0, pbmlib.ErrorCode.TRX03
		}
		return retValue, 0, pbmlib.ErrorCode.TRX03
	}
	retValue = true
	log.Printf("socketsynch.SubmitLoginData tid: %s Rcvd: %d bytes data(16) %.16X", tid, bytesRead, buffer)

	return retValue, bytesRead, pbmlib.ErrorCode.TRX00
}
