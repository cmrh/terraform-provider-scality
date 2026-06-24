package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

// IAMClient handles API communication with Scality IAM API using AWS Signature V4.
type IAMClient struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	Region     string
	HTTPClient *http.Client
}

// NewIAMClient creates a new Scality IAM API client. Region defaults to awsRegion.
func NewIAMClient(endpoint, accessKey, secretKey string, insecureSkipVerify bool) *IAMClient {
	httpClient := &http.Client{
		Timeout: defaultHTTPTimeout,
	}

	if insecureSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // #nosec G402 -- gated on the user-set insecure_skip_verify provider attribute
			},
		}
	}

	return &IAMClient{
		Endpoint:   endpoint,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		Region:     awsRegion,
		HTTPClient: httpClient,
	}
}

func (c *IAMClient) signRequest(method, requestURL, payload, accessKey, secretKey string) (map[string]string, error) {
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	host := parsedURL.Host
	canonicalQuerystring := ""

	now := time.Now().UTC()
	amzDate := now.Format(awsDateFormat)
	dateStamp := now.Format(awsDateStampFormat)

	h := sha256.New()
	h.Write([]byte(payload))
	payloadHash := hex.EncodeToString(h.Sum(nil))

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, canonicalURI, canonicalQuerystring,
		canonicalHeaders, signedHeaders, payloadHash)

	credentialScope := fmt.Sprintf("%s/%s/%s/%s", dateStamp, c.Region, awsService, awsRequestType)

	canonicalRequestHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		awsAlgorithm, amzDate, credentialScope, hex.EncodeToString(canonicalRequestHash[:]))

	signingKey := getSignatureKey(secretKey, dateStamp, c.Region, awsService)
	signature := hmacSHA256(signingKey, stringToSign)

	authorizationHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		awsAlgorithm, accessKey, credentialScope, signedHeaders, hex.EncodeToString(signature))

	headers := map[string]string{
		"X-Amz-Content-Sha256": payloadHash,
		"X-Amz-Date":           amzDate,
		"Authorization":        authorizationHeader,
	}

	return headers, nil
}

func getSignatureKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	kSigning := hmacSHA256(kService, awsRequestType)
	return kSigning
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// DoSignedRequest performs a signed HTTP request using admin credentials (JSON responses).
func (c *IAMClient) DoSignedRequest(ctx context.Context, action string, params url.Values) ([]byte, int, error) {
	params.Set("Action", action)
	params.Set("Version", apiVersion)

	payload := params.Encode()

	headers, err := c.signRequest(httpMethodPost, c.Endpoint, payload, c.AccessKey, c.SecretKey)
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

// XML error response for IAM API.
type iamErrorResponse struct {
	XMLName xml.Name `xml:"ErrorResponse"`
	Error   struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	} `xml:"Error"`
}

