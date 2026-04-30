---
page_title: "scality_iam_role Resource - scality"
subcategory: "IAM"
description: |-
  Manages an IAM role within a Scality account. Roles allow services (such as `backbeat` for replication) to perform actions on behalf of the account.
---

# scality_iam_role

Manages an IAM role within a Scality account. Roles allow services (such as `backbeat` for replication) to perform actions on behalf of the account.

## Example

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

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Changing this replaces the resource. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Changing this replaces the resource. |
| `role_name` | String | Yes | Name of the IAM role. Changing this replaces the resource. |
| `assume_role_policy` | String | Yes | JSON trust policy document that grants entities permission to assume the role. Changing this replaces the resource. |

## Attribute Reference

| Name | Type | Description |
|------|------|-------------|
| `arn` | String | ARN of the created IAM role. |

## Notes

- All attributes force resource replacement. Scality's `UpdateRole` API only supports `MaxSessionDuration`, not trust policy changes, so any change requires a destroy-and-recreate.
- Attached policies must be detached before the role can be deleted. Terraform handles this automatically when `scality_iam_role_policy_attachment` resources reference this role.
