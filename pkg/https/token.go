package https

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// TokenType is a custom type for token types.
type TokenType string

// Enum-like constants for TokenType.
const (
	ClientCredentials TokenType = "client_credentials" // service to service interaction - no login needed
	AuthorizationCode TokenType = "authorization_code" // linked to user login
	RefreshToken      TokenType = "refresh_token"
)

const refreshDelta = 10 * time.Second

type TokenManager struct {
	config      TokenConfig
	httpClient  *http.Client
	token       *oauth2.Token
	expiryDelta time.Duration // optional; if zero, fall back to default
	mu          sync.RWMutex
	tokenExpiry time.Time
	host        string
}

type TokenConfig struct {
	TokenGrantType TokenType
	TokenScope     string
	ClientID       string
	ClientSecret   string
	TokenURL       string
	HTTPTimeout    time.Duration
}

var timeNow = time.Now

func (tm *TokenManager) IsValidTokenSettings() bool {
	//log.Printf("TokenMgr config: %v", tm.config)
	if len(tm.config.ClientID) > 0 && len(tm.config.ClientSecret) > 0 && len(tm.config.TokenURL) > 0 {
		return true
	}
	return false
}

func (tm *TokenManager) PrintStats(action string, errorCode int, errorDesc string, httpCode int, isValid bool, jti string) {
	// Construct the stats string
	errcode := strconv.FormatInt(int64(errorCode), 10)
	httpcode := strconv.FormatInt(int64(httpCode), 10)
	isvalid := strconv.FormatBool(isValid)

	stats := "tokenstats host: " + tm.host +
		" action: " + action +
		" errcode: " + errcode +
		" errdesc: " + errorDesc +
		" httpcode: " + httpcode +
		" isvalid: " + isvalid +
		" jti: " + jti

	log.Printf("%s", stats)
}

func NewTokenManagerWithConfig(cfg TokenConfig) *TokenManager {
	return &TokenManager{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
	}
}

// Valid reports whether the token is non-nil, has an AccessToken, and is not expired.
func (tm *TokenManager) Valid() bool {
	if tm.token == nil || tm.token.AccessToken == "" {
		return false
	}
	return !tm.Expired()
}

func (tm *TokenManager) Expired() bool {
	if tm.token == nil {
		return true
	}
	if tm.token.Expiry.IsZero() {
		return false
	}
	expiryDelta := tm.expiryDelta
	if expiryDelta == 0 {
		expiryDelta = refreshDelta
	}
	log.Printf("Now: %v, Token expiry: %v, Adjusted cutoff: %v\n", timeNow(), tm.token.Expiry, tm.token.Expiry.Add(-expiryDelta))
	return tm.token.Expiry.Add(-expiryDelta).Before(timeNow())
}

func (tm *TokenManager) AutoRefreshToken() {
	for {
		tokenNil := tm.token == nil
		refreshAt := tm.tokenExpiry.Add(-refreshDelta) // e.g., 60–120s early
		if tokenNil {
			_ = tm.refreshToken()
			log.Printf("autorefreshtoken - waiting 10 seconds...")
			time.Sleep(10 * time.Second)
			continue
		}
		sleep := time.Until(refreshAt)
		if sleep > 0 {
			log.Printf("autorefreshtoken sleeping til next cycle %f seconds", sleep.Seconds())
			time.Sleep(sleep)
		}
		if err := tm.refreshToken(); err != nil {
			log.Printf("autorefreshtoken refresh failed: %v; retrying in 10 seconds", err)
			time.Sleep(10 * time.Second)
			//backoffSleep() // exp backoff + jitter; DO NOT touch tm.token/tm.tokenExpiry here
		}
	}
}

// GetToken returns a valid access token or an empty string if unavailable.
func (tm *TokenManager) GetToken() (string, error) {
	tm.mu.RLock()
	isValid := tm.token != nil && tm.Valid()
	accessToken := ""
	if isValid {
		accessToken = tm.token.AccessToken
	}
	tm.mu.RUnlock()

	if isValid {
		return accessToken, nil
	}

	// Now we need to refresh — get write lock
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double-check in case another goroutine refreshed it already
	if tm.token == nil || !tm.Valid() {
		if err := tm.refreshToken(); err != nil {
			log.Printf("GetToken: unable to refresh token: %v", err)
			return "", err
		}
	}

	return tm.token.AccessToken, nil
}

