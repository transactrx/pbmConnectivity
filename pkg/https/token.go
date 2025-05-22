package https

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"golang.org/x/oauth2"
	
)

// TokenType is a custom type for token types.
type TokenType string

// Enum-like constants for TokenType.
const (
	ClientCredentials TokenType = "client_credentials" // service to service interaction - no login needed 
	AuthorizationCode TokenType = "authorization_code" // linked to user login 
)

type TokenManager struct {
	config     TokenConfig
	httpClient *http.Client
	token      *oauth2.Token
}

type TokenConfig struct {
	TokenType TokenType
	ClientID     string
	ClientSecret string
	TokenURL     string
	HTTPTimeout  time.Duration
}

func (tm *TokenManager)IsValidTokenSettings()bool {
	log.Printf("TokenMgr config: %v",tm.config)
	if len(tm.config.ClientID)>0 && len(tm.config.ClientSecret)>0&&len(tm.config.TokenURL)>0 {
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

// GetToken returns a valid access token or an empty string if unavailable.
func (tm *TokenManager) GetToken() string {
	log.Printf("GetToken: tm.token.isValid()?: %t",tm.token.Valid())
	if tm.token == nil || !tm.token.Valid() {
		if err := tm.refreshToken(); err != nil {
			log.Printf("GetToken: unable to refresh token: %v", err)
			return ""
		}
	}

	return tm.token.AccessToken
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
	log.Printf("Token type: %v",tm.config.TokenType)
	switch tm.config.TokenType {
	case ClientCredentials:
		data.Set("grant_type", "client_credentials")
	case AuthorizationCode:
		data.Set("scope", "openid")
	default: 
		log.Printf("Unsupported token type: %s", tm.config.TokenType)
	}

	req, err := http.NewRequest(http.MethodPost, tm.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basicAuth(tm.config.ClientID, tm.config.ClientSecret))

	log.Println("refreshToken: requesting new token...")
	res, err := tm.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	tm.processResponse(res)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	log.Printf("refreshToken: raw response: %s", body)

	token, err := parseToken(body)
	if err != nil {
		return err
	}
	tm.token = token
	return nil
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
func (tm *TokenManager) processResponse(res *http.Response) {
	if res.StatusCode != http.StatusOK {
		log.Printf("processResponse: token request failed with status %s", res.Status)
	}
}
