package global

import (
	"fmt"
	"testing"

	"github.com/transactrx/ncpdpDestination/pkg/pbmlib"
)

// MockPBMConnect implements PBMConnect interface for testing
type MockPBMConnect struct {
	StartCalled  bool
	PostCalled   bool
	CloseCalled  bool
	StartError   error
	PostResponse []byte
	PostHeaders  map[string][]string
	PostError    pbmlib.ErrorInfo
	CloseError   error
}

func (m *MockPBMConnect) Start(config map[string]interface{}) error {
	m.StartCalled = true
	return m.StartError
}

func (m *MockPBMConnect) Post(claim []byte, header map[string][]string) ([]byte, map[string][]string, pbmlib.ErrorInfo) {
	m.PostCalled = true
	return m.PostResponse, m.PostHeaders, m.PostError
}

func (m *MockPBMConnect) Close() error {
	m.CloseCalled = true
	return m.CloseError
}

// MockPBMConnectWithStats implements PBMConnectWithStats interface for testing
type MockPBMConnectWithStats struct {
	MockPBMConnect
	GetStatsCalled bool
	GetStatsError  error
}

func (m *MockPBMConnectWithStats) GetStats(stats map[string]interface{}) error {
	m.GetStatsCalled = true
	if m.GetStatsError == nil {
		stats["sessionCount"] = "5"
		stats["totalRequests"] = "100"
		stats["errorCount"] = "2"
	}
	return m.GetStatsError
}

func TestPBMConnectInterface(t *testing.T) {
	mock := &MockPBMConnect{
		PostResponse: []byte("test response"),
		PostHeaders:  map[string][]string{"Content-Type": {"application/json"}},
		PostError:    pbmlib.ErrorCode.TRX00,
	}

	var pbmConn PBMConnect = mock

	// Test Start method
	config := map[string]interface{}{
		"pbmUrl":  "localhost",
		"pbmPort": "8080",
	}
	err := pbmConn.Start(config)
	if err != nil {
		t.Errorf("Start() failed: %v", err)
	}
	if !mock.StartCalled {
		t.Error("Start() was not called")
	}

	// Test Post method
	claim := []byte("test claim")
	headers := map[string][]string{"Authorization": {"Bearer token"}}
	
	response, respHeaders, errInfo := pbmConn.Post(claim, headers)
	if errInfo != pbmlib.ErrorCode.TRX00 {
		t.Errorf("Post() error = %v, want %v", errInfo, pbmlib.ErrorCode.TRX00)
	}
	if string(response) != "test response" {
		t.Errorf("Post() response = %v, want %v", string(response), "test response")
	}
	if respHeaders["Content-Type"][0] != "application/json" {
		t.Errorf("Post() headers = %v, want Content-Type: application/json", respHeaders)
	}
	if !mock.PostCalled {
		t.Error("Post() was not called")
	}

	// Test Close method
	err = pbmConn.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
	if !mock.CloseCalled {
		t.Error("Close() was not called")
	}
}

func TestPBMConnectWithStatsInterface(t *testing.T) {
	mock := &MockPBMConnectWithStats{
		MockPBMConnect: MockPBMConnect{
			PostResponse: []byte("stats test response"),
			PostError:    pbmlib.ErrorCode.TRX00,
		},
	}

	var pbmConn PBMConnectWithStats = mock

	// Test that it implements base PBMConnect interface
	config := map[string]interface{}{
		"pbmUrl": "localhost",
	}
	err := pbmConn.Start(config)
	if err != nil {
		t.Errorf("Start() failed: %v", err)
	}

	claim := []byte("test claim")
	headers := map[string][]string{}
	_, _, errInfo := pbmConn.Post(claim, headers)
	if errInfo != pbmlib.ErrorCode.TRX00 {
		t.Errorf("Post() error = %v, want success", errInfo)
	}

	// Test GetStats method
	stats := make(map[string]interface{})
	err = pbmConn.GetStats(stats)
	if err != nil {
		t.Errorf("GetStats() failed: %v", err)
	}
	if !mock.GetStatsCalled {
		t.Error("GetStats() was not called")
	}

	// Verify stats were populated
	if stats["sessionCount"] != "5" {
		t.Errorf("GetStats() sessionCount = %v, want %v", stats["sessionCount"], "5")
	}
	if stats["totalRequests"] != "100" {
		t.Errorf("GetStats() totalRequests = %v, want %v", stats["totalRequests"], "100")
	}
	if stats["errorCount"] != "2" {
		t.Errorf("GetStats() errorCount = %v, want %v", stats["errorCount"], "2")
	}

	err = pbmConn.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestPBMConnectErrorHandling(t *testing.T) {
	testErr := fmt.Errorf("test error")
	mock := &MockPBMConnect{
		StartError: testErr,
		PostError:  pbmlib.ErrorCode.TRX02,
		CloseError: testErr,
	}

	var pbmConn PBMConnect = mock

	// Test Start error
	config := map[string]interface{}{}
	err := pbmConn.Start(config)
	if err != testErr {
		t.Errorf("Start() error = %v, want %v", err, testErr)
	}

	// Test Post error
	claim := []byte("test claim")
	headers := map[string][]string{}
	_, _, errInfo := pbmConn.Post(claim, headers)
	if errInfo != pbmlib.ErrorCode.TRX02 {
		t.Errorf("Post() error = %v, want %v", errInfo, pbmlib.ErrorCode.TRX02)
	}

	// Test Close error
	err = pbmConn.Close()
	if err != testErr {
		t.Errorf("Close() error = %v, want %v", err, testErr)
	}
}

func TestPBMConnectWithStatsErrorHandling(t *testing.T) {
	testErr := fmt.Errorf("stats error")
	mock := &MockPBMConnectWithStats{
		GetStatsError: testErr,
	}

	var pbmConn PBMConnectWithStats = mock

	// Test GetStats error
	stats := make(map[string]interface{})
	err := pbmConn.GetStats(stats)
	if err != testErr {
		t.Errorf("GetStats() error = %v, want %v", err, testErr)
	}

	// Verify stats map is not populated on error
	if len(stats) != 0 {
		t.Errorf("GetStats() populated stats on error: %v", stats)
	}
}

func TestInterfaceCompatibility(t *testing.T) {
	// Test that PBMConnectWithStats can be used as PBMConnect
	mockWithStats := &MockPBMConnectWithStats{}
	
	var pbmConnect PBMConnect = mockWithStats
	var pbmConnectWithStats PBMConnectWithStats = mockWithStats

	// Both should work
	_ = pbmConnect.Start(map[string]interface{}{})
	_ = pbmConnectWithStats.Start(map[string]interface{}{})
	
	_, _, _ = pbmConnect.Post([]byte{}, map[string][]string{})
	_, _, _ = pbmConnectWithStats.Post([]byte{}, map[string][]string{})
	
	_ = pbmConnect.Close()
	_ = pbmConnectWithStats.Close()
	
	// Only the stats version should have GetStats
	_ = pbmConnectWithStats.GetStats(map[string]interface{}{})
}