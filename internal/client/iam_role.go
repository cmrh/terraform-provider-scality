package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
)

type iamRole struct {
	RoleName                 string `xml:"RoleName"`
	RoleId                   string `xml:"RoleId"`
	Arn                      string `xml:"Arn"`
	Path                     string `xml:"Path"`
	AssumeRolePolicyDocument string `xml:"AssumeRolePolicyDocument"`
}

type iamAttachedPolicy struct {
	PolicyName string `xml:"PolicyName"`
	PolicyArn  string `xml:"PolicyArn"`
}

type createRoleResponse struct {
	XMLName xml.Name `xml:"CreateRoleResponse"`
	Result  struct {
		Role iamRole `xml:"Role"`
	} `xml:"CreateRoleResult"`
}

type getRoleResponse struct {
	XMLName xml.Name `xml:"GetRoleResponse"`
	Result  struct {
		Role iamRole `xml:"Role"`
	} `xml:"GetRoleResult"`
}

type listAttachedRolePoliciesResponse struct {
	XMLName xml.Name `xml:"ListAttachedRolePoliciesResponse"`
	Result  struct {
		AttachedPolicies []iamAttachedPolicy `xml:"AttachedPolicies>member"`
	} `xml:"ListAttachedRolePoliciesResult"`
}

func (c *IAMClient) CreateRole(ctx context.Context, accessKey, secretKey, roleName, assumeRolePolicyDocument string) (*iamRole, error) {
	params := url.Values{
		"Action":                   {"CreateRole"},
		"RoleName":                 {roleName},
		"AssumeRolePolicyDocument": {assumeRolePolicyDocument},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	var resp createRoleResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing create role response: %w", err)
	}

	return &resp.Result.Role, nil
}

func (c *IAMClient) GetRole(ctx context.Context, accessKey, secretKey, roleName string) (*iamRole, error) {
	params := url.Values{
		"Action":   {"GetRole"},
		"RoleName": {roleName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get role: %w", err)
	}

	var resp getRoleResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing get role response: %w", err)
	}

	return &resp.Result.Role, nil
}

func (c *IAMClient) DeleteRole(ctx context.Context, accessKey, secretKey, roleName string) error {
	params := url.Values{
		"Action":   {"DeleteRole"},
		"RoleName": {roleName},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("delete role: %w", err)
	}

	return nil
}

func (c *IAMClient) AttachRolePolicy(ctx context.Context, accessKey, secretKey, roleName, policyArn string) error {
	params := url.Values{
		"Action":    {"AttachRolePolicy"},
		"RoleName":  {roleName},
		"PolicyArn": {policyArn},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return fmt.Errorf("attach role policy: %w", err)
	}

	return nil
}

func (c *IAMClient) DetachRolePolicy(ctx context.Context, accessKey, secretKey, roleName, policyArn string) error {
	params := url.Values{
		"Action":    {"DetachRolePolicy"},
		"RoleName":  {roleName},
		"PolicyArn": {policyArn},
	}

	_, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("detach role policy: %w", err)
	}

	return nil
}

func (c *IAMClient) ListAttachedRolePolicies(ctx context.Context, accessKey, secretKey, roleName string) ([]iamAttachedPolicy, error) {
	params := url.Values{
		"Action":   {"ListAttachedRolePolicies"},
		"RoleName": {roleName},
	}

	body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
	if err != nil {
		return nil, fmt.Errorf("list attached role policies: %w", err)
	}

	var resp listAttachedRolePoliciesResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing list attached role policies response: %w", err)
	}

	return resp.Result.AttachedPolicies, nil
}

// RoleListEntry represents one role in a ListRoles response.
type RoleListEntry struct {
	RoleName   string `xml:"RoleName"`
	RoleId     string `xml:"RoleId"`
	Arn        string `xml:"Arn"`
	Path       string `xml:"Path"`
	CreateDate string `xml:"CreateDate"`
}

type listRolesResponse struct {
	XMLName xml.Name `xml:"ListRolesResponse"`
	Result  struct {
		Roles       []RoleListEntry `xml:"Roles>member"`
		IsTruncated bool            `xml:"IsTruncated"`
		Marker      string          `xml:"Marker"`
	} `xml:"ListRolesResult"`
}

// ListRoles retrieves all IAM roles in the calling account, walking pagination.
func (c *IAMClient) ListRoles(ctx context.Context, accessKey, secretKey string) ([]RoleListEntry, error) {
	var all []RoleListEntry
	marker := ""
	for {
		params := url.Values{
			"Action": {"ListRoles"},
		}
		if marker != "" {
			params.Set("Marker", marker)
		}

		body, err := c.doSignedRequest(ctx, accessKey, secretKey, params)
		if err != nil {
			return nil, fmt.Errorf("list roles: %w", err)
		}

		var page listRolesResponse
		if err := xml.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parsing list roles response: %w", err)
		}

		all = append(all, page.Result.Roles...)
		if !page.Result.IsTruncated || page.Result.Marker == "" {
			break
		}
		marker = page.Result.Marker
	}
	return all, nil
}
