package https

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	var err error
	for {
		if tm.token != nil {
			expiresIn := time.Duration(tm.token.ExpiresIn) * time.Second
			expiry := timeNow().Add(expiresIn)

			refreshAfter := time.Until(expiry.Add(-refreshDelta))
			log.Printf("AutoRefreshToken refreshAfter: %s, expiry: %v", refreshAfter, expiry)

			if refreshAfter > 0 {
				time.Sleep(refreshAfter)
			}
			err = tm.refreshToken()
			if err != nil {
				log.Printf("AutoRefreshToken Token refresh failed: %v waiting 10 seconds before retrying", err)
				time.Sleep(10 * time.Second)
			}
		} else {
			log.Printf("AutoRefreshToken - waiting 10 seconds...")
			time.Sleep(10 * time.Second)
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
	log.Printf("RefreshToken Token type: %v...", tm.config.TokenGrantType)
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
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basicAuth(tm.config.ClientID, tm.config.ClientSecret))
	res, err := tm.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	err = tm.processResponse(res)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	//log.Printf("RefreshToken: raw response: %s", body)
	token, err := parseToken(body)
	if err != nil {
		return err
	}
	tm.token = token

	if IsDebugMode() {
		DebugToken(tm.token.AccessToken)
	}
	log.Printf("RefreshToken done isValid: %t", tm.Valid())
	return nil
}

func DebugToken(accessToken string) {
	parts := strings.Split(accessToken, ".")
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	log.Printf("claims: %s", payload) // JSON; confirm "aud"
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
func (tm *TokenManager) processResponse(res *http.Response) error {
	if res.StatusCode != http.StatusOK {
		log.Printf("processResponse: token request failed with status %s", res.Status)
		return fmt.Errorf("token request failed: %s", res.Status)
	}
	return nil
}
