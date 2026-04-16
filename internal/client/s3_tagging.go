package client

import (
	"context"
	"encoding/xml"
	"fmt"
)

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
