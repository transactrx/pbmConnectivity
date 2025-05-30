package https

import (
	"fmt"
	"testing"

	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

func TestMapHTTPStatusToTRXCode(t *testing.T) {
	tests := []struct {
		name         string
		httpCode     int
		expectedTRX  pbmlib.ErrorInfo
		expectedError bool
	}{
		// 2xx Success codes
		{
			name:          "200 OK",
			httpCode:      200,
			expectedTRX:   pbmlib.ErrorCode.TRX00,
			expectedError: false,
		},
		{
			name:          "201 Created",
			httpCode:      201,
			expectedTRX:   pbmlib.ErrorCode.TRX00,
			expectedError: false,
		},
		{
			name:          "299 Success boundary",
			httpCode:      299,
			expectedTRX:   pbmlib.ErrorCode.TRX00,
			expectedError: false,
		},
		
		// 3xx Redirect codes
		{
			name:          "300 Multiple Choices",
			httpCode:      300,
			expectedTRX:   pbmlib.ErrorCode.TRX02,
			expectedError: true,
		},
		{
			name:          "301 Moved Permanently",
			httpCode:      301,
			expectedTRX:   pbmlib.ErrorCode.TRX02,
			expectedError: true,
		},
		{
			name:          "399 Redirect boundary",
			httpCode:      399,
			expectedTRX:   pbmlib.ErrorCode.TRX02,
			expectedError: true,
		},
		
		// 4xx Client Error codes
		{
			name:          "400 Bad Request",
			httpCode:      400,
			expectedTRX:   pbmlib.ErrorCode.TRX04,
			expectedError: true,
		},
		{
			name:          "401 Unauthorized",
			httpCode:      401,
			expectedTRX:   pbmlib.ErrorCode.TRX03,
			expectedError: true,
		},
		{
			name:          "403 Forbidden",
			httpCode:      403,
			expectedTRX:   pbmlib.ErrorCode.TRX09,
			expectedError: true,
		},
		{
			name:          "404 Not Found",
			httpCode:      404,
			expectedTRX:   pbmlib.ErrorCode.TRX04,
			expectedError: true,
		},
		{
			name:          "499 Client error boundary",
			httpCode:      499,
			expectedTRX:   pbmlib.ErrorCode.TRX04,
			expectedError: true,
		},
		
		// 5xx Server Error codes
		{
			name:          "500 Internal Server Error",
			httpCode:      500,
			expectedTRX:   pbmlib.ErrorCode.TRX07,
			expectedError: true,
		},
		{
			name:          "502 Bad Gateway",
			httpCode:      502,
			expectedTRX:   pbmlib.ErrorCode.TRX07,
			expectedError: true,
		},
		{
			name:          "599 Server error boundary",
			httpCode:      599,
			expectedTRX:   pbmlib.ErrorCode.TRX07,
			expectedError: true,
		},
		
		// Edge cases
		{
			name:          "100 Continue",
			httpCode:      100,
			expectedTRX:   pbmlib.ErrorCode.TRX9999,
			expectedError: true,
		},
		{
			name:          "600 Unknown",
			httpCode:      600,
			expectedTRX:   pbmlib.ErrorCode.TRX9999,
			expectedError: true,
		},
		{
			name:          "0 Invalid",
			httpCode:      0,
			expectedTRX:   pbmlib.ErrorCode.TRX9999,
			expectedError: true,
		},
		{
			name:          "999 High value",
			httpCode:      999,
			expectedTRX:   pbmlib.ErrorCode.TRX9999,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trxCode, isError := MapHTTPStatusToTRXCode(tt.httpCode)
			if trxCode != tt.expectedTRX {
				t.Errorf("MapHTTPStatusToTRXCode(%d) TRX = %v, want %v", tt.httpCode, trxCode, tt.expectedTRX)
			}
			if isError != tt.expectedError {
				t.Errorf("MapHTTPStatusToTRXCode(%d) isError = %v, want %v", tt.httpCode, isError, tt.expectedError)
			}
		})
	}
}

func TestMapHTTPStatusToTRXCodeBoundaries(t *testing.T) {
	// Test exact boundaries
	boundaryTests := []struct {
		code     int
		expected pbmlib.ErrorInfo
		isError  bool
	}{
		{199, pbmlib.ErrorCode.TRX9999, true},  // Just before 2xx
		{200, pbmlib.ErrorCode.TRX00, false},   // Start of 2xx
		{299, pbmlib.ErrorCode.TRX00, false},   // End of 2xx
		{300, pbmlib.ErrorCode.TRX02, true},    // Start of 3xx
		{399, pbmlib.ErrorCode.TRX02, true},    // End of 3xx
		{400, pbmlib.ErrorCode.TRX04, true},    // Start of 4xx
		{499, pbmlib.ErrorCode.TRX04, true},    // End of 4xx
		{500, pbmlib.ErrorCode.TRX07, true},    // Start of 5xx
		{599, pbmlib.ErrorCode.TRX07, true},    // End of 5xx
		{600, pbmlib.ErrorCode.TRX9999, true},  // Just after 5xx
	}

	for _, tt := range boundaryTests {
		t.Run(fmt.Sprintf("boundary_%d", tt.code), func(t *testing.T) {
			trxCode, isError := MapHTTPStatusToTRXCode(tt.code)
			if trxCode != tt.expected {
				t.Errorf("MapHTTPStatusToTRXCode(%d) = %v, want %v", tt.code, trxCode, tt.expected)
			}
			if isError != tt.isError {
				t.Errorf("MapHTTPStatusToTRXCode(%d) isError = %v, want %v", tt.code, isError, tt.isError)
			}
		})
	}
}

func TestMapHTTPStatusToTRXCodeSpecialCases(t *testing.T) {
	// Test specific HTTP status codes that have special TRX mappings
	specialCases := []struct {
		name     string
		code     int
		expected pbmlib.ErrorInfo
	}{
		{"Unauthorized specifically maps to TRX03", 401, pbmlib.ErrorCode.TRX03},
		{"Forbidden specifically maps to TRX09", 403, pbmlib.ErrorCode.TRX09},
	}

	for _, tt := range specialCases {
		t.Run(tt.name, func(t *testing.T) {
			trxCode, isError := MapHTTPStatusToTRXCode(tt.code)
			if trxCode != tt.expected {
				t.Errorf("MapHTTPStatusToTRXCode(%d) = %v, want %v", tt.code, trxCode, tt.expected)
			}
			if !isError {
				t.Errorf("MapHTTPStatusToTRXCode(%d) should be considered an error", tt.code)
			}
		})
	}
}