# Scality Terraform/OpenTofu Provider

Terraform/OpenTofu provider for managing Scality S3C / RING storage. Supports account management, IAM (users, groups, policies), and S3 bucket configuration.

The provider authenticates via two methods:
- **IAM API** -- AWS Signature V4 for account and IAM operations
- **Console API** -- JWT for console account management

## Provider Configuration

```hcl
provider "scality" {
  # IAM API (for accounts, users, groups, buckets)
  endpoint   = "http://scality.example.com:8080"  # or SCALITY_ENDPOINT
  access_key = var.admin_ak                        # or SCALITY_ACCESS_KEY
  secret_key = var.admin_sk                        # or SCALITY_SECRET_KEY

  # Console API (for console accounts)
  console_endpoint = "http://scality.example.com:8080"  # or SCALITY_CONSOLE_ENDPOINT
  console_username = var.console_user                    # or SCALITY_CONSOLE_USERNAME
  console_password = var.console_pass                    # or SCALITY_CONSOLE_PASSWORD

  # Optional
  insecure_skip_verify = true  # Skip TLS verification (self-signed certs)
}
```

You only need to configure the APIs you use. An IAM endpoint alone is sufficient for per-account resources (buckets, users, groups).

## Resources

### Accounts

| Resource | Description |
|----------|-------------|
| [scality_account](scality_account.md) | Account via IAM API (SigV4 auth) |
| [scality_console_account](scality_console_account.md) | Account via Console API (JWT auth) |
| [scality_account_access_key](scality_account_access_key.md) | Additional root access key for an account |

### S3 Buckets

| Resource | Description |
|----------|-------------|
| [scality_bucket](scality_bucket.md) | S3 bucket with versioning and tags |
| [scality_bucket_policy](scality_bucket_policy.md) | JSON bucket policy |
| [scality_bucket_encryption](scality_bucket_encryption.md) | Server-side encryption (SSE-S3 / SSE-KMS) |
| [scality_bucket_acl](scality_bucket_acl.md) | Canned ACL |
| [scality_bucket_lifecycle](scality_bucket_lifecycle.md) | Object lifecycle rules |
| [scality_bucket_object_lock](scality_bucket_object_lock.md) | Object lock retention |
| [scality_bucket_replication](scality_bucket_replication.md) | Cross-bucket replication |

### IAM

| Resource | Description |
|----------|-------------|
| [scality_user](scality_user.md) | IAM user within an account |
| [scality_user_access_key](scality_user_access_key.md) | Access key for a user |
| [scality_user_policy](scality_user_policy.md) | Inline policy attached to a user |
| [scality_group](scality_group.md) | IAM group |
| [scality_group_membership](scality_group_membership.md) | Group membership (users in a group) |

## Credential Pattern

Most resources use per-account credentials (not provider-level admin credentials). A typical setup:

```hcl
# Create account via Console API
resource "scality_console_account" "app" {
  account_name             = "my-app"
  email                    = "app@example.com"
  generate_random_password = true
}

# Generate a stable key pair that Terraform owns
resource "scality_account_access_key" "stable" {
  account_access_key = scality_console_account.app.access_key
  account_secret_key = scality_console_account.app.secret_key
}

locals {
  ak = scality_account_access_key.stable.access_key
  sk = scality_account_access_key.stable.secret_key
}

# Use those credentials for all per-account resources
resource "scality_bucket" "data" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = "app-data"
}
```

The initial credentials from account creation may be rotated externally. The second key pair created via `scality_account_access_key` gives Terraform a stable credential that external processes will not touch.
