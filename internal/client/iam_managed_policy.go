package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

type iamManagedPolicy struct {
	PolicyName       string `xml:"PolicyName"`
	PolicyId         string `xml:"PolicyId"`
	Arn              string `xml:"Arn"`
	Path             string `xml:"Path"`
	DefaultVersionId string `xml:"DefaultVersionId"`
	AttachmentCount  int    `xml:"AttachmentCount"`
}

type createManagedPolicyResponse struct {
	XMLName xml.Name `xml:"CreatePolicyResponse"`
	Result  struct {
		Policy iamManagedPolicy `xml:"Policy"`
	} `xml:"CreatePolicyResult"`
}

type getManagedPolicyResponse struct {
	XMLName xml.Name `xml:"GetPolicyResponse"`
	Result  struct {
		Policy iamManagedPolicy `xml:"Policy"`
	} `xml:"GetPolicyResult"`
}

type getPolicyVersionResponse struct {
	XMLName xml.Name `xml:"GetPolicyVersionResponse"`
	Result  struct {
		PolicyVersion struct {
			Document         string `xml:"Document"`
			VersionId        string `xml:"VersionId"`
			IsDefaultVersion bool   `xml:"IsDefaultVersion"`
		} `xml:"PolicyVersion"`
	} `xml:"GetPolicyVersionResult"`
}

type createPolicyVersionResponse struct {
	XMLName xml.Name `xml:"CreatePolicyVersionResponse"`
	Result  struct {
		PolicyVersion struct {
			Document         string `xml:"Document"`
			VersionId        string `xml:"VersionId"`
			IsDefaultVersion bool   `xml:"IsDefaultVersion"`
		} `xml:"PolicyVersion"`
	} `xml:"CreatePolicyVersionResult"`
}

func (c *IAMClient) CreateManagedPolicy(ctx context.Context, accessKey, secretKey, policyName, policyDocument string) (*iamManagedPolicy, error) {
	params := url.Values{
		"Action":         {"CreatePolicy"},
		"PolicyName":     {policyName},
		"PolicyDocument": {policyDocument},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create policy: %w", err)
	}

	var resp createManagedPolicyResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create policy response: %w", err)
	}

	return &resp.Result.Policy, nil
}

func (c *IAMClient) GetManagedPolicy(ctx context.Context, accessKey, secretKey, policyArn string) (*iamManagedPolicy, error) {
	params := url.Values{
		"Action":    {"GetPolicy"},
		"PolicyArn": {policyArn},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil, nil
		}
		return nil, fmt.Errorf("get policy: %w", err)
	}

	var resp getManagedPolicyResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing get policy response: %w", err)
	}

	return &resp.Result.Policy, nil
}

func (c *IAMClient) GetManagedPolicyVersion(ctx context.Context, accessKey, secretKey, policyArn, versionId string) (string, error) {
	params := url.Values{
		"Action":    {"GetPolicyVersion"},
		"PolicyArn": {policyArn},
		"VersionId": {versionId},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return "", fmt.Errorf("get policy version: %w", err)
	}

	var resp getPolicyVersionResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing get policy version response: %w", err)
	}

	decoded, err := url.QueryUnescape(resp.Result.PolicyVersion.Document)
	if err != nil {
		return resp.Result.PolicyVersion.Document, nil
	}

	return decoded, nil
}

func (c *IAMClient) CreateManagedPolicyVersion(ctx context.Context, accessKey, secretKey, policyArn, policyDocument string) error {
	params := url.Values{
		"Action":         {"CreatePolicyVersion"},
		"PolicyArn":      {policyArn},
		"PolicyDocument": {policyDocument},
		"SetAsDefault":   {"true"},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("create policy version: %w", err)
	}

	return nil
}

type listPolicyVersionsResponse struct {
	XMLName xml.Name `xml:"ListPolicyVersionsResponse"`
	Result  struct {
		Versions []struct {
			VersionId        string `xml:"VersionId"`
			IsDefaultVersion bool   `xml:"IsDefaultVersion"`
		} `xml:"Versions>member"`
	} `xml:"ListPolicyVersionsResult"`
}

func (c *IAMClient) DeleteManagedPolicy(ctx context.Context, accessKey, secretKey, policyArn string) error {
	versions, err := c.doSignedRequest(ctx, accessKey, secretKey, url.Values{
		"Action":    {"ListPolicyVersions"},
		"PolicyArn": {policyArn},
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("list policy versions: %w", err)
	}

	var vResp listPolicyVersionsResponse
	if err := xml.Unmarshal(versions, &vResp); err == nil {
		for _, v := range vResp.Result.Versions {
			if v.IsDefaultVersion {
				continue
			}
			_, _ = c.doSignedRequest(ctx, accessKey, secretKey, url.Values{
				"Action":    {"DeletePolicyVersion"},
				"PolicyArn": {policyArn},
				"VersionId": {v.VersionId},
			})
		}
	}

	_, err = c.doSignedRequest(ctx, accessKey, secretKey, url.Values{
		"Action":    {"DeletePolicy"},
		"PolicyArn": {policyArn},
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			return nil
		}
		return fmt.Errorf("delete policy: %w", err)
	}

	return nil
}
