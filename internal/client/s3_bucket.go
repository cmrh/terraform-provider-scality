package client

import (
	"context"
	"encoding/xml"
	"fmt"
)

// BucketListEntry represents one bucket in a ListBuckets response.
type BucketListEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type listBucketsResponse struct {
	XMLName xml.Name          `xml:"ListAllMyBucketsResult"`
	Buckets []BucketListEntry `xml:"Buckets>Bucket"`
}

// ListBuckets returns all buckets owned by the calling account.
// S3 ListBuckets is a single-call API — no pagination.
func (c *S3Client) ListBuckets(ctx context.Context, accessKey, secretKey string) ([]BucketListEntry, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, "", "", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list buckets: %w", err)
	}

	if statusCode != 200 {
		return nil, c.formatS3Error(body, statusCode, "list buckets")
	}

	var result listBucketsResponse
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing list buckets response: %w", err)
	}

	return result.Buckets, nil
}

func (c *S3Client) CreateBucket(ctx context.Context, accessKey, secretKey, bucket string, objectLockEnabled bool) error {
	var extraHeaders map[string]string
	if objectLockEnabled {
		extraHeaders = map[string]string{
			"x-amz-bucket-object-lock-enabled": "true",
		}
	}

	body, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "", nil, extraHeaders)
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

	switch statusCode {
	case 200:
		return true, nil
	case 404:
		return false, nil
	case 403:
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
