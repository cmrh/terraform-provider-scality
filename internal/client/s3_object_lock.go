package client

import (
	"context"
	"encoding/xml"
	"fmt"
)

type objectLockConfiguration struct {
	XMLName           xml.Name        `xml:"ObjectLockConfiguration"`
	ObjectLockEnabled string          `xml:"ObjectLockEnabled"`
	Rule              *objectLockRule `xml:"Rule,omitempty"`
}

type objectLockRule struct {
	DefaultRetention objectLockRetention `xml:"DefaultRetention"`
}

type objectLockRetention struct {
	Mode  string `xml:"Mode"`
	Days  int    `xml:"Days,omitempty"`
	Years int    `xml:"Years,omitempty"`
}

type ObjectLockConfig struct {
	Enabled        bool
	RetentionMode  string
	RetentionDays  int
	RetentionYears int
}

func (c *S3Client) GetObjectLockConfiguration(ctx context.Context, accessKey, secretKey, bucket string) (*ObjectLockConfig, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, bucket, "object-lock", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get object lock config: %w", err)
	}

	if statusCode == 404 {
		return nil, nil
	}

	if statusCode != 200 {
		return nil, c.formatS3Error(body, statusCode, "get object lock config")
	}

	var config objectLockConfiguration
	if err := xml.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("parsing object lock response: %w", err)
	}

	result := &ObjectLockConfig{
		Enabled: config.ObjectLockEnabled == "Enabled",
	}

	if config.Rule != nil {
		result.RetentionMode = config.Rule.DefaultRetention.Mode
		result.RetentionDays = config.Rule.DefaultRetention.Days
		result.RetentionYears = config.Rule.DefaultRetention.Years
	}

	return result, nil
}

func (c *S3Client) PutObjectLockConfiguration(ctx context.Context, accessKey, secretKey, bucket string, cfg ObjectLockConfig) error {
	config := objectLockConfiguration{
		ObjectLockEnabled: "Enabled",
	}

	if cfg.RetentionMode != "" {
		config.Rule = &objectLockRule{
			DefaultRetention: objectLockRetention{
				Mode:  cfg.RetentionMode,
				Days:  cfg.RetentionDays,
				Years: cfg.RetentionYears,
			},
		}
	}

	xmlBody, err := xml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling object lock config: %w", err)
	}

	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "object-lock", xmlBody, nil)
	if err != nil {
		return fmt.Errorf("put object lock config: %w", err)
	}

	if statusCode != 200 {
		return c.formatS3Error(respBody, statusCode, "put object lock config")
	}

	return nil
}
