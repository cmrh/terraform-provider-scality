package client

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type serverSideEncryptionConfiguration struct {
	XMLName xml.Name                   `xml:"ServerSideEncryptionConfiguration"`
	Rules   []serverSideEncryptionRule `xml:"Rule"`
}

type serverSideEncryptionRule struct {
	Apply serverSideEncryptionByDefault `xml:"ApplyServerSideEncryptionByDefault"`
}

type serverSideEncryptionByDefault struct {
	SSEAlgorithm   string `xml:"SSEAlgorithm"`
	KMSMasterKeyID string `xml:"KMSMasterKeyID,omitempty"`
}

type EncryptionConfig struct {
	SSEAlgorithm   string
	KMSMasterKeyID string
}

func (c *S3Client) GetBucketEncryption(ctx context.Context, accessKey, secretKey, bucket string) (*EncryptionConfig, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, bucket, "encryption", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get bucket encryption: %w", err)
	}

	tflog.Debug(ctx, "GetBucketEncryption raw response", map[string]interface{}{
		"status_code": statusCode,
		"body":        string(body),
	})

	if statusCode == 404 {
		return nil, nil
	}

	if statusCode != 200 {
		return nil, c.formatS3Error(body, statusCode, "get bucket encryption")
	}

	var config serverSideEncryptionConfiguration
	if err := xml.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("parsing encryption response: %w", err)
	}

	tflog.Debug(ctx, "GetBucketEncryption parsed config", map[string]interface{}{
		"rules_count": len(config.Rules),
	})

	if len(config.Rules) == 0 {
		return nil, nil
	}

	return &EncryptionConfig{
		SSEAlgorithm:   config.Rules[0].Apply.SSEAlgorithm,
		KMSMasterKeyID: config.Rules[0].Apply.KMSMasterKeyID,
	}, nil
}

func (c *S3Client) PutBucketEncryption(ctx context.Context, accessKey, secretKey, bucket string, cfg EncryptionConfig) error {
	config := serverSideEncryptionConfiguration{
		Rules: []serverSideEncryptionRule{{
			Apply: serverSideEncryptionByDefault{
				SSEAlgorithm:   cfg.SSEAlgorithm,
				KMSMasterKeyID: cfg.KMSMasterKeyID,
			},
		}},
	}

	xmlBody, err := xml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling encryption config: %w", err)
	}

	tflog.Debug(ctx, "PutBucketEncryption request", map[string]interface{}{
		"body": string(xmlBody),
	})

	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "encryption", xmlBody, nil)
	if err != nil {
		return fmt.Errorf("put bucket encryption: %w", err)
	}

	tflog.Debug(ctx, "PutBucketEncryption response", map[string]interface{}{
		"status_code": statusCode,
		"body":        string(respBody),
	})

	if statusCode != 200 {
		return c.formatS3Error(respBody, statusCode, "put bucket encryption")
	}

	return nil
}

func (c *S3Client) DeleteBucketEncryption(ctx context.Context, accessKey, secretKey, bucket string) error {
	body, statusCode, err := c.doRequest(ctx, "DELETE", accessKey, secretKey, bucket, "encryption", nil, nil)
	if err != nil {
		return fmt.Errorf("delete bucket encryption: %w", err)
	}

	if statusCode != 204 && statusCode != 404 {
		return c.formatS3Error(body, statusCode, "delete bucket encryption")
	}

	return nil
}
