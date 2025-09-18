package https

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	"strings"
	"time"
)

var CustomHttpClient *fasthttp.Client

func CreateGlobalHttpContext() {
	CustomHttpClient = &fasthttp.Client{
		TLSConfig: &tls.Config{InsecureSkipVerify: Cfg.PbmInsecureSkipVerify}, // Accept invalid certs
		//MaxConnsPerHost: 100,  // Adjust based on expected traffic
	}

}

func FastPost(body []byte, conf RouteInfo, url string) (string, int, error) {
	// Initialize fasthttp.Request and fasthttp.Response
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	//log.Printf("FastPost (final) route:%s url:%s", conf.RouteCode, url)
	// Set request URL and method
	//if IsDebugMode() {
	log.Printf("FastPost route: %s sending to url: %s", conf.RouteCode, url)
	//}
	req.SetRequestURI(url)
	req.Header.SetMethod("POST")

	// Set timeout if specified
	if conf.Timeout > 0 {
		req.SetTimeout(time.Duration(conf.Timeout) * time.Second)
	}
	var err error
	for _, header := range conf.Headers {
		readyHeader := header.Value
		if IsDebugMode() {
			log.Printf("Fastpost header  %s: %s prefix: '%s' Base64encode: %t", header.Key, readyHeader, header.Prefix, header.Base64encode)
		}
		if header.Base64encode {
			readyHeader, err = encodeAuthorization(header.Value, header.Prefix)
			if err != nil {
				return fmt.Sprintf("Fastpost failed encoding api key for route code: %s", conf.RouteCode), 401, err
			}
		} else if len(header.Prefix) > 0 {
			readyHeader = header.Prefix + " " + readyHeader
		}
		if IsDebugMode() {
			log.Printf("Fastpost header  readyHeader: %s", readyHeader)
		}
		req.Header.Set(header.Key, readyHeader)
	}

	if len(req.Header.ContentType()) == 0 {
		log.Printf("Fastpost route code %s defaulting the content type to application/EDI-NCPDP as it was not provided", conf.RouteCode)
		req.Header.SetContentType("application/EDI-NCPDP")
	}
	req.SetBody(body)
	// Perform the request
	err = CustomHttpClient.DoTimeout(req, resp, time.Duration(conf.Timeout*float64(time.Second)))
	if err != nil {
		log.Printf("route code %s error sending request: %v", conf.RouteCode, err)
		return "Fastpost failed sending request", resp.StatusCode(), err
	}
	statusCode := resp.StatusCode()
	log.Printf("Fastpost route %s http response code: %d", conf.RouteCode, statusCode)
	return string(resp.Body()), statusCode, nil
}

func encodeAuthorization(auth, prefix string) (string, error) {
	stringToEncode := auth
	if strings.HasPrefix(auth, prefix) {
		parts := strings.SplitN(auth, " ", 2) // Use SplitN to ensure splitting into two parts: "Bearer" and token
		if len(parts) != 2 || parts[0] != prefix {
			log.Printf("Invalid APIKey format")
			return "", fmt.Errorf("invalid API Key format")
		}
		stringToEncode = parts[1]
	}

	// Encode the second part of the part slice, which is the token
	encodedString := EncodeToBase64(stringToEncode)
	// Return the concatenation of "Bearer" with the encoded string
	if len(prefix) > 0 {
		encodedString = prefix + " " + encodedString
	}
	return encodedString, nil
}

// EncodeToBase64 takes a string as input and returns its base64 encoded string
func EncodeToBase64(input string) string {
	// Convert string to byte slice
	data := []byte(input)
	// Encode to base64
	encodedString := base64.StdEncoding.EncodeToString(data)
	return encodedString
}
