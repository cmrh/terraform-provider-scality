---
page_title: "scality_user Data Source - scality"
subcategory: "IAM"
description: |-
  Looks up an existing IAM user by name within an account.
---

# scality_user (Data Source)

Looks up an existing IAM user by name within an account. Use this to attach Terraform-managed sub-resources (`scality_user_policy`, `scality_user_access_key`, `scality_group_membership`) to a user created outside the current Terraform configuration.

## Example

```hcl
data "scality_user" "existing" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  username           = "platform-bot"
}

resource "scality_user_policy" "read_only" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  username           = data.scality_user.existing.username
  policy_name        = "read-only-buckets"
  policy_document    = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:ListAllMyBuckets", "s3:GetObject"]
        Resource = "*"
      }
    ]
  })
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account that owns the user.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account that owns the user.
- `username` (Required, String) — Name of the user to look up.

## Attribute Reference

- `id` (String) — Same as `user_id`.
- `user_id` (String) — Stable unique identifier for the user.
- `arn` (String) — ARN of the user.
- `path` (String) — IAM path of the user (usually `/`).

## Notes

- If no user with the given name exists, the data source errors at plan time.
- Looks up by **name**, not ID.
