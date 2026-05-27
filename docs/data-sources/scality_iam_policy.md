---
page_title: "scality_iam_policy Data Source - scality"
subcategory: "IAM"
description: |-
  Looks up an existing customer-managed IAM policy by name.
---

# scality_iam_policy (Data Source)

Looks up an existing customer-managed IAM policy by name within an account. Returns the ARN and the default version's policy document.

Implementation note: lookup walks `ListPolicies` (scope `Local`) and matches by name. Scality ships no AWS-managed policies, so `Local` is the only relevant scope.

## Example

```hcl
data "scality_iam_policy" "read_only" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  policy_name        = "ReadOnlyAccess"
}

resource "scality_iam_role_policy_attachment" "attach" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  role_name          = "auditor"
  policy_arn         = data.scality_iam_policy.read_only.arn
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account that owns the policy.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account that owns the policy.
- `policy_name` (Required, String) — Name of the managed policy to look up.

## Attribute Reference

- `id` (String) — Same as `arn`.
- `arn` (String) — ARN of the managed policy.
- `policy_document` (String) — JSON policy document of the default version.

## Notes

- If no policy with the given name exists, the data source errors at plan time.
- Looks up by **name**, not ARN. To reference by ARN, just use the ARN string directly in attachments — no data source needed.
