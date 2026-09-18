package client

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	stsService    = "sts"
	stsAPIVersion = "2011-06-15"
)

// AssumedCredentials are the temporary credentials returned by STS AssumeRole.
type AssumedCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      string
}

// STSClient calls the Scality STS API. STS shares the IAM/Vault endpoint; nginx
// routes on the SigV4 credential-scope service (".../sts/aws4_request").
type STSClient struct {
	Endpoint   string
	Region     string
	HTTPClient *http.Client
}

// NewSTSClient creates a new Scality STS client. Region defaults to awsRegion.
func NewSTSClient(endpoint string, insecureSkipVerify bool) *STSClient {
	httpClient := &http.Client{Timeout: defaultHTTPTimeout}

	if insecureSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // #nosec G402 -- gated on the user-set insecure_skip_verify provider attribute
			},
		}
	}

	return &STSClient{
		Endpoint:   strings.TrimRight(endpoint, "/"),
		Region:     awsRegion,
		HTTPClient: httpClient,
	}
}

type assumeRoleResponse struct {
	XMLName xml.Name `xml:"AssumeRoleResponse"`
	Result  struct {
		Credentials struct {
			AccessKeyId     string `xml:"AccessKeyId"`
			SecretAccessKey string `xml:"SecretAccessKey"`
			SessionToken    string `xml:"SessionToken"`
			Expiration      string `xml:"Expiration"`
		} `xml:"Credentials"`
	} `xml:"AssumeRoleResult"`
}

// AssumeRole exchanges the caller's long-term credentials for temporary
// credentials scoped to roleArn. Signed with service "sts".
func (c *STSClient) AssumeRole(ctx context.Context, callerAccessKey, callerSecretKey, roleArn, sessionName string) (*AssumedCredentials, error) {
	params := url.Values{}
	params.Set("Action", "AssumeRole")
	params.Set("Version", stsAPIVersion)
	params.Set("RoleArn", roleArn)
	params.Set("RoleSessionName", sessionName)
	body := params.Encode()

	parsedURL, err := url.Parse(c.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint: %w", err)
	}
	host := parsedURL.Host
	now := time.Now().UTC()
	datestamp := now.Format(awsDateStampFormat)
	amzdate := now.Format(awsDateFormat)

	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-date:%s\n", host, amzdate)
	signedHeaders := "host;x-amz-date"
	canonicalRequest := strings.Join([]string{
		httpMethodPost, canonicalURI, "", canonicalHeaders, signedHeaders, sha256Hex([]byte(body)),
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/%s/%s", datestamp, c.Region, stsService, awsRequestType)
	stringToSign := strings.Join([]string{
		awsAlgorithm, amzdate, credentialScope, sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	signingKey := getSignatureKey(callerSecretKey, datestamp, c.Region, stsService)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		awsAlgorithm, callerAccessKey, credentialScope, signedHeaders, signature)

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
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var e iamErrorResponse
		if xmlErr := xml.Unmarshal(respBody, &e); xmlErr == nil && e.Error.Code != "" {
			return nil, &APIError{Code: e.Error.Code, Message: e.Error.Message, StatusCode: resp.StatusCode}
		}
		return nil, fmt.Errorf("STS request failed (status %d)", resp.StatusCode)
	}

	var parsed assumeRoleResponse
	if err := xml.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parsing assume role response: %w", err)
	}
	creds := parsed.Result.Credentials
	if creds.AccessKeyId == "" || creds.SessionToken == "" {
		return nil, fmt.Errorf("assume role: response missing credentials")
	}

	return &AssumedCredentials{
		AccessKeyID:     creds.AccessKeyId,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Expiration:      creds.Expiration,
	}, nil
}
