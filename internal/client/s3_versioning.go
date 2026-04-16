package client

import (
	"context"
	"encoding/xml"
	"fmt"
)

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
