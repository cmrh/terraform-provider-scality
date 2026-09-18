package client

import "fmt"

// ProviderClients bundles IAM, Console, and S3 clients for resource access.
type ProviderClients struct {
	IAM     *IAMClient
	Console *ConsoleClient
	S3      *S3Client

	// Assumed holds temporary credentials from a provider-level assume_role.
	// Per-account resources fall back to these when they omit their own
	// account_access_key/account_secret_key. Nil when no role was assumed.
	Assumed *AssumedAccountCreds
}

// AssumedAccountCreds are per-account credentials (with session token) that
// per-account resources use when they don't carry their own.
type AssumedAccountCreds struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
	iam          *IAMClient // clone of IAM carrying the session token
	s3           *S3Client  // clone of S3 carrying the session token
}

// SetAssumed records assumed-role credentials and builds token-bearing client
// clones. Called from the provider's Configure after a successful AssumeRole.
func (p *ProviderClients) SetAssumed(creds *AssumedCredentials) {
	if creds == nil {
		return
	}
	a := &AssumedAccountCreds{
		AccessKey:    creds.AccessKeyID,
		SecretKey:    creds.SecretAccessKey,
		SessionToken: creds.SessionToken,
	}
	if p.IAM != nil {
		clone := *p.IAM
		clone.AccessKey, clone.SecretKey, clone.SessionToken = creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken
		a.iam = &clone
	}
	if p.S3 != nil {
		clone := *p.S3
		clone.SessionToken = creds.SessionToken
		a.s3 = &clone
	}
	p.Assumed = a
}

var errNoAccountCreds = fmt.Errorf("no account credentials: set account_access_key and account_secret_key on the resource, or configure an assume_role block on the provider")

// ResolveIAM returns the IAM client and credentials for a per-account operation.
// Explicit resource credentials win; otherwise the provider's assumed-role
// credentials are used.
func (p *ProviderClients) ResolveIAM(accessKey, secretKey string) (*IAMClient, string, string, error) {
	if accessKey != "" && secretKey != "" {
		return p.IAM, accessKey, secretKey, nil
	}
	if p.Assumed != nil {
		return p.Assumed.iam, p.Assumed.AccessKey, p.Assumed.SecretKey, nil
	}
	return nil, "", "", errNoAccountCreds
}

// ResolveS3 mirrors ResolveIAM for the S3 client.
func (p *ProviderClients) ResolveS3(accessKey, secretKey string) (*S3Client, string, string, error) {
	if accessKey != "" && secretKey != "" {
		return p.S3, accessKey, secretKey, nil
	}
	if p.Assumed != nil {
		return p.Assumed.s3, p.Assumed.AccessKey, p.Assumed.SecretKey, nil
	}
	return nil, "", "", errNoAccountCreds
}
