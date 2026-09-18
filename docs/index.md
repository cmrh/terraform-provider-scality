---
page_title: "Scality Provider"
subcategory: ""
description: |-
  Terraform/OpenTofu provider for managing Scality S3C / RING storage.
---

# Scality Provider

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
  region               = "us-east-1"  # SigV4 signing region, or SCALITY_REGION (default: us-east-1)
  insecure_skip_verify = true         # Skip TLS verification (self-signed certs)

  # Optional: assume a role in another account (see "Delegated Cross-Account Management")
  assume_role {
    role_arn     = "arn:aws:iam::123456789012:role/account-manager"
    session_name = "terraform"  # optional, defaults to "terraform"
  }
}
```

You only need to configure the APIs you use. An IAM endpoint alone is sufficient for per-account resources (buckets, users, groups). The same endpoint serves the STS, IAM, S3, and superadmin APIs; the load balancer routes each request to the right backend.

## Credential Tiers

Resources fall into two credential tiers. Match the credential to the resource:

**Platform (superadmin) tier** — manages accounts themselves, using the provider's `access_key`/`secret_key` (or Console credentials).

| Resource | Credential |
|----------|------------|
| `scality_account` | Platform admin (IAM superadmin) |
| `scality_console_account` | Console admin |

**Account tier** — manages resources *inside* an account. Each resource takes its own `account_access_key`/`account_secret_key`, or falls back to the provider's `assume_role` credentials (see below). These never use the platform superadmin.

| Resource | Credential |
|----------|------------|
| `scality_account_access_key` | Account |
| `scality_bucket`, `scality_bucket_policy`, `scality_bucket_encryption`, `scality_bucket_lifecycle`, `scality_bucket_object_lock`, `scality_bucket_replication` | Account |
| `scality_user`, `scality_user_access_key`, `scality_user_policy` | Account |
| `scality_group`, `scality_group_membership` | Account |
| `scality_iam_policy`, `scality_iam_role`, `scality_iam_role_policy_attachment` | Account |

Data sources follow the same tier as their matching resource.

`assume_role` serves the **account tier** only. STS is an account-level operation, so the platform-tier resources above cannot use it and are unaffected by it.

## Delegated Cross-Account Management (`assume_role`)

A management account can manage resources in other accounts without holding each
account's long-lived keys. Configure the management account's own credentials on
the provider, add an `assume_role` block naming a role in the target account, and
the provider exchanges them once (at configuration) for temporary role
credentials via STS. Per-account resources then fall back to those temporary
credentials whenever they omit their own `account_access_key` /
`account_secret_key`.

```hcl
# Base identity: an IAM user in the management account (not the account root,
# not the platform superadmin) that has sts:AssumeRole permission.
provider "scality" {
  endpoint   = "http://scality.example.com:8080"
  access_key = var.mgmt_user_ak
  secret_key = var.mgmt_user_sk
}

# One aliased provider per customer account, each assuming that account's role.
provider "scality" {
  alias      = "customer_a"
  endpoint   = "http://scality.example.com:8080"
  access_key = var.mgmt_user_ak
  secret_key = var.mgmt_user_sk

  assume_role {
    role_arn = "arn:aws:iam::111111111111:role/account-manager"
  }
}

