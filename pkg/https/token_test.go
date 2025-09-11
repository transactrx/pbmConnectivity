package https

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenConfig(t *testing.T) {
	config := TokenConfig{
		TokenType:    ClientCredentials,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     "https://example.com/token",
		HTTPTimeout:  30 * time.Second,
	}

	tm := NewTokenManagerWithConfig(config)

	if tm.config.TokenType != ClientCredentials {
		t.Errorf("TokenType = %v, want %v", tm.config.TokenType, ClientCredentials)
	}
	if tm.config.ClientID != "test-client" {
		t.Errorf("ClientID = %v, want %v", tm.config.ClientID, "test-client")
	}
	if tm.config.ClientSecret != "test-secret" {
		t.Errorf("ClientSecret = %v, want %v", tm.config.ClientSecret, "test-secret")
	}
	if tm.config.TokenURL != "https://example.com/token" {
		t.Errorf("TokenURL = %v, want %v", tm.config.TokenURL, "https://example.com/token")
	}
	if tm.httpClient.Timeout != 30*time.Second {
		t.Errorf("HTTPTimeout = %v, want %v", tm.httpClient.Timeout, 30*time.Second)
	}
}

func TestIsValidTokenSettings(t *testing.T) {
	tests := []struct {
		name     string
		config   TokenConfig
		expected bool
	}{
		{
			name: "all required fields present",
			config: TokenConfig{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				TokenURL:     "https://example.com/token",
			},
			expected: true,
		},
		{
			name: "missing client ID",
			config: TokenConfig{
				ClientSecret: "test-secret",
				TokenURL:     "https://example.com/token",
			},
			expected: false,
		},
		{
			name: "missing client secret",
			config: TokenConfig{
				ClientID: "test-client",
				TokenURL: "https://example.com/token",
			},
			expected: false,
		},
		{
			name: "missing token URL",
			config: TokenConfig{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			expected: false,
		},
		{
			name: "empty client ID",
			config: TokenConfig{
				ClientID:     "",
				ClientSecret: "test-secret",
				TokenURL:     "https://example.com/token",
			},
			expected: false,
		},
		{
			name:     "all fields empty",
			config:   TokenConfig{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTokenManagerWithConfig(tt.config)
			result := tm.IsValidTokenSettings()
			if result != tt.expected {
				t.Errorf("IsValidTokenSettings() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTokenManagerValid(t *testing.T) {
	tm := NewTokenManagerWithConfig(TokenConfig{})

	// Test with nil token
	if tm.Valid() {
		t.Error("Valid() should return false with nil token")
	}

	// Test with empty access token
	tm.token = &oauth2.Token{AccessToken: ""}
	if tm.Valid() {
		t.Error("Valid() should return false with empty access token")
	}

	// Test with valid non-expired token
	future := time.Now().Add(1 * time.Hour)
	tm.token = &oauth2.Token{
		AccessToken: "valid-token",
		Expiry:      future,
	}
	if !tm.Valid() {
		t.Error("Valid() should return true with valid non-expired token")
	}

	// Test with expired token
	past := time.Now().Add(-1 * time.Hour)
	tm.token = &oauth2.Token{
		AccessToken: "expired-token",
		Expiry:      past,
	}
	if tm.Valid() {
		t.Error("Valid() should return false with expired token")
	}
}

func TestTokenManagerExpired(t *testing.T) {
	tm := NewTokenManagerWithConfig(TokenConfig{})

	// Test with nil token
	if !tm.Expired() {
		t.Error("Expired() should return true with nil token")
	}

	// Test with zero expiry (never expires)
	tm.token = &oauth2.Token{
		AccessToken: "never-expires",
		Expiry:      time.Time{},
	}
	if tm.Expired() {
		t.Error("Expired() should return false with zero expiry")
	}

	// Test with future expiry
	future := time.Now().Add(1 * time.Hour)
	tm.token = &oauth2.Token{
		AccessToken: "future-token",
		Expiry:      future,
	}
	if tm.Expired() {
		t.Error("Expired() should return false with future expiry")
	}

	// Test with past expiry
	past := time.Now().Add(-1 * time.Hour)
	tm.token = &oauth2.Token{
		AccessToken: "past-token",
		Expiry:      past,
	}
	if !tm.Expired() {
		t.Error("Expired() should return true with past expiry")
	}

	// Test expiry within refresh delta
	almostExpired := time.Now().Add(5 * time.Second) // Within default 10s refresh delta
	tm.token = &oauth2.Token{
		AccessToken: "almost-expired",
		Expiry:      almostExpired,
	}
	if !tm.Expired() {
		t.Error("Expired() should return true when within refresh delta")
	}
}

func TestTokenManagerExpiredWithCustomDelta(t *testing.T) {
	tm := NewTokenManagerWithConfig(TokenConfig{})
	tm.expiryDelta = 30 * time.Second

	// Test expiry within custom delta
	almostExpired := time.Now().Add(20 * time.Second) // Within custom 30s delta
	tm.token = &oauth2.Token{
		AccessToken: "almost-expired",
		Expiry:      almostExpired,
	}
	if !tm.Expired() {
		t.Error("Expired() should return true when within custom expiry delta")
	}

	// Test expiry outside custom delta
	notExpired := time.Now().Add(40 * time.Second) // Outside custom 30s delta
	tm.token = &oauth2.Token{
		AccessToken: "not-expired",
		Expiry:      notExpired,
	}
	if tm.Expired() {
		t.Error("Expired() should return false when outside custom expiry delta")
	}
}

func TestGetIDToken(t *testing.T) {
	tm := NewTokenManagerWithConfig(TokenConfig{})

	// Test with nil token
	if id := tm.GetIDToken(); id != "" {
		t.Errorf("GetIDToken() = %v, want empty string with nil token", id)
	}

	// Test with token without ID token
	tm.token = &oauth2.Token{AccessToken: "access-token"}
	if id := tm.GetIDToken(); id != "" {
		t.Errorf("GetIDToken() = %v, want empty string without id_token", id)
	}

	// Test with token containing ID token
	tokenWithID := &oauth2.Token{AccessToken: "access-token"}
	tokenWithID = tokenWithID.WithExtra(map[string]interface{}{
		"id_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...",
	})
	tm.token = tokenWithID

	id := tm.GetIDToken()
	if id != "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9..." {
		t.Errorf("GetIDToken() = %v, want JWT token", id)
	}
}

func TestBasicAuth(t *testing.T) {
	tests := []struct {
		username string
		password string
		expected string
	}{
		{
			username: "user",
			password: "pass",
			expected: "dXNlcjpwYXNz", // base64("user:pass")
		},
		{
			username: "client-id",
			password: "client-secret",
			expected: "Y2xpZW50LWlkOmNsaWVudC1zZWNyZXQ=", // base64("client-id:client-secret")
		},
		{
			username: "",
			password: "",
			expected: "Og==", // base64(":")
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s:%s", tt.username, tt.password), func(t *testing.T) {
			result := basicAuth(tt.username, tt.password)
			if result != tt.expected {
				t.Errorf("basicAuth(%q, %q) = %q, want %q", tt.username, tt.password, result, tt.expected)
			}
		})
	}
}

func TestParseToken(t *testing.T) {
	tests := []struct {
		name    string
		raw     []byte
		wantErr bool
	}{
		{
			name: "valid token response",
			raw: []byte(`{
				"access_token": "token123",
				"token_type": "Bearer",
				"expires_in": 3600
			}`),
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			raw:     []byte(`{invalid json`),
			wantErr: true,
		},
		{
			name:    "empty response",
			raw:     []byte(``),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := parseToken(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == nil {
				t.Error("parseToken() returned nil token without error")
			}
			if !tt.wantErr && token.AccessToken != "token123" {
				t.Errorf("parseToken() token.AccessToken = %v, want %v", token.AccessToken, "token123")
			}
		})
	}
}

func TestRefreshToken(t *testing.T) {
	// Mock server for token endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected application/x-www-form-urlencoded content type")
		}
		if r.Header.Get("Authorization") == "" {
			t.Errorf("Expected Authorization header")
		}

		response := map[string]interface{}{
			"access_token": "new-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := TokenConfig{
		TokenType:    ClientCredentials,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     server.URL,
		HTTPTimeout:  5 * time.Second,
	}

	tm := NewTokenManagerWithConfig(config)
	err := tm.refreshToken()
	if err != nil {
		t.Errorf("refreshToken() error = %v", err)
	}

	if tm.token == nil {
		t.Error("refreshToken() did not set token")
	}
	if tm.token.AccessToken != "new-token" {
		t.Errorf("refreshToken() token.AccessToken = %v, want %v", tm.token.AccessToken, "new-token")
	}
}

func TestRefreshTokenError(t *testing.T) {
	// Mock server that returns malformed JSON (will cause parsing error)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	config := TokenConfig{
		TokenType:    ClientCredentials,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     server.URL,
		HTTPTimeout:  5 * time.Second,
	}

	tm := NewTokenManagerWithConfig(config)
	err := tm.refreshToken()
	if err == nil {
		t.Error("refreshToken() should return error for malformed JSON response")
	}
}

func TestGetTokenWithRefresh(t *testing.T) {
	// Mock server for token endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"access_token": "refreshed-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := TokenConfig{
		TokenType:    ClientCredentials,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     server.URL,
		HTTPTimeout:  5 * time.Second,
	}

	tm := NewTokenManagerWithConfig(config)

	// Test GetToken with no existing token (should refresh)
	token, err := tm.GetToken()
	if err != nil && token != "refreshed-token" {
		t.Errorf("GetToken() = %v, want %v", token, "refreshed-token")
	}

	// Test GetToken with valid existing token (should not refresh)
	future := time.Now().Add(1 * time.Hour)
	tm.token = &oauth2.Token{
		AccessToken: "existing-valid-token",
		Expiry:      future,
	}

	token, err = tm.GetToken()
	if err != nil && token != "existing-valid-token" {
		t.Errorf("GetToken() = %v, want %v", token, "existing-valid-token")
	}
}

func TestTokenTypes(t *testing.T) {
	tests := []struct {
		name      string
		tokenType TokenType
		expected  string
	}{
		{
			name:      "client credentials",
			tokenType: ClientCredentials,
			expected:  "client_credentials",
		},
		{
			name:      "authorization code",
			tokenType: AuthorizationCode,
			expected:  "authorization_code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.tokenType) != tt.expected {
				t.Errorf("TokenType %v = %v, want %v", tt.tokenType, string(tt.tokenType), tt.expected)
			}
		})
	}
}

// Test concurrent access to token manager
func TestTokenManagerConcurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"access_token": "concurrent-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := TokenConfig{
		TokenType:    ClientCredentials,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     server.URL,
		HTTPTimeout:  5 * time.Second,
	}

	tm := NewTokenManagerWithConfig(config)

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			token, err := tm.GetToken()
			if err != nil && token == "" {
				t.Error("GetToken() returned empty token in concurrent access")
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
