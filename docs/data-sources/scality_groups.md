---
page_title: "scality_groups Data Source - scality"
subcategory: "IAM"
description: |-
  Lists all IAM groups in the calling account.
---

# scality_groups (Data Source)

Lists all IAM groups in the calling account. Use this for inventory, audit, or dashboard tooling — `for_each` over the result and drill down with [`data.scality_group`](scality_group.md) per-group. Pagination is handled inside the client.

## Example

```hcl
data "scality_groups" "all" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
}

output "group_names" {
  value = [for g in data.scality_groups.all.groups : g.group_name]
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account whose groups to list.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account whose groups to list.

## Attribute Reference

- `id` (String) — Synthetic identifier for the data source instance.
- `groups` (List of Object) — List of IAM groups in the account. Each entry exposes:
  - `group_id` (String) — Stable unique identifier for the group.
  - `group_name` (String) — Group name.
  - `arn` (String) — ARN of the group.
  - `path` (String) — IAM path of the group.
  - `create_date` (String) — Group creation date.

## Notes

- Empty account returns an empty list, not an error.
- The list view does not include group memberships. Use [`data.scality_group`](scality_group.md) for per-group drill-down.
