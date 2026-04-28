# OpenTofu/Terraform Provider for Scality

Manage Scality S3C / RING storage infrastructure with OpenTofu or Terraform. Supports account management, IAM users/groups/policies, and S3 bucket configuration.

## Resources

### Accounts

| Resource | Description |
|----------|-------------|
| [scality_account](docs/scality_account.md) | Account via IAM API (SigV4 auth) |
| [scality_console_account](docs/scality_console_account.md) | Account via Console API (JWT auth) |
| [scality_account_access_key](docs/scality_account_access_key.md) | Additional root access key for an account |

### S3 Buckets

| Resource | Description |
|----------|-------------|
| [scality_bucket](docs/scality_bucket.md) | Bucket with versioning and tags |
| [scality_bucket_policy](docs/scality_bucket_policy.md) | JSON bucket policy |
| [scality_bucket_encryption](docs/scality_bucket_encryption.md) | Server-side encryption (SSE-S3 / SSE-KMS) |
| [scality_bucket_lifecycle](docs/scality_bucket_lifecycle.md) | Object lifecycle rules |
| [scality_bucket_object_lock](docs/scality_bucket_object_lock.md) | Object lock retention |
| [scality_bucket_replication](docs/scality_bucket_replication.md) | Cross-bucket replication |

### IAM

| Resource | Description |
|----------|-------------|
| [scality_user](docs/scality_user.md) | IAM user within an account |
| [scality_user_access_key](docs/scality_user_access_key.md) | Access key for a user |
| [scality_user_policy](docs/scality_user_policy.md) | Inline policy attached to a user |
| [scality_group](docs/scality_group.md) | IAM group |
| [scality_group_membership](docs/scality_group_membership.md) | Group membership |

## Quick Start

Set credentials via environment variables:

```bash
# For IAM and S3 resources
export SCALITY_ENDPOINT="http://scality.example.com:8080"
export SCALITY_ACCESS_KEY="admin-access-key"
export SCALITY_SECRET_KEY="admin-secret-key"

# For console account resources
export SCALITY_CONSOLE_ENDPOINT="http://scality.example.com:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="password"

# Optional
export SCALITY_INSECURE_SKIP_VERIFY="true"
```

```hcl
terraform {
  required_providers {
    scality = { source = "scality/scality" }
  }
}

provider "scality" {}

# Create an account with a stable key pair for Terraform
resource "scality_console_account" "app" {
  account_name             = "my-app"
  email                    = "app@example.com"
  generate_random_password = true
}

resource "scality_account_access_key" "stable" {
  account_access_key = scality_console_account.app.access_key
  account_secret_key = scality_console_account.app.secret_key
}

locals {
  ak = scality_account_access_key.stable.access_key
  sk = scality_account_access_key.stable.secret_key
}

# Create a bucket and a user
resource "scality_bucket" "data" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = "app-data"
  versioning         = true
}

resource "scality_user" "operator" {
  account_access_key = local.ak
  account_secret_key = local.sk
  username           = "app-operator"
}

resource "scality_user_access_key" "operator" {
  account_access_key = local.ak
  account_secret_key = local.sk
  username           = scality_user.operator.username
}
```

## Building from Source

```bash
git clone https://github.com/scality/terraform-provider-scality
cd terraform-provider-scality
go build -o terraform-provider-scality .
```

For development, add a `dev_overrides` block to `~/.tofurc` (or `~/.terraformrc`) so you can skip `tofu init`:

```hcl
provider_installation {
  dev_overrides {
    "scality/scality" = "/path/to/binary/directory"
  }
  direct {}
}
```

## Documentation

See [docs/README.md](docs/README.md) for the full resource reference.

## License

Apache 2.0
