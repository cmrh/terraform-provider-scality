package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// maxManagedPolicyVersions is the IAM cap on versions per managed policy.
const maxManagedPolicyVersions = 5

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
		if IsNotFound(err) {
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
	// Managed policies cap at maxManagedPolicyVersions; prune the oldest
	// non-default version when full so updates don't fail with LimitExceeded.
	versions, err := c.listManagedPolicyVersions(ctx, accessKey, secretKey, policyArn)
	if err != nil {
		return err
	}
	if len(versions) >= maxManagedPolicyVersions {
		oldest := oldestNonDefaultVersion(versions)
		if oldest == "" {
			return fmt.Errorf("cannot create policy version: %d versions exist and none are prunable", len(versions))
		}
		if err := c.deleteManagedPolicyVersion(ctx, accessKey, secretKey, policyArn, oldest); err != nil {
			return err
		}
	}

	params := url.Values{
		"Action":         {"CreatePolicyVersion"},
		"PolicyArn":      {policyArn},
		"PolicyDocument": {policyDocument},
		"SetAsDefault":   {"true"},
	}

	if _, err := c.doSignedRequest(ctx, accessKey, secretKey, params); err != nil {
		return fmt.Errorf("create policy version: %w", err)
	}

	return nil
}

type policyVersion struct {
	VersionId        string
	IsDefaultVersion bool
}

func (c *IAMClient) listManagedPolicyVersions(ctx context.Context, accessKey, secretKey, policyArn string) ([]policyVersion, error) {
	body, err := c.doSignedRequest(ctx, accessKey, secretKey, url.Values{
		"Action":    {"ListPolicyVersions"},
		"PolicyArn": {policyArn},
	})
	if err != nil {
		return nil, fmt.Errorf("list policy versions: %w", err)
	}

	var resp listPolicyVersionsResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing list policy versions response: %w", err)
	}

	out := make([]policyVersion, 0, len(resp.Result.Versions))
	for _, v := range resp.Result.Versions {
		out = append(out, policyVersion{VersionId: v.VersionId, IsDefaultVersion: v.IsDefaultVersion})
	}
	return out, nil
}

func (c *IAMClient) deleteManagedPolicyVersion(ctx context.Context, accessKey, secretKey, policyArn, versionId string) error {
	if _, err := c.doSignedRequest(ctx, accessKey, secretKey, url.Values{
		"Action":    {"DeletePolicyVersion"},
		"PolicyArn": {policyArn},
		"VersionId": {versionId},
	}); err != nil {
		return fmt.Errorf("delete policy version %s: %w", versionId, err)
	}
	return nil
}

// oldestNonDefaultVersion returns the non-default version with the lowest
// numeric id (vN); empty when every version is the default.
func oldestNonDefaultVersion(versions []policyVersion) string {
	oldest := ""
	oldestNum := -1
	for _, v := range versions {
		if v.IsDefaultVersion {
			continue
		}
		n := versionNumber(v.VersionId)
		if oldest == "" || (n >= 0 && (oldestNum < 0 || n < oldestNum)) {
			oldest, oldestNum = v.VersionId, n
		}
	}
	return oldest
}

func versionNumber(versionId string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(versionId, "v"))
	if err != nil {
		return -1
	}
	return n
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
	// Non-default versions must be deleted before the policy itself.
	versions, err := c.listManagedPolicyVersions(ctx, accessKey, secretKey, policyArn)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	for _, v := range versions {
		if v.IsDefaultVersion {
			continue
		}
		_ = c.deleteManagedPolicyVersion(ctx, accessKey, secretKey, policyArn, v.VersionId)
	}

	if _, err := c.doSignedRequest(ctx, accessKey, secretKey, url.Values{
		"Action":    {"DeletePolicy"},
		"PolicyArn": {policyArn},
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("delete policy: %w", err)
	}

	return nil
}

// PolicyListEntry represents one managed policy in a ListPolicies response.
type PolicyListEntry struct {
	PolicyName       string `xml:"PolicyName"`
	PolicyId         string `xml:"PolicyId"`
	Arn              string `xml:"Arn"`
	Path             string `xml:"Path"`
	DefaultVersionId string `xml:"DefaultVersionId"`
	AttachmentCount  int    `xml:"AttachmentCount"`
	CreateDate       string `xml:"CreateDate"`
	UpdateDate       string `xml:"UpdateDate"`
}

type listPoliciesResponse struct {
	XMLName xml.Name `xml:"ListPoliciesResponse"`
	Result  struct {
		Policies    []PolicyListEntry `xml:"Policies>member"`
		IsTruncated bool              `xml:"IsTruncated"`
		Marker      string            `xml:"Marker"`
	} `xml:"ListPoliciesResult"`
}

// ListPolicies retrieves all customer-managed (Scope=Local) IAM policies in
// the calling account, walking pagination. Scality ships no AWS-managed
// policies, but Scope=Local keeps the contract explicit.
func (c *IAMClient) ListPolicies(ctx context.Context, accessKey, secretKey string) ([]PolicyListEntry, error) {
	var all []PolicyListEntry
	marker := ""
	for {
		params := url.Values{
			"Action": {"ListPolicies"},
			"Scope":  {"Local"},
		}
		if marker != "" {
			params.Set("Marker", marker)
		}

		body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
		if err != nil {
			return nil, fmt.Errorf("list policies: %w", err)
		}

		var page listPoliciesResponse
		if err := xml.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parsing list policies response: %w", err)
		}

		all = append(all, page.Result.Policies...)
		if !page.Result.IsTruncated || page.Result.Marker == "" {
			break
		}
		marker = page.Result.Marker
	}
	return all, nil
}
