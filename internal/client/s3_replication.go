package client

import (
	"context"
	"encoding/xml"
	"fmt"
)

type replicationConfiguration struct {
	XMLName xml.Name          `xml:"ReplicationConfiguration"`
	Role    string            `xml:"Role"`
	Rules   []replicationRule `xml:"Rule"`
}

type replicationRule struct {
	ID          string                 `xml:"ID,omitempty"`
	Status      string                 `xml:"Status"`
	Prefix      string                 `xml:"Prefix"`
	Destination replicationDestination `xml:"Destination"`
}

type replicationDestination struct {
	Bucket       string `xml:"Bucket"`
	StorageClass string `xml:"StorageClass,omitempty"`
}

type ReplicationRule struct {
	ID                      string
	Status                  string
	Prefix                  string
	DestinationBucket       string
	DestinationStorageClass string
}

func (c *S3Client) GetBucketReplication(ctx context.Context, accessKey, secretKey, bucket string) (string, []ReplicationRule, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, bucket, "replication", nil, nil)
	if err != nil {
		return "", nil, fmt.Errorf("get bucket replication: %w", err)
	}

	if statusCode == 404 {
		return "", nil, nil
	}

	if statusCode != 200 {
		return "", nil, c.formatS3Error(body, statusCode, "get bucket replication")
	}

	var config replicationConfiguration
	if err := xml.Unmarshal(body, &config); err != nil {
		return "", nil, fmt.Errorf("parsing replication response: %w", err)
	}

	rules := make([]ReplicationRule, 0, len(config.Rules))
	for _, r := range config.Rules {
		rules = append(rules, ReplicationRule{
			ID:                      r.ID,
			Status:                  r.Status,
			Prefix:                  r.Prefix,
			DestinationBucket:       r.Destination.Bucket,
			DestinationStorageClass: r.Destination.StorageClass,
		})
	}

	return config.Role, rules, nil
}

func (c *S3Client) PutBucketReplication(ctx context.Context, accessKey, secretKey, bucket, role string, rules []ReplicationRule) error {
	config := replicationConfiguration{Role: role}

	for _, r := range rules {
		config.Rules = append(config.Rules, replicationRule{
			ID:     r.ID,
			Status: r.Status,
			Prefix: r.Prefix,
			Destination: replicationDestination{
				Bucket:       r.DestinationBucket,
				StorageClass: r.DestinationStorageClass,
			},
		})
	}

	xmlBody, err := xml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling replication config: %w", err)
	}

	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "replication", xmlBody, nil)
	if err != nil {
		return fmt.Errorf("put bucket replication: %w", err)
	}

	if statusCode != 200 {
		return c.formatS3Error(respBody, statusCode, "put bucket replication")
	}

	return nil
}

func (c *S3Client) DeleteBucketReplication(ctx context.Context, accessKey, secretKey, bucket string) error {
	body, statusCode, err := c.doRequest(ctx, "DELETE", accessKey, secretKey, bucket, "replication", nil, nil)
	if err != nil {
		return fmt.Errorf("delete bucket replication: %w", err)
	}

	if statusCode != 204 && statusCode != 404 {
		return c.formatS3Error(body, statusCode, "delete bucket replication")
	}

	return nil
}
