package client

import (
	"bytes"
	"context"
	"crypto/md5" // #nosec G501 -- MD5 required by S3 Content-MD5 protocol header
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const s3Service = "s3"

type S3Client struct {
	Endpoint   string
	Region     string
	HTTPClient *http.Client
	// SessionToken, when set, is sent as X-Amz-Security-Token — set when the
	// credentials are STS assumed-role credentials.
	SessionToken string
	// bucketLocks serializes mutating requests to the same bucket. The backend
	// persists all bucket subresources into one record via read-modify-write, so
	// concurrent writes to distinct subresources clobber one another. A pointer
	// so assumed-role client clones (clone := *c) share the same lock set.
	bucketLocks *keyedMutex
}

func NewS3Client(endpoint string, insecureSkipVerify bool) *S3Client {
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

	return &S3Client{
		Endpoint:    strings.TrimRight(endpoint, "/"),
		Region:      awsRegion,
		HTTPClient:  httpClient,
		bucketLocks: &keyedMutex{},
	}
}

type s3ErrorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

func (c *S3Client) doRequest(ctx context.Context, method, accessKey, secretKey, bucket, subresource string, body []byte, extraHeaders map[string]string) ([]byte, int, error) {
	// Serialize mutating requests to the same bucket; reads need no lock.
	if c.bucketLocks != nil && bucket != "" && method != http.MethodGet && method != http.MethodHead {
		c.bucketLocks.lock(bucket)
		defer c.bucketLocks.unlock(bucket)
	}

	parsedURL, err := url.Parse(c.Endpoint)
	if err != nil {
		return nil, 0, fmt.Errorf("parsing endpoint: %w", err)
	}

	host := parsedURL.Host
	escapedBucket := url.PathEscape(bucket)
	canonicalURI := "/" + escapedBucket

	requestURL := c.Endpoint + "/" + escapedBucket
	canonicalQueryString := ""
	if subresource != "" {
		requestURL += "?" + subresource
		canonicalQueryString = subresource + "="
	}

	now := time.Now().UTC()
	amzdate := now.Format(awsDateFormat)
	datestamp := now.Format(awsDateStampFormat)

	payloadHash := sha256Hex(body)

	headerMap := map[string]string{
		"host":                 host,
		"x-amz-content-sha256": payloadHash,
		"x-amz-date":           amzdate,
	}
	if c.SessionToken != "" {
		headerMap["x-amz-security-token"] = c.SessionToken
	}

	for k, v := range extraHeaders {
		headerMap[strings.ToLower(k)] = v
	}

	headerNames := make([]string, 0, len(headerMap))
	for k := range headerMap {
		headerNames = append(headerNames, k)
	}
	sort.Strings(headerNames)

	var canonicalHeaders strings.Builder
	for _, name := range headerNames {
		canonicalHeaders.WriteString(name + ":" + headerMap[name] + "\n")
	}
	signedHeaders := strings.Join(headerNames, ";")

	canonicalRequest := strings.Join([]string{
		method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := fmt.Sprintf("%s/%s/%s/%s", datestamp, c.Region, s3Service, awsRequestType)
	stringToSign := strings.Join([]string{
		awsAlgorithm,
		amzdate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := getSignatureKey(secretKey, datestamp, c.Region, s3Service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		awsAlgorithm, accessKey, credentialScope, signedHeaders, signature)

	httpReq, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("X-Amz-Date", amzdate)
	httpReq.Header.Set("X-Amz-Content-Sha256", payloadHash)
	httpReq.Header.Set("Authorization", authHeader)
	if c.SessionToken != "" {
		httpReq.Header.Set("X-Amz-Security-Token", c.SessionToken)
	}

	for k, v := range extraHeaders {
		httpReq.Header.Set(k, v)
	}

	if len(body) > 0 {
		hasContentType := false
		for k := range extraHeaders {
			if strings.EqualFold(k, "content-type") {
				hasContentType = true
				break
			}
		}
		if !hasContentType {
			httpReq.Header.Set("Content-Type", "application/xml")
		}
		md5Hash := md5.Sum(body) // #nosec G401 -- S3 Content-MD5 header is required by the protocol
		httpReq.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(md5Hash[:]))
	}

	respBody, statusCode, err := doWithRetry(ctx, func() (*http.Response, error) {
		req := httpReq.Clone(ctx)
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		return c.HTTPClient.Do(req)
	})
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}

	return respBody, statusCode, nil
}

func (c *S3Client) formatS3Error(respBody []byte, statusCode int, operation string) error {
	var s3Err s3ErrorResponse
	if xmlErr := xml.Unmarshal(respBody, &s3Err); xmlErr == nil && s3Err.Code != "" {
		return fmt.Errorf("%s: %s: %s", operation, s3Err.Code, s3Err.Message)
	}
	return fmt.Errorf("%s failed (status %d): %s", operation, statusCode, string(respBody))
}
