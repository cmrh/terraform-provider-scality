package client

import (
	"bytes"
	"context"
	"crypto/md5"
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
	HTTPClient *http.Client
}

func NewS3Client(endpoint string, insecureSkipVerify bool) *S3Client {
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

	return &S3Client{
		Endpoint:   strings.TrimRight(endpoint, "/"),
		HTTPClient: httpClient,
	}
}

type s3ErrorResponse struct {
	XMLName xml.Name `xml:"Error"`
	Code    string   `xml:"Code"`
	Message string   `xml:"Message"`
}

func (c *S3Client) doRequest(ctx context.Context, method, accessKey, secretKey, bucket, subresource string, body []byte, extraHeaders map[string]string) ([]byte, int, error) {
	parsedURL, err := url.Parse(c.Endpoint)
	if err != nil {
		return nil, 0, fmt.Errorf("parsing endpoint: %w", err)
	}

	host := parsedURL.Host
	canonicalURI := "/" + bucket

	requestURL := c.Endpoint + "/" + bucket
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

	credentialScope := fmt.Sprintf("%s/%s/%s/%s", datestamp, awsRegion, s3Service, awsRequestType)
	stringToSign := strings.Join([]string{
		awsAlgorithm,
		amzdate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := getSignatureKey(secretKey, datestamp, awsRegion, s3Service)
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
		md5Hash := md5.Sum(body)
		httpReq.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(md5Hash[:]))
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("reading response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

func (c *S3Client) formatS3Error(respBody []byte, statusCode int, operation string) error {
	var s3Err s3ErrorResponse
	if xmlErr := xml.Unmarshal(respBody, &s3Err); xmlErr == nil && s3Err.Code != "" {
		return fmt.Errorf("%s: %s: %s", operation, s3Err.Code, s3Err.Message)
	}
	return fmt.Errorf("%s failed (status %d): %s", operation, statusCode, string(respBody))
}
