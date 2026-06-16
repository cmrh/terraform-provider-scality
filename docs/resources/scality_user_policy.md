---
page_title: "scality_user_policy Resource - scality"
subcategory: "IAM"
description: |-
  Attaches an inline IAM policy to a user. Policies define what actions the user is allowed to perform.
---

# scality_user_policy

Attaches an inline IAM policy to a user. Policies define what actions the user is allowed to perform.

## Example

```hcl
resource "scality_user_policy" "s3_access" {
  account_access_key = local.ak
  account_secret_key = local.sk
  username           = scality_user.operator.username
  policy_name        = "S3FullAccess"

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "s3:*"
      Resource = "*"
    }]
  })
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `username` | String | Yes | User to attach the policy to. Forces replacement. |
| `policy_name` | String | Yes | Name of the policy. Forces replacement. |
| `policy_document` | String | Yes | JSON policy document. Use `jsonencode()` for readability. |

## Notes

- Updating the `policy_document` applies the change in-place (PutUserPolicy is an upsert).
- Changing `username` or `policy_name` forces replacement.
- The policy document follows standard AWS IAM policy syntax.

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the identity portion and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_user_policy.example USERNAME:POLICY_NAME
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_user_policy.example "ACCESS_KEY:SECRET_KEY:USERNAME:POLICY_NAME"
```