# No account_access_key/account_secret_key needed — the assumed role is used.
resource "scality_bucket" "data" {
  provider = scality.customer_a
  bucket   = "customer-a-data"
}
```

Notes:

- The base credentials must be an **IAM user** in the management account with
  `sts:AssumeRole` permission — not the account root, and not the platform
  superadmin. STS assume-role is an account-level operation: the account root
  cannot assume a role (`AccessDenied: Roles may not be assumed by root
  accounts`), and the superadmin account-management APIs do not support STS at
  all.
- The target role's trust policy must allow the management account as a
  principal. See [`scality_iam_role`](resources/scality_iam_role.md) for the
  principal forms Vault accepts.
- Credentials are assumed **once** at provider configuration and held in memory
  (never written to state). A very long single apply could outlive them;
  re-running picks up fresh credentials. Automatic refresh is not yet supported.
- Explicit `account_access_key` / `account_secret_key` on a resource always take
  precedence, so mixing assumed and explicit credentials in one configuration
  works.

## Resources

### Accounts

| Resource | Description |
|----------|-------------|
| [scality_account](resources/scality_account.md) | Account via IAM API (SigV4 auth) |
| [scality_console_account](resources/scality_console_account.md) | Account via Console API (JWT auth) |
| [scality_account_access_key](resources/scality_account_access_key.md) | Additional root access key for an account |

### S3 Buckets

| Resource | Description |
|----------|-------------|
| [scality_bucket](resources/scality_bucket.md) | S3 bucket with versioning and tags |
| [scality_bucket_policy](resources/scality_bucket_policy.md) | JSON bucket policy |
| [scality_bucket_encryption](resources/scality_bucket_encryption.md) | Server-side encryption (SSE-S3 / SSE-KMS) |
| [scality_bucket_lifecycle](resources/scality_bucket_lifecycle.md) | Object lifecycle rules |
| [scality_bucket_object_lock](resources/scality_bucket_object_lock.md) | Object lock retention |
| [scality_bucket_replication](resources/scality_bucket_replication.md) | Cross-region replication |

### IAM

| Resource | Description |
|----------|-------------|
| [scality_user](resources/scality_user.md) | IAM user within an account |
| [scality_user_access_key](resources/scality_user_access_key.md) | Access key for a user |
| [scality_user_policy](resources/scality_user_policy.md) | Inline policy attached to a user |
| [scality_group](resources/scality_group.md) | IAM group |
| [scality_group_membership](resources/scality_group_membership.md) | Group membership (users in a group) |
| [scality_iam_policy](resources/scality_iam_policy.md) | IAM managed policy (attachable to roles) |
| [scality_iam_role](resources/scality_iam_role.md) | IAM role with trust policy |
| [scality_iam_role_policy_attachment](resources/scality_iam_role_policy_attachment.md) | Attach a managed policy to a role |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| [scality_account](data-sources/scality_account.md) | Look up an existing account by name |
| [scality_accounts](data-sources/scality_accounts.md) | List all accounts in the cluster |
| [scality_bucket](data-sources/scality_bucket.md) | Look up an existing bucket within an account |
| [scality_buckets](data-sources/scality_buckets.md) | List all buckets owned by an account |
| [scality_user](data-sources/scality_user.md) | Look up an existing IAM user by name |
| [scality_users](data-sources/scality_users.md) | List all IAM users in an account |
| [scality_group](data-sources/scality_group.md) | Look up an existing IAM group by name |
| [scality_groups](data-sources/scality_groups.md) | List all IAM groups in an account |
| [scality_iam_policy](data-sources/scality_iam_policy.md) | Look up an existing managed policy by name |
| [scality_iam_policies](data-sources/scality_iam_policies.md) | List all customer-managed policies in an account |
| [scality_iam_role](data-sources/scality_iam_role.md) | Look up an existing IAM role by name |
| [scality_iam_roles](data-sources/scality_iam_roles.md) | List all IAM roles in an account |

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

## Credential Dependencies

When using IAM user credentials for resources like buckets, encryption, or replication, always declare a `depends_on` pointing at the user's access policy. This ensures Terraform destroys resources in the correct order -- removing buckets and other resources before the policy that grants permission to manage them.

```hcl
resource "scality_bucket" "data" {
  account_access_key = scality_user_access_key.operator.access_key_id
  account_secret_key = scality_user_access_key.operator.secret_access_key
  bucket             = "my-data"

  depends_on = [scality_user_policy.operator]
}
```

Without this, `terraform destroy` may remove the user's policy first, leaving Terraform unable to delete the remaining resources.
