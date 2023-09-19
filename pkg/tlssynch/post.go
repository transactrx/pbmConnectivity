package tlssynch

import (
	"crypto/tls"
	"log"
	"net"
	"strconv"
	"time"
	//"github.com/transactrx/rxtransactionmodels/pkg/transaction"
	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

const PBM_DATA_BUFFER = 16384
func (pc *TLSSyncConnect) Post(claim []byte, header map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {

	conn, err := Connect()
	var responseBuffer []byte
	bytesRead := 0
	tmp,_ := strconv.Atoi(Cfg.PbmReceiveTimeOut)
	timeOut := time.Duration(float64(tmp)*float64(time.Second))
	
	if err != pbmlib.ErrorCode.TRX00 {

		log.Printf("TLSSyncConnect.Post Connect failed, error: '%s'", err.Message)
		//response.BuildResponseError(claim, errorCode, startTime)
		return nil, nil, err
	} else {
		responseBuffer, bytesRead, err = SubmitRequest(string(claim), conn,timeOut) // TODO read from env variables
		if bytesRead <= 0 {
			log.Printf("TLSSyncConnect.Post SubmitRequest failed, error: %s", err.Message)
			return responseBuffer, nil, err
		}
	}
	log.Printf("TLSSyncConnect.Post response: '%s'",responseBuffer)
	return responseBuffer, nil, pbmlib.ErrorCode.TRX00
}

func Connect() (net.Conn, pbmlib.ErrorInfo) {

	// Combine host and port into an address
	address := Cfg.PbmUrl + ":" + Cfg.PbmPort
	log.Printf("TLSSyncConnect.Connect connecting to '%s'", address)
	// Create a TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // You might want to set this to false in production
	}
	// Establish a TCP connection to the address
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("TLSSyncConnect.net.Dial failed, error: '%s'", err)
		return nil, pbmlib.ErrorCode.TRX02
		//return nil,models.ErrorMap
	} else {
		log.Printf("TLSSyncConnect.net.Dial connected to '%s' SUCCESS", address)
	}
	// Upgrade the connection to TLS
	tlsConn := tls.Client(conn, tlsConfig)
	// Handshake with the server
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("TLSSyncConnect TLS Handshake error: '%s'", err)
		if conn != nil {
			conn.Close()
		}
		//return nil, pbmlib.ErrorCode.TRX03
	}
	return tlsConn, pbmlib.ErrorCode.TRX00
}

func SubmitRequest(claim string, conn net.Conn, timeout time.Duration) ([]byte, int, pbmlib.ErrorInfo) {

	defer conn.Close()

	log.Printf("TLSSyncConnect SubmitRequest data(16) %.16s time-out value: %f seconds", claim, timeout.Seconds())
	// Set a read deadline for the connection
	conn.SetReadDeadline(time.Now().Add(timeout))
	// Send a message to the server
	bytes, err := conn.Write([]byte(claim))
	if err != nil {
		log.Printf("TLSSyncConnect.SubmitRequest Write data error: '%s'", err)
		return nil, 0, pbmlib.ErrorCode.TRX10
	} else {
		log.Printf("TLSSyncConnect.SubmitRequest Write Snd %d bytes OK", bytes)
	}
	// Receive and print the response from the server
	buffer := make([]byte, PBM_DATA_BUFFER)
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Handle the read timeout error
			log.Printf("TLSSyncConnect.SubmitRequest Read conn.Read failed timeout error: %s", err)
			return nil, 0, pbmlib.ErrorCode.TRX05
		}
		log.Printf("TLSSyncConnect.SubmitRequest Read failed error: %s", err)
		return nil, 0, pbmlib.ErrorCode.TRX10
	}
	log.Printf("TLSSyncConnect.SubmitRequest Rcvd: %d bytes", bytesRead)
	responseBuffer := make([]byte, bytesRead)
	copy(responseBuffer, buffer[:bytesRead])
	return responseBuffer, bytesRead, pbmlib.ErrorCode.TRX00
}



