# OpenTofu/Terraform Provider for Scality

A provider for managing Scality S3C / RING storage infrastructure. Covers account management, IAM (users, groups, policies), and S3 bucket configuration.

Compatible with both [OpenTofu](https://opentofu.org/) (recommended) and Terraform.

## Resources

### Accounts

| Resource | Auth | Description |
|----------|------|-------------|
| `scality_account` | Admin IAM (SigV4) | Account via IAM API. Supports custom attributes. |
| `scality_console_account` | Console (JWT) | Account via Console API. Optional password generation. |
| `scality_account_access_key` | Per-account IAM | Additional root access key for an account. |

### S3 Buckets

| Resource | Description |
|----------|-------------|
| `scality_bucket` | Bucket with inline versioning and tags. |
| `scality_bucket_policy` | JSON bucket policy. |
| `scality_bucket_encryption` | Server-side encryption (AES256 / aws:kms). |
| `scality_bucket_lifecycle` | Object expiration and cleanup rules. |
| `scality_bucket_object_lock` | WORM retention (governance / compliance). |
| `scality_bucket_replication` | Cross-bucket replication. |

### IAM

| Resource | Description |
|----------|-------------|
| `scality_user` | IAM user within an account. |
| `scality_user_access_key` | Access key pair for a user. |
| `scality_user_policy` | Inline policy attached to a user. |
| `scality_group` | IAM group. |
| `scality_group_membership` | Authoritative group membership (set-diff updates). |

## Architecture

```
terraform-provider-scality/
├── main.go
├── internal/
│   ├── client/
│   │   ├── iam.go                  # IAM client (SigV4 signing, service "iam")
│   │   ├── console.go              # Console client (JWT auth)
│   │   ├── s3.go                   # S3 client (SigV4 signing, service "s3")
│   │   ├── s3_bucket.go            # CreateBucket, HeadBucket, DeleteBucket
│   │   ├── s3_versioning.go        # Get/PutBucketVersioning
│   │   ├── s3_tagging.go           # Get/Put/DeleteBucketTagging
│   │   ├── s3_policy.go            # Get/Put/DeleteBucketPolicy (JSON)
│   │   ├── s3_encryption.go        # Get/Put/DeleteBucketEncryption
│   │   ├── s3_lifecycle.go         # Get/Put/DeleteBucketLifecycle
│   │   ├── s3_object_lock.go       # Get/PutObjectLockConfiguration
│   │   ├── s3_replication.go       # Get/Put/DeleteBucketReplication
│   │   └── provider_clients.go     # ProviderClients bundle (IAM + Console + S3)
│   ├── provider/
│   │   └── provider.go             # Provider config, env vars, resource registration
│   └── resources/
│       ├── account/                # model.go + resource.go
│       ├── console_account/
│       ├── account_access_key/
│       ├── bucket/
│       ├── bucket_policy/
│       ├── bucket_encryption/
│       ├── bucket_lifecycle/
│       ├── bucket_object_lock/
│       ├── bucket_replication/
│       ├── user/
│       ├── user_access_key/
│       ├── user_policy/
│       ├── group/
│       └── group_membership/
└── docs/                           # Per-resource documentation
```

## Authentication

The provider supports two API backends, configured independently:

**IAM API** (accounts, users, groups, buckets):
```bash
export SCALITY_ENDPOINT="http://scality.example.com:8080"
export SCALITY_ACCESS_KEY="admin-access-key"
export SCALITY_SECRET_KEY="admin-secret-key"
```

**Console API** (console accounts):
```bash
export SCALITY_CONSOLE_ENDPOINT="http://scality.example.com:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="password"
```

Most resources use **per-account credentials** (passed as resource attributes), not the provider-level admin credentials. The typical pattern: create an account, generate a stable key pair, then use those keys for all resources within that account.

## Building

```bash
go build -o terraform-provider-scality .
```

For development with OpenTofu, use `dev_overrides` in `~/.tofurc` to skip `tofu init`:

```hcl
provider_installation {
  dev_overrides {
    "scality/scality" = "/path/to/plugin/directory"
  }
  direct {}
}
```

## Documentation

See `docs/index.md` for the full resource reference with examples.
