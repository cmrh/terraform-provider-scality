---
page_title: "scality_iam_role Resource - scality"
subcategory: "IAM"
description: |-
  Manages an IAM role within a Scality account. Roles allow services (such as `backbeat` for replication) to perform actions on behalf of the account.
---

# scality_iam_role

Manages an IAM role within a Scality account. Roles allow services (such as `backbeat` for replication) to perform actions on behalf of the account.

## Examples

The role's `assume_role_policy` (trust policy) decides who may assume it. Any of
the principal forms below is valid — a role is not limited to service principals.

### Service principal

A platform service (e.g. `backbeat` for replication) assumes the role:

```hcl
resource "scality_iam_role" "replication" {
  account_access_key = local.ak
  account_secret_key = local.sk
  role_name          = "replication-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "backbeat" }
      Action    = "sts:AssumeRole"
    }]
  })
}
```

### Account principal (cross-account)

Another account assumes the role — the basis of the provider
[`assume_role`](../index.md#delegated-cross-account-management-assume_role)
block. Grant the other account's **root**:

```hcl
resource "scality_iam_role" "account_manager" {
  account_access_key = local.ak
  account_secret_key = local.sk
  role_name          = "account-manager"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = "arn:aws:iam::123456789012:root" }  # management account
      Action    = "sts:AssumeRole"
    }]
  })
}
```

### User principal

A specific IAM user (in this or another account) assumes the role:

```hcl
resource "scality_iam_role" "operator" {
  account_access_key = local.ak
  account_secret_key = local.sk
  role_name          = "operator-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = "arn:aws:iam::123456789012:user/deploy-bot" }
      Action    = "sts:AssumeRole"
    }]
  })
}
```

## Trust Policy Principals

The `assume_role_policy` (trust policy) declares who may assume the role. Vault
accepts these principal forms:

| Principal | Example | Use |
|-----------|---------|-----|
| Service | `{ "Service": "backbeat" }` | Platform services, e.g. replication. |
| Account root | `{ "AWS": "arn:aws:iam::111111111111:root" }` | Let another account (e.g. a management account) assume the role. |
| IAM user | `{ "AWS": "arn:aws:iam::111111111111:user/name" }` | Let a specific user assume the role. |
| Account ID | `{ "AWS": "111111111111" }` | Shorthand for the account root. |
| Wildcard | `{ "AWS": "*" }` | Any principal (use with care). |

The "Account principal" example above is what a **management account** needs to
manage this account via the provider
[`assume_role`](../index.md#delegated-cross-account-management-assume_role) block.

Always use the account **root** ARN (`arn:aws:iam::<id>:root`), a user ARN, or the
bare account ID as the cross-account principal. The path-style ARN exposed as
`scality_account.arn` (`arn:aws:iam::<id>:/name/`) is **not** a valid principal —
Vault rejects it with `MalformedPolicyDocument`.

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | No | Access key of the owning account. Sensitive. Changing this replaces the resource. Omit to use the provider's assume_role credentials. |
| `account_secret_key` | String | No | Secret key of the owning account. Sensitive. Changing this replaces the resource. Omit to use the provider's assume_role credentials. |
| `role_name` | String | Yes | Name of the IAM role. Changing this replaces the resource. |
| `assume_role_policy` | String | Yes | JSON trust policy document that grants entities permission to assume the role. Changing this replaces the resource. |

## Attribute Reference

| Name | Type | Description |
|------|------|-------------|
| `arn` | String | ARN of the created IAM role. |

## Notes

- All attributes force resource replacement. Scality's `UpdateRole` API only supports `MaxSessionDuration`, not trust policy changes, so any change requires a destroy-and-recreate.
- Attached policies must be detached before the role can be deleted. Terraform handles this automatically when `scality_iam_role_policy_attachment` resources reference this role.

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the role name and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_iam_role.example ROLE_NAME
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_iam_role.example "ACCESS_KEY:SECRET_KEY:ROLE_NAME"
```