// GetIDToken extracts the raw ID token from the token response.
func (tm *TokenManager) GetIDToken() string {
	if tm.token == nil {
		return ""
	}
	idToken, ok := tm.token.Extra("id_token").(string)
	if !ok {
		log.Println("GetIDToken: id_token not found in token response")
		return ""
	}
	return idToken
}

// refreshToken performs a client credentials token request and updates the stored token.
func (tm *TokenManager) refreshToken() error {
	data := url.Values{}
	tm.PrintStats("request", 0, "na", 0, tm.Valid(), "na")
	switch tm.config.TokenGrantType {
	case ClientCredentials:
		data.Set("grant_type", "client_credentials")
	case AuthorizationCode:
		data.Set("grant_type", "authorization_code")
	case RefreshToken:
		data.Set("grant_type", "refresh_token")
	default:
		log.Printf("RefreshToken Unsupported token type: %s", tm.config.TokenGrantType)
	}
	if len(tm.config.TokenScope) > 0 {
		data.Set("scope", tm.config.TokenScope)
	}

	req, err := http.NewRequest(http.MethodPost, tm.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		tm.PrintStats("response", -1, fmt.Sprintf("%v", err), -1, tm.Valid(), "na")
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basicAuth(tm.config.ClientID, tm.config.ClientSecret))
	res, err := tm.httpClient.Do(req)
	if err != nil {
		tm.PrintStats("response", -1, fmt.Sprintf("%v", err), -1, tm.Valid(), "na")
		return err
	}
	defer res.Body.Close()
	err1, httpCode := tm.processResponse(res)
	if err1 != nil {
		tm.PrintStats("response", httpCode, fmt.Sprintf("%v", err), httpCode, tm.Valid(), "na")
		return err1
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		tm.PrintStats("response", -1, fmt.Sprintf("%v", err), -1, tm.Valid(), "na")
		return err
	}
	//log.Printf("RefreshToken: raw response: %s", body)
	token, err := parseToken(body)
	if err != nil {
		tm.PrintStats("response", -1, fmt.Sprintf("%v", err), -1, tm.Valid(), "na")
		return err
	}
	tm.setToken(token)
	//if IsDebugMode() {
	jti := DebugToken(tm.token.AccessToken)
	//}
	tm.PrintStats("response", 0, "success", 0, tm.Valid(), jti)
	return nil
}

func (tm *TokenManager) setToken(t *oauth2.Token) {
	//tm.mu.Lock()
	tm.token = t
	tm.tokenExpiry = time.Now().Add(time.Second * time.Duration(t.ExpiresIn))
	//tm.mu.Unlock()
}

func DebugToken(tok string) string {
	parts := strings.Split(tok, ".")
	if len(parts) < 2 {
		log.Print("not a JWT")
		return ""
	}

	// Decode payload (2nd part)
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		log.Printf("decode error: %v", err)
		return ""
	}

	// Log claims JSON for reference
	//log.Printf("claims: %s", b)

	// Parse claims as JSON
	var claims map[string]interface{}
	if err := json.Unmarshal(b, &claims); err != nil {
		log.Printf("unmarshal error: %v", err)
		return ""
	}

	// Extract jti if present
	if jti, ok := claims["jti"].(string); ok {
		//log.Printf("jti: %s", jti)
		return jti
	}

	log.Print("no jti claim found")
	return ""
}

// parseToken parses the OAuth2 token from the raw response body.
func parseToken(raw []byte) (*oauth2.Token, error) {
	var token oauth2.Token
	if err := json.Unmarshal(raw, &token); err != nil {
		log.Printf("parseToken: failed to unmarshal token: %v", err)
		return nil, err
	}
	return &token, nil
}

// basicAuth encodes client ID and secret using Base64.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// processResponse logs HTTP status info.
func (tm *TokenManager) processResponse(res *http.Response) (error, int) {
	if res.StatusCode != http.StatusOK {
		log.Printf("processResponse: token request failed with status %s", res.Status)
		return fmt.Errorf("token request failed: %s", res.Status), res.StatusCode
	}
	return nil, res.StatusCode
}
