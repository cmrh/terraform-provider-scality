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

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the group name and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_group.example GROUP_NAME
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_group.example "ACCESS_KEY:SECRET_KEY:GROUP_NAME"
```
