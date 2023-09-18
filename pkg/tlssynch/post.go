package tlssynch

import (
	"crypto/tls"
	"log"
	"net"
	"time"

	"github.com/transactrx/rxtransactionmodels/pkg/transaction"
)

var counter int32 = 0

const PBM_DATA_BUFFER = 16384

func (pc *TLSSyncConnect) Post(claim []byte, header map[string][]string, timeOut time.Duration) ([]byte, map[string][]string, transaction.ErrorInfo) {

	conn, err := Connect()
	var responseBuffer []byte
	bytesRead := 0
	if err != transaction.ErrorCode.TRX00 {

		log.Printf("Pbm.RouteTransaction ConnectToPbm failed, error: '%s'", err.Message)
		//response.BuildResponseError(claim, errorCode, startTime)
		return nil, nil, err
	} else {
		responseBuffer, bytesRead, err = SubmitRequest(string(claim), conn, timeOut*time.Second) // TODO read from env variables
		if bytesRead <= 0 {
			log.Printf("Pbm.RouteTransaction Post failed, error: %s", err.Message)
			return responseBuffer, nil, err
		}
	}
	return responseBuffer, nil, transaction.ErrorCode.TRX00
}

func Connect() (net.Conn, transaction.ErrorInfo) {

	// Combine host and port into an address
	address := Cfg.PbmUrl + ":" + Cfg.PbmPort
	log.Printf("tlssynch.Connect connecting to '%s'", address)

	// Create a TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // You might want to set this to false in production
	}
	// Establish a TCP connection to the address
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("tlssynch.net.Dial failed, error: '%s'", err)
		return nil, transaction.ErrorCode.TRX02
		//return nil,models.ErrorMap
	} else {
		log.Printf("tlssynch.net.Dial connected to '%s' SUCCESS", address)
	}
	// Upgrade the connection to TLS
	tlsConn := tls.Client(conn, tlsConfig)
	// Handshake with the server
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("tlssynch TLS Handshake error: '%s'", err)
		if conn != nil {
			conn.Close()
		}
		//return nil, pbmlib.ErrorCode.TRX03
	}
	return conn, transaction.ErrorCode.TRX00
}

func SubmitRequest(claim string, conn net.Conn, timeout time.Duration) ([]byte, int, transaction.ErrorInfo) {

	//defer conn.Close()

	log.Printf("Post data(16) %.16s time-out value: %f seconds", claim, timeout.Seconds())
	// Set a read deadline for the connection
	conn.SetReadDeadline(time.Now().Add(timeout))

	// Send a message to the server
	bytes, err := conn.Write([]byte(claim))
	if err != nil {
		log.Printf("Post.error sending data error: '%s'", err)
		return nil, 0, transaction.ErrorCode.TRX10
	} else {
		log.Printf("Post.Snd %d bytes OK", bytes)
	}

	// Receive and print the response from the server
	buffer := make([]byte, PBM_DATA_BUFFER)
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Handle the read timeout error
			log.Printf("Post.Error conn.Read timeout error: %s", err)
			return nil, 0, transaction.ErrorCode.TRX05
		}
		log.Printf("Post.Error conn.Read failed error: %s", err)
		return nil, 0, transaction.ErrorCode.TRX10
	}
	log.Printf("Post.Rcvd %d bytes OK", bytesRead)
	responseBuffer := make([]byte, bytesRead)
	copy(responseBuffer, buffer[:bytesRead])
	return responseBuffer, bytesRead, transaction.ErrorCode.TRX00
}
