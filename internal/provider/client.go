package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// AWS Signature V4 constants
const (
	awsService         = "iam"
	awsRegion          = "us-east-1"
	awsAlgorithm       = "AWS4-HMAC-SHA256"
	awsDateFormat      = "20060102T150405Z"
	awsDateStampFormat = "20060102"
	awsRequestType     = "aws4_request"
)

// API constants
const (
	apiVersion      = "2010-05-08"
	contentTypeForm = "application/x-www-form-urlencoded"
	canonicalURI    = "/"
)

// HTTP constants
const (
	httpMethodPost     = "POST"
	defaultHTTPTimeout = 30 * time.Second
)

// ScalityClient handles API communication with Scality
type ScalityClient struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	HTTPClient *http.Client
}

// NewScalityClient creates a new Scality API client
func NewScalityClient(endpoint, accessKey, secretKey string, insecureSkipVerify bool) *ScalityClient {
	httpClient := &http.Client{
		Timeout: defaultHTTPTimeout,
	}

	if insecureSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	return &ScalityClient{
		Endpoint:   endpoint,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		HTTPClient: httpClient,
	}
}

// signRequest signs a request with AWS Signature Version 4
// Based on the working pattern from list_accounts.py
func (c *ScalityClient) signRequest(method, requestURL, payload string) (map[string]string, error) {
	// Parse URL to get host
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	host := parsedURL.Hostname()
	canonicalQuerystring := ""

	// Create timestamp
	now := time.Now().UTC()
	amzDate := now.Format(awsDateFormat)
	dateStamp := now.Format(awsDateStampFormat)

	// Create payload hash
	h := sha256.New()
	h.Write([]byte(payload))
	payloadHash := hex.EncodeToString(h.Sum(nil))

	// Create canonical headers
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	// Create canonical request
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, canonicalURI, canonicalQuerystring,
		canonicalHeaders, signedHeaders, payloadHash)

	// Create string to sign
	credentialScope := fmt.Sprintf("%s/%s/%s/%s", dateStamp, awsRegion, awsService, awsRequestType)

	canonicalRequestHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		awsAlgorithm, amzDate, credentialScope, hex.EncodeToString(canonicalRequestHash[:]))

	// Calculate signature
	signingKey := c.getSignatureKey(dateStamp, awsRegion, awsService)
	signature := hmacSHA256(signingKey, stringToSign)

	// Create authorization header
	authorizationHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		awsAlgorithm, c.AccessKey, credentialScope, signedHeaders, hex.EncodeToString(signature))

	headers := map[string]string{
		"X-Amz-Content-Sha256": payloadHash,
		"X-Amz-Date":           amzDate,
		"Authorization":        authorizationHeader,
	}

	return headers, nil
}

// getSignatureKey derives the signing key
func (c *ScalityClient) getSignatureKey(dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+c.SecretKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, awsRequestType)
	return kSigning
}

