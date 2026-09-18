package client

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ErrServerAccessLoggingDisabled is returned by the bucket-logging methods when
// the cluster responds 501 NotImplemented — server access logging is not enabled
// at the cluster level. Callers detect it with errors.Is and surface actionable
// guidance instead of a raw API error.
var ErrServerAccessLoggingDisabled = errors.New("server access logging is not enabled on this cluster")

// bucketLoggingXMLNS is the S3 schema namespace CloudServer requires on the
// request body — without it, the empty (disable) form is rejected as MalformedXML.
const bucketLoggingXMLNS = "http://s3.amazonaws.com/doc/2006-03-01/"

type bucketLoggingStatus struct {
	XMLName xml.Name        `xml:"BucketLoggingStatus"`
	Xmlns   string          `xml:"xmlns,attr,omitempty"`
	Enabled *loggingEnabled `xml:"LoggingEnabled,omitempty"`
}

type loggingEnabled struct {
	TargetBucket string `xml:"TargetBucket"`
	TargetPrefix string `xml:"TargetPrefix"`
}

// LoggingConfig is the server access logging configuration for a bucket.
type LoggingConfig struct {
	TargetBucket string
	TargetPrefix string
}

// GetBucketLogging returns the logging configuration for a bucket. It returns
// (nil, nil) when the feature is enabled but logging is not configured, and
// ErrServerAccessLoggingDisabled when the feature is disabled cluster-side.
func (c *S3Client) GetBucketLogging(ctx context.Context, accessKey, secretKey, bucket string) (*LoggingConfig, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, bucket, "logging", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get bucket logging: %w", err)
	}

	tflog.Debug(ctx, "GetBucketLogging raw response", map[string]interface{}{
		"status_code": statusCode,
		"body":        string(body),
	})

	if statusCode == 501 {
		return nil, ErrServerAccessLoggingDisabled
	}

	if statusCode != 200 {
		return nil, c.formatS3Error(body, statusCode, "get bucket logging")
	}

	var status bucketLoggingStatus
	if err := xml.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("parsing logging response: %w", err)
	}

	if status.Enabled == nil || status.Enabled.TargetBucket == "" {
		return nil, nil
	}

	return &LoggingConfig{
		TargetBucket: status.Enabled.TargetBucket,
		TargetPrefix: status.Enabled.TargetPrefix,
	}, nil
}

// PutBucketLogging enables server access logging on a bucket.
func (c *S3Client) PutBucketLogging(ctx context.Context, accessKey, secretKey, bucket string, cfg LoggingConfig) error {
	status := bucketLoggingStatus{
		Xmlns: bucketLoggingXMLNS,
		Enabled: &loggingEnabled{
			TargetBucket: cfg.TargetBucket,
			TargetPrefix: cfg.TargetPrefix,
		},
	}

	xmlBody, err := xml.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshaling logging config: %w", err)
	}

	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "logging", xmlBody, nil)
	if err != nil {
		return fmt.Errorf("put bucket logging: %w", err)
	}

	tflog.Debug(ctx, "PutBucketLogging response", map[string]interface{}{
		"status_code": statusCode,
		"body":        string(respBody),
	})

	if statusCode == 501 {
		return ErrServerAccessLoggingDisabled
	}

	if statusCode != 200 {
		return c.formatS3Error(respBody, statusCode, "put bucket logging")
	}

	return nil
}

// DeleteBucketLogging disables server access logging by putting an empty
// BucketLoggingStatus.
func (c *S3Client) DeleteBucketLogging(ctx context.Context, accessKey, secretKey, bucket string) error {
	xmlBody, err := xml.Marshal(bucketLoggingStatus{Xmlns: bucketLoggingXMLNS})
	if err != nil {
		return fmt.Errorf("marshaling empty logging config: %w", err)
	}

	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "logging", xmlBody, nil)
	if err != nil {
		return fmt.Errorf("disable bucket logging: %w", err)
	}

	if statusCode == 501 {
		return ErrServerAccessLoggingDisabled
	}

	if statusCode != 200 {
		return c.formatS3Error(respBody, statusCode, "disable bucket logging")
	}

	return nil
}
