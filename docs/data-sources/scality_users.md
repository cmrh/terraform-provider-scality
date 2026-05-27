---
page_title: "scality_users Data Source - scality"
subcategory: "IAM"
description: |-
  Lists all IAM users in the calling account.
---

# scality_users (Data Source)

Lists all IAM users in the calling account. Use this for inventory, audit, or dashboard tooling — `for_each` over the result and drill down with [`data.scality_user`](scality_user.md) per-user. Pagination is handled inside the client.

## Example

```hcl
data "scality_users" "all" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
}

output "user_names" {
  value = [for u in data.scality_users.all.users : u.username]
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account whose users to list.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account whose users to list.

## Attribute Reference

- `id` (String) — Synthetic identifier for the data source instance.
- `users` (List of Object) — List of IAM users in the account. Each entry exposes:
  - `user_id` (String) — Stable unique identifier for the user.
  - `username` (String) — User name.
  - `arn` (String) — ARN of the user.
  - `path` (String) — IAM path of the user.
  - `create_date` (String) — User creation date.

## Notes

- Empty account returns an empty list, not an error.
- The list view does not include attached policies or access keys. Use [`data.scality_user`](scality_user.md) for per-user drill-down.
