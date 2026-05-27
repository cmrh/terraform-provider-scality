---
page_title: "scality_iam_policies Data Source - scality"
subcategory: "IAM"
description: |-
  Lists all customer-managed IAM policies in the calling account.
---

# scality_iam_policies (Data Source)

Lists all customer-managed (scope `Local`) IAM policies in the calling account. Use this for inventory, audit, or dashboard tooling — `for_each` over the result and drill down with [`data.scality_iam_policy`](scality_iam_policy.md) per-policy. Pagination is handled inside the client.

## Example

```hcl
data "scality_iam_policies" "all" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
}

output "policy_names" {
  value = [for p in data.scality_iam_policies.all.policies : p.policy_name]
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account whose policies to list.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account whose policies to list.

## Attribute Reference

- `id` (String) — Synthetic identifier for the data source instance.
- `policies` (List of Object) — List of managed policies in the account. Each entry exposes:
  - `policy_id` (String) — Stable unique identifier for the policy.
  - `policy_name` (String) — Policy name.
  - `arn` (String) — ARN of the policy.
  - `path` (String) — IAM path of the policy.
  - `default_version_id` (String) — Default version identifier for the policy document.
  - `attachment_count` (Number) — Number of principals (users, groups, roles) the policy is attached to.
  - `create_date` (String) — Policy creation date.
  - `update_date` (String) — Last update date.

## Notes

- Empty account returns an empty list, not an error.
- The list view does not include policy documents. Use [`data.scality_iam_policy`](scality_iam_policy.md) for per-policy drill-down.
