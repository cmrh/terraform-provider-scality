---
page_title: "scality_account Data Source - scality"
subcategory: "Accounts"
description: |-
  Looks up an existing Scality account by name via the IAM API. Useful for referencing accounts created outside the current Terraform configuration.
---

# scality_account (Data Source)

Looks up an existing Scality account by name. Use this when you need to reference an account that was created outside the current Terraform configuration — for example, to pass its canonical ID into a bucket policy, or its ID into a downstream module.

The data source does **not** expose `access_key` / `secret_key`. The IAM API returns those only at account creation. To get a key pair for an existing account, mint a new one via [`scality_account_access_key`](../resources/scality_account_access_key.md) using a key you already hold.

## Example

```hcl
data "scality_account" "existing" {
  name = "platform-tenant"
}

# Reference attributes in downstream resources:

resource "scality_bucket_policy" "shared" {
  account_access_key = var.operator_ak
  account_secret_key = var.operator_sk
  bucket             = "shared-data"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          CanonicalUser = data.scality_account.existing.canonical_id
        }
        Action   = ["s3:GetObject"]
        Resource = ["arn:aws:s3:::shared-data/*"]
      }
    ]
  })
}
```

## Argument Reference

- `name` (Required, String) — Name of the account to look up.

## Attribute Reference

- `id` (String) — Vault account ID.
- `email_address` (String) — Email address registered against the account.
- `quota_max` (Number) — Maximum bytes storable by the account (0 = unlimited).
- `custom_attributes` (Map of String) — Custom attributes (key-value string pairs) attached to the account.
- `arn` (String) — Amazon Resource Name of the account.
- `canonical_id` (String) — Canonical ID of the account, used in S3 bucket policies.
- `create_date` (String) — Account creation date.

## Notes

- If no account with the given name exists, the data source errors at plan time.
- Looks up by **name**, not ID. Account name is the natural key for tenant-management work.
- Uses the provider's `vault_admin_*` credentials (same as the `scality_account` resource).
