package tlssynch

import (
	"crypto/tls"
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
	"log"
	"net"
	"strconv"
	"time"
)

func (pc *TLSSyncConnect) Post(claim []byte, header map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {

	var responseBuffer []byte
	bytesRead := 0
	tmp, _ := strconv.Atoi(Cfg.PbmReceiveTimeOut)
	timeOut := time.Duration(float64(tmp) * float64(time.Second))
	tid := "Unknown-TID"
	urlOverride := ""
	if values, ok := header["transmissionId"]; ok && len(values) > 0 {
		tid = values[0]
	}
	if values, ok := header["urlOverride"]; ok && len(values) > 0 {		
	   urlOverride = values[0]
	}
	conn, err := Connect(tid,urlOverride)
	if err != pbmlib.ErrorCode.TRX00 {

		log.Printf("tlssynch.Post tid: %s Connect failed, error: '%s'", tid, err.Message)
		return nil, nil, err
	} else {
		responseBuffer, bytesRead, err = SubmitRequest(string(claim), tid, conn, timeOut) // TODO read from env variables
		if bytesRead <= 0 {
			log.Printf("tlssynch.post tid: %s SubmitRequest failed, error: %s", tid, err.Message)
			return responseBuffer, nil, err
		}
	}
	log.Printf("tlssynch.post tid: %s responsedata(16): %.16s", tid, responseBuffer)
	return responseBuffer, nil, pbmlib.ErrorCode.TRX00
}

func Connect(tid string,urlOverride string) (net.Conn, pbmlib.ErrorInfo) {

	url := Cfg.PbmUrl
	if len(urlOverride)>0 {
		url = urlOverride
	}
	address := url + ":" + Cfg.PbmPort
	log.Printf("tlssynch.connect tid: %s connecting to '%s' Pbm Certificate Insecure Skip Verify: %t", tid, address, Cfg.PbmInsecureSkipVerify)
	// Create a TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: Cfg.PbmInsecureSkipVerify, // You might want to set this to false in production
		ServerName: url,
	}
	// Create a timeout for the connection attempt
	timeout := 5 * time.Second // Adjust the timeout duration as needed
	// Establish a TCP connection to the address
	conn, err := net.DialTimeout("tcp", address, timeout)
	//net.DialTimeout()
	if err != nil {
		log.Printf("tlssynch.connect tid: %s failed, error: '%s'", tid, err)
		return nil, pbmlib.ErrorCode.TRX02
		//return nil,models.ErrorMap
	} else {
		log.Printf("tlssynch.connect tid: %s connected to '%s' SUCCESS", tid, address)
	}
	// Upgrade the connection to TLS
	tlsConn := tls.Client(conn, tlsConfig)
	tlsConn.SetReadDeadline(time.Now().Add(timeout))
	// Handshake with the server
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("tlssynch.connect tls handshake failed tid: %s error: '%s'", tid, err)
		if conn != nil {
			conn.Close()
		}
		return nil, pbmlib.ErrorCode.TRX03
	}
	return tlsConn, pbmlib.ErrorCode.TRX00
}

func SubmitRequest(claim string, tid string, conn net.Conn, timeout time.Duration) ([]byte, int, pbmlib.ErrorInfo) {

	peerAddr := conn.RemoteAddr().String()

	defer conn.Close()

	log.Printf("tlssynch.submitRequest tid: %s data(16) %.16s time-out value: %f seconds url: %s", tid, claim, timeout.Seconds(), peerAddr)
	// Set a read deadline for the connection
	//conn.SetReadDeadline(time.Now().Add(timeout))
	// Send a message to the server
	log.Printf("SLEEPING 200 mseconds")
	time.Sleep(300 * time.Millisecond)
	bytes, err := conn.Write([]byte(claim))
	if err != nil {
		log.Printf("tlssynch.submitRequest tid: %s Write data error: '%s'", tid, err)
		return nil, 0, pbmlib.ErrorCode.TRX10
	} else {
		log.Printf("tlssynch.submitRequest tid: %s Write Snd %d bytes OK", tid, bytes)
	}
	//log.Printf("tlssynch.submitRequest tls.Write %s",string(claim))
	//log.Printf("tlssynch.submitRequest tls.Write %v",claim)
	
	// Receive and print the response from the server
	buffer := make([]byte, PBM_DATA_BUFFER)
	conn.SetReadDeadline(time.Now().Add(timeout))
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Handle the read timeout error
			log.Printf("tlssynch.submitRequest tid: %s Read conn.Read failed timeout error: %s", tid, err)
			return nil, 0, pbmlib.ErrorCode.TRX05
		}
		if bytesRead > 0 { // check this case in case some good data was received
			log.Printf("tlssynch.submitRequest tid: %s Read.error raised but bytesRead > 0 error: %s bytesRead: %d", tid, err, bytesRead)
		} else {
			log.Printf("tlssynch.submitRequest tid: %s Read failed error: %s url: %s", tid, err, peerAddr)
			return nil, 0, pbmlib.ErrorCode.TRX10
		}
	}
	log.Printf("tlssynch.submitRequest tid: %s Rcvd: %d bytes", tid, bytesRead)
	responseBuffer := make([]byte, bytesRead)
	copy(responseBuffer, buffer[:bytesRead])
	return responseBuffer, bytesRead, pbmlib.ErrorCode.TRX00
}
