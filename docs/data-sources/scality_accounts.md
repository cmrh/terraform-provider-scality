---
page_title: "scality_accounts Data Source - scality"
subcategory: "Accounts"
description: |-
  Lists all Scality accounts in the cluster. Useful for inventory, audit, and dashboard tooling.
---

# scality_accounts (Data Source)

Lists every Scality account in the cluster. Use this for inventory, cross-account audit, or to feed a `for_each` over all tenants — combine with `data.scality_account` for per-account drill-down.

The list does **not** include `custom_attributes` on each entry — those require a per-account API call. If you need attributes, iterate over `data.scality_accounts.accounts` with `for_each` into `data.scality_account` (composition pattern shown below).

Empty cluster returns an empty `accounts` list, not an error.

## Example

```hcl
data "scality_accounts" "all" {}

output "account_names" {
  value = [for a in data.scality_accounts.all.accounts : a.name]
}

# Composition pattern — drill down into each account for full details:

data "scality_account" "per_account" {
  for_each = { for a in data.scality_accounts.all.accounts : a.name => a }
  name     = each.value.name
}

output "full_attributes" {
  value = {
    for name, account in data.scality_account.per_account :
    name => account.custom_attributes
  }
}
```

## Argument Reference

This data source has no input arguments.

## Attribute Reference

- `id` (String) — Synthetic identifier for the data source instance.
- `accounts` (List of Object) — All accounts in the cluster. Each element has:
  - `id` (String) — Vault account ID.
  - `name` (String) — Account name.
  - `email_address` (String) — Email address registered against the account.
  - `arn` (String) — Amazon Resource Name of the account.
  - `canonical_id` (String) — Canonical ID, used in S3 bucket policies.
  - `create_date` (String) — Account creation date.
  - `quota_max` (Number) — Maximum bytes storable (0 = unlimited).

## Notes

- Uses the provider's `vault_admin_*` credentials (same as the `scality_account` resource and data source).
- Pagination is handled inside the provider — the data source returns a fully-walked flat list.
- Filtering inputs are not currently supported. Use HCL `for` expressions on the result to filter client-side.
- For per-account `custom_attributes`, use `data.scality_account` as shown in the composition pattern above.
