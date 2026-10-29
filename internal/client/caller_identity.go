package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
)

type getCallerIdentityResponse struct {
	XMLName xml.Name `xml:"GetCallerIdentityResponse"`
	Result  struct {
		Account string `xml:"Account"`
	} `xml:"GetCallerIdentityResult"`
}

// CallerAccount returns the account ID the credentials act in, via STS
// GetCallerIdentity. For assumed-role credentials it is the role's account.
func (c *IAMClient) CallerAccount(ctx context.Context, accessKey, secretKey string) (string, error) {
	if c.callerAccounts != nil {
		if id, ok := c.callerAccounts.Load(accessKey); ok {
			return id.(string), nil
		}
	}

	body, err := c.doServiceRequest(ctx, stsService, stsAPIVersion, accessKey, secretKey, url.Values{"Action": {"GetCallerIdentity"}})
	if err != nil {
		return "", fmt.Errorf("get caller identity: %w", err)
	}

	var resp getCallerIdentityResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing get caller identity response: %w", err)
	}
	if resp.Result.Account == "" {
		return "", fmt.Errorf("get caller identity: response has no account")
	}

	if c.callerAccounts != nil {
		c.callerAccounts.Store(accessKey, resp.Result.Account)
	}
	return resp.Result.Account, nil
}