// hmacSHA256 computes HMAC-SHA256
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// doSignedRequest performs a signed HTTP request to the Scality API
func (c *ScalityClient) doSignedRequest(ctx context.Context, action string, params url.Values) ([]byte, int, error) {
	params.Set("Action", action)
	params.Set("Version", apiVersion)

	payload := params.Encode()

	headers, err := c.signRequest(httpMethodPost, c.Endpoint, payload)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to sign request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, httpMethodPost, c.Endpoint, bytes.NewBufferString(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
	httpReq.Header.Set("Content-Type", contentTypeForm)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	return body, resp.StatusCode, nil
}

// AccountCreateRequest represents account creation parameters
type AccountCreateRequest struct {
	Action            string `json:"-"`
	Version           string `json:"-"`
	Name              string `json:"name"`
	EmailAddress      string `json:"emailAddress"`
	QuotaMax          int64  `json:"quotaMax,omitempty"`
	ExternalAccountID string `json:"externalAccountId,omitempty"`
}

// AccountCreateResponse represents the API response
type AccountCreateResponse struct {
	Account AccountData `json:"account"`
}

// AccountData contains account details
type AccountData struct {
	Data struct {
		ARN              string                 `json:"arn"`
		CanonicalID      string                 `json:"canonicalId"`
		ID               string                 `json:"id"`
		EmailAddress     string                 `json:"emailAddress"`
		Name             string                 `json:"name"`
		CreateDate       string                 `json:"createDate"`
		QuotaMax         int64                  `json:"quotaMax"`
		AliasList        []string               `json:"aliasList"`
		OIDCPList        []string               `json:"oidcpList"`
		CustomAttributes map[string]interface{} `json:"customAttributes"`
	} `json:"data"`
}

// AccessKeyResponse represents access key generation response
type AccessKeyResponse struct {
	Data struct {
		ID              string `json:"id"`
		Value           string `json:"value"`
		CreateDate      string `json:"createDate"`
		LastUsedDate    string `json:"lastUsedDate"`
		LastUsedRegion  string `json:"lastUsedRegion"`
		LastUsedService string `json:"lastUsedService"`
		Status          string `json:"status"`
		UserID          string `json:"userId"`
	} `json:"data"`
}

// CreateAccount creates a new Scality account
func (c *ScalityClient) CreateAccount(ctx context.Context, req AccountCreateRequest) (*AccountCreateResponse, error) {
	params := url.Values{}
	params.Set("name", req.Name)
	params.Set("emailAddress", req.EmailAddress)
	if req.QuotaMax > 0 {
		params.Set("quotaMax", fmt.Sprintf("%d", req.QuotaMax))
	}
	if req.ExternalAccountID != "" {
		params.Set("externalAccountId", req.ExternalAccountID)
	}

	body, statusCode, err := c.doSignedRequest(ctx, "CreateAccount", params)
	if err != nil {
		return nil, err
	}

	if statusCode == 409 {
		return nil, fmt.Errorf("account already exists")
	}

	if statusCode != 201 {
		return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
	}

	var result AccountCreateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GenerateAccountAccessKey generates S3 API credentials for an account
func (c *ScalityClient) GenerateAccountAccessKey(ctx context.Context, accountName string) (*AccessKeyResponse, error) {
	params := url.Values{}
	params.Set("AccountName", accountName)

	body, statusCode, err := c.doSignedRequest(ctx, "GenerateAccountAccessKey", params)
	if err != nil {
		return nil, err
	}

	if statusCode != 201 {
		return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
	}

	var result AccessKeyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetAccount retrieves account details (for Read/refresh)
func (c *ScalityClient) GetAccount(ctx context.Context, accountName string) (*AccountData, error) {
	params := url.Values{}
	params.Set("AccountName", accountName)

	body, statusCode, err := c.doSignedRequest(ctx, "GetAccount", params)
	if err != nil {
		return nil, err
	}

	if statusCode == 404 {
		return nil, nil // Account doesn't exist
	}

	if statusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
	}

	var result AccountData
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// DeleteAccount deletes a Scality account
func (c *ScalityClient) DeleteAccount(ctx context.Context, accountName string) error {
	params := url.Values{}
	params.Set("AccountName", accountName)

	body, statusCode, err := c.doSignedRequest(ctx, "DeleteAccount", params)
	if err != nil {
		return err
	}

	if statusCode == 404 {
		return nil // Account already deleted (idempotent)
	}

	if statusCode == 409 {
		return fmt.Errorf(
			"cannot delete account '%s' - the account contains resources that must be removed first.\n\n"+
				"The account may contain:\n"+
				"  • IAM users\n"+
				"  • IAM policies\n"+
				"  • S3 buckets (empty or with data)\n\n"+
				"Required actions before deletion:\n"+
				"  1. Delete all IAM users in the account\n"+
				"  2. Delete all IAM policies in the account\n"+
				"  3. Delete all objects from S3 buckets\n"+
				"  4. Delete all S3 buckets\n"+
				"  5. Retry account deletion",
			accountName,
		)
	}

	if statusCode != 200 {
		return fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
	}

	return nil
}
