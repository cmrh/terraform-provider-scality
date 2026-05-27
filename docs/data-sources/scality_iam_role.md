---
page_title: "scality_iam_role Data Source - scality"
subcategory: "IAM"
description: |-
  Looks up an existing IAM role by name within an account.
---

# scality_iam_role (Data Source)

Looks up an existing IAM role by name within an account. Use this to attach Terraform-managed sub-resources (`scality_iam_role_policy_attachment`) to a role created outside the current Terraform configuration.

## Example

```hcl
data "scality_iam_role" "backbeat" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  role_name          = "backbeat-crr"
}

resource "scality_iam_role_policy_attachment" "crr_write" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  role_name          = data.scality_iam_role.backbeat.role_name
  policy_arn         = "arn:aws:iam::123456789012:policy/CRRWriteAccess"
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account that owns the role.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account that owns the role.
- `role_name` (Required, String) — Name of the role to look up.

## Attribute Reference

- `id` (String) — Same as `role_name`.
- `arn` (String) — ARN of the role.
- `path` (String) — IAM path of the role (usually `/`).
- `assume_role_policy` (String) — Trust policy (AssumeRolePolicyDocument) attached to the role, as a JSON string.

## Notes

- If no role with the given name exists, the data source errors at plan time.
- Looks up by **name**, not ID.
