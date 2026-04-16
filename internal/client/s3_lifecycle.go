package client

import (
	"context"
	"encoding/xml"
	"fmt"
)

type lifecycleConfiguration struct {
	XMLName xml.Name        `xml:"LifecycleConfiguration"`
	Rules   []lifecycleRule `xml:"Rule"`
}

type lifecycleRule struct {
	ID                             string                         `xml:"ID"`
	Status                         string                         `xml:"Status"`
	Filter                         *lifecycleFilter               `xml:"Filter,omitempty"`
	Expiration                     *lifecycleExpiration           `xml:"Expiration,omitempty"`
	NoncurrentVersionExpiration    *lifecycleNoncurrentExpiration `xml:"NoncurrentVersionExpiration,omitempty"`
	AbortIncompleteMultipartUpload *lifecycleAbortIncomplete      `xml:"AbortIncompleteMultipartUpload,omitempty"`
}

type lifecycleFilter struct {
	Prefix string `xml:"Prefix"`
}

type lifecycleExpiration struct {
	Days int    `xml:"Days,omitempty"`
	Date string `xml:"Date,omitempty"`
}

type lifecycleNoncurrentExpiration struct {
	NoncurrentDays int `xml:"NoncurrentDays"`
}

type lifecycleAbortIncomplete struct {
	DaysAfterInitiation int `xml:"DaysAfterInitiation"`
}

type LifecycleRule struct {
	ID                                 string
	Status                             string
	Prefix                             string
	ExpirationDays                     int
	ExpirationDate                     string
	NoncurrentVersionExpirationDays    int
	AbortIncompleteMultipartUploadDays int
}

func (c *S3Client) GetBucketLifecycle(ctx context.Context, accessKey, secretKey, bucket string) ([]LifecycleRule, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, bucket, "lifecycle", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get bucket lifecycle: %w", err)
	}

	if statusCode == 404 {
		return nil, nil
	}

	if statusCode != 200 {
		return nil, c.formatS3Error(body, statusCode, "get bucket lifecycle")
	}

	var config lifecycleConfiguration
	if err := xml.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("parsing lifecycle response: %w", err)
	}

	rules := make([]LifecycleRule, 0, len(config.Rules))
	for _, r := range config.Rules {
		rule := LifecycleRule{
			ID:     r.ID,
			Status: r.Status,
		}
		if r.Filter != nil {
			rule.Prefix = r.Filter.Prefix
		}
		if r.Expiration != nil {
			rule.ExpirationDays = r.Expiration.Days
			rule.ExpirationDate = r.Expiration.Date
		}
		if r.NoncurrentVersionExpiration != nil {
			rule.NoncurrentVersionExpirationDays = r.NoncurrentVersionExpiration.NoncurrentDays
		}
		if r.AbortIncompleteMultipartUpload != nil {
			rule.AbortIncompleteMultipartUploadDays = r.AbortIncompleteMultipartUpload.DaysAfterInitiation
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (c *S3Client) PutBucketLifecycle(ctx context.Context, accessKey, secretKey, bucket string, rules []LifecycleRule) error {
	config := lifecycleConfiguration{}

	for _, r := range rules {
		xmlRule := lifecycleRule{
			ID:     r.ID,
			Status: r.Status,
			Filter: &lifecycleFilter{Prefix: r.Prefix},
		}
		if r.ExpirationDays > 0 || r.ExpirationDate != "" {
			xmlRule.Expiration = &lifecycleExpiration{
				Days: r.ExpirationDays,
				Date: r.ExpirationDate,
			}
		}
		if r.NoncurrentVersionExpirationDays > 0 {
			xmlRule.NoncurrentVersionExpiration = &lifecycleNoncurrentExpiration{
				NoncurrentDays: r.NoncurrentVersionExpirationDays,
			}
		}
		if r.AbortIncompleteMultipartUploadDays > 0 {
			xmlRule.AbortIncompleteMultipartUpload = &lifecycleAbortIncomplete{
				DaysAfterInitiation: r.AbortIncompleteMultipartUploadDays,
			}
		}
		config.Rules = append(config.Rules, xmlRule)
	}

	xmlBody, err := xml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling lifecycle config: %w", err)
	}

	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "lifecycle", xmlBody, nil)
	if err != nil {
		return fmt.Errorf("put bucket lifecycle: %w", err)
	}

	if statusCode != 200 {
		return c.formatS3Error(respBody, statusCode, "put bucket lifecycle")
	}

	return nil
}

func (c *S3Client) DeleteBucketLifecycle(ctx context.Context, accessKey, secretKey, bucket string) error {
	body, statusCode, err := c.doRequest(ctx, "DELETE", accessKey, secretKey, bucket, "lifecycle", nil, nil)
	if err != nil {
		return fmt.Errorf("delete bucket lifecycle: %w", err)
	}

	if statusCode != 204 && statusCode != 404 {
		return c.formatS3Error(body, statusCode, "delete bucket lifecycle")
	}

	return nil
}
