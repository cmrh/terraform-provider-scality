package client

import (
	"context"
	"encoding/xml"
	"fmt"
)

type accessControlPolicy struct {
	XMLName           xml.Name `xml:"AccessControlPolicy"`
	Owner             aclOwner `xml:"Owner"`
	AccessControlList aclList  `xml:"AccessControlList"`
}

type aclOwner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type aclList struct {
	Grants []aclGrant `xml:"Grant"`
}

type aclGrant struct {
	Grantee    aclGrantee `xml:"Grantee"`
	Permission string     `xml:"Permission"`
}

type aclGrantee struct {
	Type        string `xml:"type,attr"`
	ID          string `xml:"ID,omitempty"`
	DisplayName string `xml:"DisplayName,omitempty"`
	URI         string `xml:"URI,omitempty"`
}

func (c *S3Client) GetBucketACL(ctx context.Context, accessKey, secretKey, bucket string) (*accessControlPolicy, error) {
	body, statusCode, err := c.doRequest(ctx, "GET", accessKey, secretKey, bucket, "acl", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get bucket acl: %w", err)
	}

	if statusCode != 200 {
		return nil, c.formatS3Error(body, statusCode, "get bucket acl")
	}

	var acp accessControlPolicy
	if err := xml.Unmarshal(body, &acp); err != nil {
		return nil, fmt.Errorf("parsing acl response: %w", err)
	}

	return &acp, nil
}

func (c *S3Client) PutBucketACL(ctx context.Context, accessKey, secretKey, bucket, cannedACL string) error {
	headers := map[string]string{"x-amz-acl": cannedACL}
	respBody, statusCode, err := c.doRequest(ctx, "PUT", accessKey, secretKey, bucket, "acl", nil, headers)
	if err != nil {
		return fmt.Errorf("put bucket acl: %w", err)
	}

	if statusCode != 200 {
		return c.formatS3Error(respBody, statusCode, "put bucket acl")
	}

	return nil
}
