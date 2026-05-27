---
page_title: "scality_iam_roles Data Source - scality"
subcategory: "IAM"
description: |-
  Lists all IAM roles in the calling account.
---

# scality_iam_roles (Data Source)

Lists all IAM roles in the calling account. Use this for inventory, audit, or dashboard tooling — `for_each` over the result and drill down with [`data.scality_iam_role`](scality_iam_role.md) per-role. Pagination is handled inside the client.

## Example

```hcl
data "scality_iam_roles" "all" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
}

output "role_names" {
  value = [for r in data.scality_iam_roles.all.roles : r.role_name]
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account whose roles to list.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account whose roles to list.

## Attribute Reference

- `id` (String) — Synthetic identifier for the data source instance.
- `roles` (List of Object) — List of IAM roles in the account. Each entry exposes:
  - `role_id` (String) — Stable unique identifier for the role.
  - `role_name` (String) — Role name.
  - `arn` (String) — ARN of the role.
  - `path` (String) — IAM path of the role.
  - `create_date` (String) — Role creation date.

## Notes

- Empty account returns an empty list, not an error.
- The list view does not include trust policies. Use [`data.scality_iam_role`](scality_iam_role.md) for per-role drill-down.