// doSignedRequest performs a signed HTTP request with per-call credentials (XML responses).
// Uses its own SigV4 signing that includes the port in the Host header, matching the
// standard AWS IAM signing convention expected by the Scality Vault service.
func (c *IAMClient) doSignedRequest(ctx context.Context, accessKey, secretKey string, params url.Values) ([]byte, error) {
	params.Set("Version", apiVersion)

	parsedURL, err := url.Parse(c.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint: %w", err)
	}

	host := parsedURL.Host
	now := time.Now().UTC()
	datestamp := now.Format(awsDateStampFormat)
	amzdate := now.Format(awsDateFormat)

	body := params.Encode()

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-date:%s\n", host, amzdate)
	signedHeaders := "host;x-amz-date"
	payloadHash := sha256Hex([]byte(body))

	canonicalRequest := strings.Join([]string{
		httpMethodPost,
		canonicalURI,
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/%s/%s", datestamp, c.Region, awsService, awsRequestType)
	stringToSign := strings.Join([]string{
		awsAlgorithm,
		amzdate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := getSignatureKey(secretKey, datestamp, c.Region, awsService)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		awsAlgorithm, accessKey, credentialScope, signedHeaders, signature)

	httpReq, err := http.NewRequestWithContext(ctx, httpMethodPost, c.Endpoint+"/", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", contentTypeForm)
	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("X-Amz-Date", amzdate)
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var iamErr iamErrorResponse
		if xmlErr := xml.Unmarshal(respBody, &iamErr); xmlErr == nil && iamErr.Error.Code != "" {
			return nil, &APIError{
				Code:       iamErr.Error.Code,
				Message:    iamErr.Error.Message,
				StatusCode: resp.StatusCode,
			}
		}
		return nil, fmt.Errorf("IAM request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// AccountCreateRequest represents account creation parameters.
type AccountCreateRequest struct {
	Name              string
	EmailAddress      string
	QuotaMax          int64
	ExternalAccountID string
}

// AccountCreateResponse represents the API response.
type AccountCreateResponse struct {
	Account AccountData `json:"account"`
}

// AccountData contains account details.
type AccountData struct {
	Data struct {
		ARN              string         `json:"arn"`
		CanonicalID      string         `json:"canonicalId"`
		ID               string         `json:"id"`
		EmailAddress     string         `json:"emailAddress"`
		Name             string         `json:"name"`
		CreateDate       string         `json:"createDate"`
		QuotaMax         int64          `json:"quotaMax"`
		AliasList        []string       `json:"aliasList"`
		OIDCPList        []string       `json:"oidcpList"`
		CustomAttributes map[string]any `json:"customAttributes"`
	} `json:"data"`
}

// AccountGetResponse represents the flat GetAccount API response.
type AccountGetResponse struct {
	ARN              string            `json:"arn"`
	CanonicalID      string            `json:"canonicalId"`
	ID               string            `json:"id"`
	EmailAddress     string            `json:"emailAddress"`
	Name             string            `json:"name"`
	CreateDate       string            `json:"createDate"`
	QuotaMax         int64             `json:"quotaMax"`
	CustomAttributes map[string]string `json:"customAttributes"`
}

// AccessKeyResponse represents access key generation response.
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

// CreateAccount creates a new Scality account.
func (c *IAMClient) CreateAccount(ctx context.Context, req AccountCreateRequest) (*AccountCreateResponse, error) {
	params := url.Values{}
	params.Set("name", req.Name)
	params.Set("emailAddress", req.EmailAddress)
	if req.QuotaMax > 0 {
		params.Set("quotaMax", fmt.Sprintf("%d", req.QuotaMax))
	}
	if req.ExternalAccountID != "" {
		params.Set("externalAccountId", req.ExternalAccountID)
	}

	body, statusCode, err := c.DoSignedRequest(ctx, "CreateAccount", params)
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

// GenerateAccountAccessKey generates S3 API credentials for an account.
func (c *IAMClient) GenerateAccountAccessKey(ctx context.Context, accountName string) (*AccessKeyResponse, error) {
	params := url.Values{}
	params.Set("AccountName", accountName)

	body, statusCode, err := c.DoSignedRequest(ctx, "GenerateAccountAccessKey", params)
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

// DeleteAccountAccessKey deletes an access key for an account (admin-level).
func (c *IAMClient) DeleteAccountAccessKey(ctx context.Context, accessKeyID, accountName string) error {
	params := url.Values{}
	params.Set("AccessKeyId", accessKeyID)
	params.Set("AccountName", accountName)

	body, statusCode, err := c.DoSignedRequest(ctx, "DeleteAccessKey", params)
	if err != nil {
		return err
	}

	if statusCode == 404 {
		return nil
	}

	if statusCode != 200 {
		return fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
	}

	return nil
}

// GetAccount retrieves account details (for Read/refresh).
func (c *IAMClient) GetAccount(ctx context.Context, accountName string) (*AccountGetResponse, error) {
	params := url.Values{}
	params.Set("accountName", accountName)

	body, statusCode, err := c.DoSignedRequest(ctx, "GetAccount", params)
	if err != nil {
		return nil, err
	}

	if statusCode == 404 {
		return nil, nil
	}

	if statusCode != 200 && statusCode != 201 {
		return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
	}

	var result AccountGetResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// AccountListEntry represents one account in a ListAccounts response.
// Mirrors the flat fields of AccountGetResponse.
type AccountListEntry struct {
	ARN              string            `json:"arn"`
	CanonicalID      string            `json:"canonicalId"`
	ID               string            `json:"id"`
	EmailAddress     string            `json:"emailAddress"`
	Name             string            `json:"name"`
	CreateDate       string            `json:"createDate"`
	QuotaMax         int64             `json:"quotaMax"`
	CustomAttributes map[string]string `json:"customAttributes,omitempty"`
}

type accountListPage struct {
	Accounts    []AccountListEntry `json:"accounts"`
	Marker      string             `json:"marker"`
	IsTruncated bool               `json:"isTruncated"`
}

// ListAccounts retrieves all Scality accounts, walking pagination.
// Uses admin credentials configured on the IAMClient.
func (c *IAMClient) ListAccounts(ctx context.Context) ([]AccountListEntry, error) {
	var all []AccountListEntry
	marker := ""
	for {
		params := url.Values{}
		if marker != "" {
			params.Set("marker", marker)
		}

		body, statusCode, err := c.DoSignedRequest(ctx, "ListAccounts", params)
		if err != nil {
			return nil, err
		}

		if statusCode != 200 && statusCode != 201 {
			return nil, fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
		}

		var page accountListPage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("failed to parse ListAccounts response: %w", err)
		}

		all = append(all, page.Accounts...)
		if !page.IsTruncated || page.Marker == "" {
			break
		}
		marker = page.Marker
	}
	return all, nil
}

// DeleteAccount deletes a Scality account.
func (c *IAMClient) DeleteAccount(ctx context.Context, accountName string) error {
	params := url.Values{}
	params.Set("AccountName", accountName)

	body, statusCode, err := c.DoSignedRequest(ctx, "DeleteAccount", params)
	if err != nil {
		return err
	}

	if statusCode == 404 {
		return nil
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

// UpdateAccountAttributes overwrites all custom attributes on an account.
func (c *IAMClient) UpdateAccountAttributes(ctx context.Context, accountName string, attrs map[string]string) error {
	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		return fmt.Errorf("failed to marshal custom attributes: %w", err)
	}

	params := url.Values{}
	params.Set("name", accountName)
	params.Set("customAttributes", string(attrsJSON))

	body, statusCode, err := c.DoSignedRequest(ctx, "UpdateAccountAttributes", params)
	if err != nil {
		return err
	}

	if statusCode != 200 {
		return fmt.Errorf("unexpected status %d: %s", statusCode, string(body))
	}

	return nil
}

// --- IAM XML Response Types ---

type createUserResponse struct {
	XMLName xml.Name `xml:"CreateUserResponse"`
	Result  struct {
		User iamUser `xml:"User"`
	} `xml:"CreateUserResult"`
}

type getUserResponse struct {
	XMLName xml.Name `xml:"GetUserResponse"`
	Result  struct {
		User iamUser `xml:"User"`
	} `xml:"GetUserResult"`
}

type iamUser struct {
	UserName string `xml:"UserName"`
	UserId   string `xml:"UserId"`
	Arn      string `xml:"Arn"`
	Path     string `xml:"Path"`
}

type getUserPolicyResponse struct {
	XMLName xml.Name `xml:"GetUserPolicyResponse"`
	Result  struct {
		UserName       string `xml:"UserName"`
		PolicyName     string `xml:"PolicyName"`
		PolicyDocument string `xml:"PolicyDocument"`
	} `xml:"GetUserPolicyResult"`
}

type createAccessKeyXMLResponse struct {
	XMLName xml.Name `xml:"CreateAccessKeyResponse"`
	Result  struct {
		AccessKey iamAccessKeyXML `xml:"AccessKey"`
	} `xml:"CreateAccessKeyResult"`
}

type listAccessKeysXMLResponse struct {
	XMLName xml.Name `xml:"ListAccessKeysResponse"`
	Result  struct {
		AccessKeyMetadata []iamAccessKeyMetadataXML `xml:"AccessKeyMetadata>member"`
	} `xml:"ListAccessKeysResult"`
}

type iamAccessKeyXML struct {
	UserName        string `xml:"UserName"`
	AccessKeyId     string `xml:"AccessKeyId"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	Status          string `xml:"Status"`
	CreateDate      string `xml:"CreateDate"`
}

type iamAccessKeyMetadataXML struct {
	UserName    string `xml:"UserName"`
	AccessKeyId string `xml:"AccessKeyId"`
	Status      string `xml:"Status"`
	CreateDate  string `xml:"CreateDate"`
}

type createGroupResponse struct {
	XMLName xml.Name `xml:"CreateGroupResponse"`
	Result  struct {
		Group iamGroup `xml:"Group"`
	} `xml:"CreateGroupResult"`
}

type getGroupResponse struct {
	XMLName xml.Name `xml:"GetGroupResponse"`
	Result  struct {
		Group iamGroup  `xml:"Group"`
		Users []iamUser `xml:"Users>member"`
	} `xml:"GetGroupResult"`
}

type iamGroup struct {
	GroupName string `xml:"GroupName"`
	GroupId   string `xml:"GroupId"`
	Arn       string `xml:"Arn"`
	Path      string `xml:"Path"`
}

// --- IAM User Operations (per-account auth) ---

func (c *IAMClient) CreateUser(ctx context.Context, accessKey, secretKey, userName string) (*iamUser, error) {
	params := url.Values{
		"Action":   {"CreateUser"},
		"UserName": {userName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	var resp createUserResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create user response: %w", err)
	}

	return &resp.Result.User, nil
}

func (c *IAMClient) GetUser(ctx context.Context, accessKey, secretKey, userName string) (*iamUser, error) {
	params := url.Values{
		"Action":   {"GetUser"},
		"UserName": {userName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	var resp getUserResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing get user response: %w", err)
	}

	return &resp.Result.User, nil
}

func (c *IAMClient) DeleteUser(ctx context.Context, accessKey, secretKey, userName string) error {
	params := url.Values{
		"Action":   {"DeleteUser"},
		"UserName": {userName},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

// UserListEntry represents one user in a ListUsers response.
type UserListEntry struct {
	UserName   string `xml:"UserName"`
	UserId     string `xml:"UserId"`
	Arn        string `xml:"Arn"`
	Path       string `xml:"Path"`
	CreateDate string `xml:"CreateDate"`
}

type listUsersResponse struct {
	XMLName xml.Name `xml:"ListUsersResponse"`
	Result  struct {
		Users       []UserListEntry `xml:"Users>member"`
		IsTruncated bool            `xml:"IsTruncated"`
		Marker      string          `xml:"Marker"`
	} `xml:"ListUsersResult"`
}

// ListUsers retrieves all IAM users in the calling account, walking pagination.
func (c *IAMClient) ListUsers(ctx context.Context, accessKey, secretKey string) ([]UserListEntry, error) {
	var all []UserListEntry
	marker := ""
	for {
		params := url.Values{
			"Action": {"ListUsers"},
		}
		if marker != "" {
			params.Set("Marker", marker)
		}

		body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
		if err != nil {
			return nil, fmt.Errorf("list users: %w", err)
		}

		var page listUsersResponse
		if err := xml.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parsing list users response: %w", err)
		}

		all = append(all, page.Result.Users...)
		if !page.Result.IsTruncated || page.Result.Marker == "" {
			break
		}
		marker = page.Result.Marker
	}
	return all, nil
}

// --- IAM User Policy Operations (per-account auth) ---

func (c *IAMClient) PutUserPolicy(ctx context.Context, accessKey, secretKey, userName, policyName, policyDocument string) error {
	params := url.Values{
		"Action":         {"PutUserPolicy"},
		"UserName":       {userName},
		"PolicyName":     {policyName},
		"PolicyDocument": {policyDocument},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("put user policy: %w", err)
	}

	return nil
}

func (c *IAMClient) GetUserPolicy(ctx context.Context, accessKey, secretKey, userName, policyName string) (string, error) {
	params := url.Values{
		"Action":     {"GetUserPolicy"},
		"UserName":   {userName},
		"PolicyName": {policyName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("get user policy: %w", err)
	}

	var resp getUserPolicyResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing get user policy response: %w", err)
	}

	decoded, err := url.QueryUnescape(resp.Result.PolicyDocument)
	if err != nil {
		return resp.Result.PolicyDocument, nil
	}

	return decoded, nil
}

func (c *IAMClient) DeleteUserPolicy(ctx context.Context, accessKey, secretKey, userName, policyName string) error {
	params := url.Values{
		"Action":     {"DeleteUserPolicy"},
		"UserName":   {userName},
		"PolicyName": {policyName},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("delete user policy: %w", err)
	}

	return nil
}

// --- IAM User Access Key Operations (per-account auth) ---

func (c *IAMClient) CreateUserAccessKey(ctx context.Context, accessKey, secretKey, userName string) (*iamAccessKeyXML, error) {
	params := url.Values{
		"Action":   {"CreateAccessKey"},
		"UserName": {userName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create access key: %w", err)
	}

	var resp createAccessKeyXMLResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create access key response: %w", err)
	}

	return &resp.Result.AccessKey, nil
}

func (c *IAMClient) ListUserAccessKeys(ctx context.Context, accessKey, secretKey, userName string) ([]iamAccessKeyMetadataXML, error) {
	params := url.Values{
		"Action":   {"ListAccessKeys"},
		"UserName": {userName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("list access keys: %w", err)
	}

	var resp listAccessKeysXMLResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing list access keys response: %w", err)
	}

	return resp.Result.AccessKeyMetadata, nil
}

func (c *IAMClient) DeleteUserAccessKey(ctx context.Context, accountAccessKey, accountSecretKey, userName, accessKeyId string) error {
	params := url.Values{
		"Action":      {"DeleteAccessKey"},
		"UserName":    {userName},
		"AccessKeyId": {accessKeyId},
	}

	_, err := c.doSignedRequest(ctx, accountAccessKey, accountSecretKey, params)
	if err != nil {
		return fmt.Errorf("delete access key: %w", err)
	}

	return nil
}

// --- Account Root Access Key Operations (per-account auth) ---

func (c *IAMClient) CreateRootAccessKey(ctx context.Context, accessKey, secretKey string) (*iamAccessKeyXML, error) {
	params := url.Values{
		"Action": {"CreateAccessKey"},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create root access key: %w", err)
	}

	var resp createAccessKeyXMLResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create access key response: %w", err)
	}

	return &resp.Result.AccessKey, nil
}

func (c *IAMClient) ListRootAccessKeys(ctx context.Context, accessKey, secretKey string) ([]iamAccessKeyMetadataXML, error) {
	params := url.Values{
		"Action": {"ListAccessKeys"},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("list root access keys: %w", err)
	}

	var resp listAccessKeysXMLResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing list access keys response: %w", err)
	}

	return resp.Result.AccessKeyMetadata, nil
}

func (c *IAMClient) DeleteRootAccessKey(ctx context.Context, accountAccessKey, accountSecretKey, accessKeyId string) error {
	params := url.Values{
		"Action":      {"DeleteAccessKey"},
		"AccessKeyId": {accessKeyId},
	}

	_, err := c.doSignedRequest(ctx, accountAccessKey, accountSecretKey, params)
	if err != nil {
		return fmt.Errorf("delete root access key: %w", err)
	}

	return nil
}

// --- IAM Group Operations (per-account auth) ---

func (c *IAMClient) CreateGroup(ctx context.Context, accessKey, secretKey, groupName string) (*iamGroup, error) {
	params := url.Values{
		"Action":    {"CreateGroup"},
		"GroupName": {groupName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}

	var resp createGroupResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create group response: %w", err)
	}

	return &resp.Result.Group, nil
}

func (c *IAMClient) GetGroup(ctx context.Context, accessKey, secretKey, groupName string) (*iamGroup, []iamUser, error) {
	params := url.Values{
		"Action":    {"GetGroup"},
		"GroupName": {groupName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("get group: %w", err)
	}

	var resp getGroupResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, nil, fmt.Errorf("parsing get group response: %w", err)
	}

	return &resp.Result.Group, resp.Result.Users, nil
}

func (c *IAMClient) DeleteGroup(ctx context.Context, accessKey, secretKey, groupName string) error {
	params := url.Values{
		"Action":    {"DeleteGroup"},
		"GroupName": {groupName},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	return nil
}

// GroupListEntry represents one group in a ListGroups response.
type GroupListEntry struct {
	GroupName  string `xml:"GroupName"`
	GroupId    string `xml:"GroupId"`
	Arn        string `xml:"Arn"`
	Path       string `xml:"Path"`
	CreateDate string `xml:"CreateDate"`
}

type listGroupsResponse struct {
	XMLName xml.Name `xml:"ListGroupsResponse"`
	Result  struct {
		Groups      []GroupListEntry `xml:"Groups>member"`
		IsTruncated bool             `xml:"IsTruncated"`
		Marker      string           `xml:"Marker"`
	} `xml:"ListGroupsResult"`
}

// ListGroups retrieves all IAM groups in the calling account, walking pagination.
func (c *IAMClient) ListGroups(ctx context.Context, accessKey, secretKey string) ([]GroupListEntry, error) {
	var all []GroupListEntry
	marker := ""
	for {
		params := url.Values{
			"Action": {"ListGroups"},
		}
		if marker != "" {
			params.Set("Marker", marker)
		}

		body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
		if err != nil {
			return nil, fmt.Errorf("list groups: %w", err)
		}

		var page listGroupsResponse
		if err := xml.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parsing list groups response: %w", err)
		}

		all = append(all, page.Result.Groups...)
		if !page.Result.IsTruncated || page.Result.Marker == "" {
			break
		}
		marker = page.Result.Marker
	}
	return all, nil
}

func (c *IAMClient) AddUserToGroup(ctx context.Context, accessKey, secretKey, groupName, userName string) error {
	params := url.Values{
		"Action":    {"AddUserToGroup"},
		"GroupName": {groupName},
		"UserName":  {userName},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("add user to group: %w", err)
	}

	return nil
}

func (c *IAMClient) RemoveUserFromGroup(ctx context.Context, accessKey, secretKey, groupName, userName string) error {
	params := url.Values{
		"Action":    {"RemoveUserFromGroup"},
		"GroupName": {groupName},
		"UserName":  {userName},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("remove user from group: %w", err)
	}

	return nil
}
