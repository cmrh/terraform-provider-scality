package provider

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Console API constants
const (
	consoleAuthPath    = "/_/console/authenticate"
	consoleAccountPath = "/_/console/vault/accounts"
	consoleContentType = "application/json"
	tokenCachePrefix   = ".scality_console_token_"
	tokenLifetime      = 86400 // 24 hours in seconds
	tokenSafetyMargin  = 84600 // 23.5 hours in seconds
	filePermissions    = 0600  // Owner read/write only
)

// ConsoleClient handles API communication with Scality Console API
type ConsoleClient struct {
	Endpoint   string
	Username   string
	Password   string
	HTTPClient *http.Client
	token      string
}

// NewConsoleClient creates a new Scality Console API client
func NewConsoleClient(endpoint, username, password string) *ConsoleClient {
	return &ConsoleClient{
		Endpoint:   endpoint,
		Username:   username,
		Password:   password,
		HTTPClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// TokenCache represents cached token data
type TokenCache struct {
	Token     string  `json:"token"`
	Timestamp float64 `json:"timestamp"`
}

// getCacheFile returns the path to the token cache file
func (c *ConsoleClient) getCacheFile() string {
	// Create unique cache key from endpoint and username
	cacheKey := fmt.Sprintf("%s:%s", c.Endpoint, c.Username)
	hash := md5.Sum([]byte(cacheKey))
	cacheHash := hex.EncodeToString(hash[:])

	cacheDir := os.TempDir()
	return filepath.Join(cacheDir, tokenCachePrefix+cacheHash)
}

// getCachedToken retrieves a cached token if valid
func (c *ConsoleClient) getCachedToken() (string, error) {
	cacheFile := c.getCacheFile()

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return "", err // Cache doesn't exist or can't be read
	}

	var cache TokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return "", err
	}

	// Check if token is expired
	// Use safety margin to refresh before actual expiry
	tokenAge := time.Now().Unix() - int64(cache.Timestamp)
	if tokenAge >= tokenSafetyMargin {
		// Token expired, remove cache file
		_ = os.Remove(cacheFile)
		return "", fmt.Errorf("token expired")
	}

	return cache.Token, nil
}

// cacheToken stores a token with timestamp
func (c *ConsoleClient) cacheToken(token string) error {
	cacheFile := c.getCacheFile()

	cache := TokenCache{
		Token:     token,
		Timestamp: float64(time.Now().Unix()),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	// Write with restrictive permissions (owner read/write only)
	if err := os.WriteFile(cacheFile, data, filePermissions); err != nil {
		return err
	}

	return nil
}

// Authenticate authenticates with the Console API and caches the token
func (c *ConsoleClient) Authenticate(ctx context.Context) error {
	// Try to use cached token first
	if cachedToken, err := c.getCachedToken(); err == nil {
		c.token = cachedToken
		return nil
	}

	// No valid cache, authenticate
	authURL := c.Endpoint + consoleAuthPath

	payload := map[string]string{
		"username": c.Username,
		"password": c.Password,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, httpMethodPost, authURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", consoleContentType)
	req.Header.Set("Accept", consoleContentType)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Token   string `json:"token"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("authentication failed: %s", result.Message)
	}

	c.token = result.Token

	// Cache the token (ignore errors as caching is not critical)
	_ = c.cacheToken(c.token)

	return nil
}

// ConsoleAccountCreateRequest represents Console account creation parameters
type ConsoleAccountCreateRequest struct {
	AccountName string `json:"accountName"`
	Email       string `json:"email"`
	Quota       int64  `json:"quota,omitempty"`
	Password    string `json:"password,omitempty"`
}

// ConsoleAccountCreateResponse represents the Console API account creation response
type ConsoleAccountCreateResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AccountName string `json:"accountName"`
		Email       string `json:"email"`
		Quota       int64  `json:"quota"`
		CreatedAt   string `json:"createdAt"`
	} `json:"data"`
}

// CreateConsoleAccount creates a new account via Console API (without password)
func (c *ConsoleClient) CreateConsoleAccount(ctx context.Context, req ConsoleAccountCreateRequest) (*ConsoleAccountCreateResponse, error) {
	if c.token == "" {
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	accountURL := c.Endpoint + consoleAccountPath

	jsonPayload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, httpMethodPost, accountURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", consoleContentType)
	httpReq.Header.Set("Accept", consoleContentType)
	httpReq.Header.Set("x-access-token", c.token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 409 {
		return nil, fmt.Errorf("account already exists")
	}

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result ConsoleAccountCreateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ConsoleAccessKeyResponse represents Console API access key generation response
type ConsoleAccessKeyResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AccessKey string `json:"accessKey"`
		SecretKey string `json:"secretKey"`
		Status    string `json:"status"`
	} `json:"data"`
}

// GenerateConsoleAccessKey generates persistent S3 access keys for an account
func (c *ConsoleClient) GenerateConsoleAccessKey(ctx context.Context, accountName string) (*ConsoleAccessKeyResponse, error) {
	if c.token == "" {
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	keysURL := fmt.Sprintf("%s%s/%s/keys", c.Endpoint, consoleAccountPath, accountName)

	httpReq, err := http.NewRequestWithContext(ctx, httpMethodPost, keysURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", consoleContentType)
	httpReq.Header.Set("x-access-token", c.token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result ConsoleAccessKeyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetConsoleAccount retrieves Console account details
func (c *ConsoleClient) GetConsoleAccount(ctx context.Context, accountName string) (map[string]interface{}, error) {
	if c.token == "" {
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	accountURL := fmt.Sprintf("%s%s/%s", c.Endpoint, consoleAccountPath, accountName)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", accountURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", consoleContentType)
	httpReq.Header.Set("x-access-token", c.token)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 404 {
		return nil, nil // Account doesn't exist
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// DeleteConsoleAccount deletes a Console account (two-step process)
func (c *ConsoleClient) DeleteConsoleAccount(ctx context.Context, accountName string) error {
	if c.token == "" {
		if err := c.Authenticate(ctx); err != nil {
			return err
		}
	}

	// Step 1: Delete account
	accountURL := fmt.Sprintf("%s%s/%s", c.Endpoint, consoleAccountPath, accountName)

	req1, err := http.NewRequestWithContext(ctx, "DELETE", accountURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req1.Header.Set("Accept", "*/*")
	req1.Header.Set("x-access-token", c.token)

	resp1, err := c.HTTPClient.Do(req1)
	if err != nil {
		return fmt.Errorf("account delete request failed: %w", err)
	}
	defer func() {
		_ = resp1.Body.Close()
	}()

	if resp1.StatusCode != 200 && resp1.StatusCode != 204 && resp1.StatusCode != 404 {
		body, _ := io.ReadAll(resp1.Body)
		return fmt.Errorf("account delete failed with status %d: %s", resp1.StatusCode, string(body))
	}

	// Step 2: Delete user
	userURL := fmt.Sprintf("%s%s/%s/user", c.Endpoint, consoleAccountPath, accountName)

	req2, err := http.NewRequestWithContext(ctx, "DELETE", userURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create user delete request: %w", err)
	}

	req2.Header.Set("Accept", "*/*")
	req2.Header.Set("x-access-token", c.token)

	resp2, err := c.HTTPClient.Do(req2)
	if err != nil {
		return fmt.Errorf("user delete request failed: %w", err)
	}
	defer func() {
		_ = resp2.Body.Close()
	}()

	if resp2.StatusCode != 200 && resp2.StatusCode != 204 && resp2.StatusCode != 404 {
		body, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("user delete failed with status %d: %s", resp2.StatusCode, string(body))
	}

	return nil
}
