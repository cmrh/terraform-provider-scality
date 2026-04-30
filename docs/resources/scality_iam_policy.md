---
page_title: "scality_iam_policy Resource - scality"
subcategory: "IAM"
description: |-
  Manages an IAM managed policy within a Scality account. Managed policies can be attached to multiple roles, unlike inline policies which are embedded directly in a single entity.
---

# scality_iam_policy

Manages an IAM managed policy within a Scality account. Managed policies can be attached to multiple roles, unlike inline policies which are embedded directly in a single entity.

## Example

```hcl
resource "scality_iam_policy" "replication" {
  account_access_key = local.ak
  account_secret_key = local.sk
  policy_name        = "replication-policy"
  policy_document    = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:ReplicateObject", "s3:ReplicateDelete", "s3:ReplicateTags"]
        Resource = "arn:aws:s3:::my-bucket/*"
      },
    ]
  })
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Changing this replaces the resource. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Changing this replaces the resource. |
| `policy_name` | String | Yes | Name of the IAM policy. Changing this replaces the resource. |
| `policy_document` | String | Yes | JSON policy document. Can be updated in-place. |

## Attribute Reference

| Name | Type | Description |
|------|------|-------------|
| `arn` | String | ARN of the created IAM policy. |

## Notes

- Updating `policy_document` creates a new policy version via `CreatePolicyVersion`. Scality supports a maximum of 5 policy versions.
- Changing `policy_name` or account credentials forces resource replacement.
- The policy must be detached from all roles before it can be deleted. Use `scality_iam_role_policy_attachment` to manage attachments declaratively.
