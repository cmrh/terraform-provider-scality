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
		"host":                  host,
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
		httpReq.Header.Set("Content-Type", "application/xml")
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

func (c *S3Client) CreateBucket(ctx context.Context, accessKey, secretKey, bucket string) error {
	body, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "", nil, nil)
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}

	if statusCode == 409 {
		return fmt.Errorf("bucket %q already exists", bucket)
	}

	if statusCode != 200 {
		return c.formatS3Error(body, statusCode, "create bucket")
	}

	return nil
}

func (c *S3Client) HeadBucket(ctx context.Context, accessKey, secretKey, bucket string) (bool, error) {
	_, statusCode, err := c.doRequest(ctx, "HEAD", accessKey, secretKey, bucket, "", nil, nil)
	if err != nil {
		return false, fmt.Errorf("head bucket: %w", err)
	}

	switch {
	case statusCode == 200:
		return true, nil
	case statusCode == 404:
		return false, nil
	case statusCode == 403:
		return false, fmt.Errorf("access denied to bucket %q", bucket)
	default:
		return false, fmt.Errorf("head bucket unexpected status %d", statusCode)
	}
}

func (c *S3Client) DeleteBucket(ctx context.Context, accessKey, secretKey, bucket string) error {
	body, statusCode, err := c.doRequest(ctx, "DELETE", accessKey, secretKey, bucket, "", nil, nil)
	if err != nil {
		return fmt.Errorf("delete bucket: %w", err)
	}

	if statusCode == 404 {
		return nil
	}

	if statusCode != 204 {
		return c.formatS3Error(body, statusCode, "delete bucket")
	}

	return nil
}

// --- Versioning ---

type versioningConfiguration struct {
	XMLName xml.Name `xml:"VersioningConfiguration"`
	Status  string   `xml:"Status,omitempty"`
}

func (c *S3Client) GetBucketVersioning(ctx context.Context, accessKey, secretKey, bucket string) (string, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, bucket, "versioning", nil, nil)
	if err != nil {
		return "", fmt.Errorf("get bucket versioning: %w", err)
	}

	if statusCode != 200 {
		return "", c.formatS3Error(body, statusCode, "get bucket versioning")
	}

	var config versioningConfiguration
	if err := xml.Unmarshal(body, &config); err != nil {
		return "", fmt.Errorf("parsing versioning response: %w", err)
	}

	return config.Status, nil
}

func (c *S3Client) PutBucketVersioning(ctx context.Context, accessKey, secretKey, bucket, status string) error {
	config := versioningConfiguration{Status: status}

	xmlBody, err := xml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling versioning config: %w", err)
	}

	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "versioning", xmlBody, nil)
	if err != nil {
		return fmt.Errorf("put bucket versioning: %w", err)
	}

	if statusCode != 200 {
		return c.formatS3Error(respBody, statusCode, "put bucket versioning")
	}

	return nil
}

// --- Tagging ---

type tagging struct {
	XMLName xml.Name `xml:"Tagging"`
	TagSet  tagSet   `xml:"TagSet"`
}

type tagSet struct {
	Tags []tag `xml:"Tag"`
}

type tag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

func (c *S3Client) GetBucketTagging(ctx context.Context, accessKey, secretKey, bucket string) (map[string]string, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, bucket, "tagging", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get bucket tagging: %w", err)
	}

	if statusCode == 404 {
		return nil, nil
	}

	if statusCode != 200 {
		return nil, c.formatS3Error(body, statusCode, "get bucket tagging")
	}

	var result tagging
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing tagging response: %w", err)
	}

	tags := make(map[string]string, len(result.TagSet.Tags))
	for _, t := range result.TagSet.Tags {
		tags[t.Key] = t.Value
	}

	return tags, nil
}

func (c *S3Client) PutBucketTagging(ctx context.Context, accessKey, secretKey, bucket string, tags map[string]string) error {
	tagList := make([]tag, 0, len(tags))
	for k, v := range tags {
		tagList = append(tagList, tag{Key: k, Value: v})
	}

	t := tagging{TagSet: tagSet{Tags: tagList}}

	xmlBody, err := xml.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshaling tagging config: %w", err)
	}

	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "tagging", xmlBody, nil)
	if err != nil {
		return fmt.Errorf("put bucket tagging: %w", err)
	}

	if statusCode != 200 && statusCode != 204 {
		return c.formatS3Error(respBody, statusCode, "put bucket tagging")
	}

	return nil
}

func (c *S3Client) DeleteBucketTagging(ctx context.Context, accessKey, secretKey, bucket string) error {
	body, statusCode, err := c.doRequest(ctx, "DELETE", accessKey, secretKey, bucket, "tagging", nil, nil)
	if err != nil {
		return fmt.Errorf("delete bucket tagging: %w", err)
	}

	if statusCode != 204 && statusCode != 404 {
		return c.formatS3Error(body, statusCode, "delete bucket tagging")
	}

	return nil
}
