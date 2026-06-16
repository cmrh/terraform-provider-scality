---
page_title: "scality_user Resource - scality"
subcategory: "IAM"
description: |-
  Creates an IAM user within a Scality account. Users can be assigned policies and access keys for day-to-day S3 operations.
---

# scality_user

Creates an IAM user within a Scality account. Users can be assigned policies and access keys for day-to-day S3 operations.

## Example

```hcl
resource "scality_user" "operator" {
  account_access_key = local.ak
  account_secret_key = local.sk
  username           = "bucket-operator"
}

output "user_arn" {
  value = scality_user.operator.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `username` | String | Yes | IAM username. Forces replacement. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `user_id` | Unique user identifier. |
| `arn` | ARN of the user (useful in bucket policies). |
| `path` | IAM path of the user. |

## Notes

- All arguments force replacement on change -- users cannot be renamed.
- Delete the user's access keys and policies before deleting the user.

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the username and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_user.example USERNAME
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_user.example "ACCESS_KEY:SECRET_KEY:USERNAME"
```
