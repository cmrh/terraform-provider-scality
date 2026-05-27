---
page_title: "scality_group Data Source - scality"
subcategory: "IAM"
description: |-
  Looks up an existing IAM group by name within an account.
---

# scality_group (Data Source)

Looks up an existing IAM group by name within an account. Use this to attach Terraform-managed sub-resources (`scality_group_membership`) to a group created outside the current Terraform configuration.

## Example

```hcl
data "scality_group" "ops" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  group_name         = "ops-team"
}

resource "scality_group_membership" "ops" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  group_name         = data.scality_group.ops.group_name
  users              = ["alice", "bob"]
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account that owns the group.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account that owns the group.
- `group_name` (Required, String) — Name of the group to look up.

## Attribute Reference

- `id` (String) — Same as `group_id`.
- `group_id` (String) — Stable unique identifier for the group.
- `arn` (String) — ARN of the group.
- `path` (String) — IAM path of the group (usually `/`).

## Notes

- If no group with the given name exists, the data source errors at plan time.
- Looks up by **name**, not ID.
