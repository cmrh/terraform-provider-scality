---
page_title: "scality_group_membership Resource - scality"
subcategory: "IAM"
description: |-
  Manages the set of IAM users that belong to a group. This resource owns the full membership list -- users not listed will be removed from the group.
---

# scality_group_membership

Manages the set of IAM users that belong to a group. This resource owns the full membership list -- users not listed will be removed from the group.

## Example

```hcl
resource "scality_group_membership" "developers" {
  account_access_key = local.ak
  account_secret_key = local.sk
  group_name         = scality_group.developers.group_name

  users = [
    scality_user.alice.username,
    scality_user.bob.username,
  ]
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `group_name` | String | Yes | Target group. Forces replacement. |
| `users` | Set(String) | Yes | Complete set of usernames that should belong to the group. |

## Notes

- This resource is authoritative. It performs set-diff updates: users added to the list are added to the group, users removed from the list are removed from the group.
- Only one `scality_group_membership` resource should exist per group.

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the group name and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_group_membership.example GROUP_NAME
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_group_membership.example "ACCESS_KEY:SECRET_KEY:GROUP_NAME"
```
