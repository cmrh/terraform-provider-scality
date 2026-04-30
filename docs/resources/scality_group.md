---
page_title: "scality_group Resource - scality"
subcategory: "IAM"
description: |-
  Creates an IAM group within a Scality account. Groups let you manage permissions for multiple users collectively.
---

# scality_group

Creates an IAM group within a Scality account. Groups let you manage permissions for multiple users collectively.

## Example

```hcl
resource "scality_group" "developers" {
  account_access_key = local.ak
  account_secret_key = local.sk
  group_name         = "developers"
}

output "group_arn" {
  value = scality_group.developers.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `group_name` | String | Yes | Name of the group. Forces replacement. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `group_id` | Unique group identifier. |
| `arn` | ARN of the group. |
| `path` | IAM path of the group. |

## Notes

- Use `scality_group_membership` to add users to the group.
- Remove all members before deleting a group.
