package client

import (
	"context"
	"fmt"
)

func (c *S3Client) GetBucketPolicy(ctx context.Context, accessKey, secretKey, bucket string) (string, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, bucket, "policy", nil, nil)
	if err != nil {
		return "", fmt.Errorf("get bucket policy: %w", err)
	}

	if statusCode == 404 {
		return "", nil
	}

	if statusCode != 200 {
		return "", c.formatS3Error(body, statusCode, "get bucket policy")
	}

	return string(body), nil
}

func (c *S3Client) PutBucketPolicy(ctx context.Context, accessKey, secretKey, bucket, policyJSON string) error {
	headers := map[string]string{"Content-Type": "application/json"}
	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "policy", []byte(policyJSON), headers)
	if err != nil {
		return fmt.Errorf("put bucket policy: %w", err)
	}

	if statusCode != 200 && statusCode != 204 {
		return c.formatS3Error(respBody, statusCode, "put bucket policy")
	}

	return nil
}

func (c *S3Client) DeleteBucketPolicy(ctx context.Context, accessKey, secretKey, bucket string) error {
	body, statusCode, err := c.doRequest(ctx, "DELETE", accessKey, secretKey, bucket, "policy", nil, nil)
	if err != nil {
		return fmt.Errorf("delete bucket policy: %w", err)
	}

	if statusCode != 204 && statusCode != 404 {
		return c.formatS3Error(body, statusCode, "delete bucket policy")
	}

	return nil
}
