---
page_title: "scality_iam_role_policy_attachment Resource - scality"
subcategory: "IAM"
description: |-
  Attaches an IAM managed policy to an IAM role within a Scality account.
---

# scality_iam_role_policy_attachment

Attaches an IAM managed policy to an IAM role within a Scality account.

## Example

```hcl
resource "scality_iam_role_policy_attachment" "replication" {
  account_access_key = local.ak
  account_secret_key = local.sk
  role_name          = scality_iam_role.replication.role_name
  policy_arn         = scality_iam_policy.replication.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | No | Access key of the owning account. Sensitive. Changing this replaces the resource. Omit to use the provider's assume_role credentials. |
| `account_secret_key` | String | No | Secret key of the owning account. Sensitive. Changing this replaces the resource. Omit to use the provider's assume_role credentials. |
| `role_name` | String | Yes | Name of the IAM role to attach the policy to. Changing this replaces the resource. |
| `policy_arn` | String | Yes | ARN of the IAM managed policy to attach. Changing this replaces the resource. |

## Notes

- All attributes force resource replacement. To change the attached policy, Terraform will detach the old policy and attach the new one.
- If the attachment is removed outside of Terraform, the resource will be removed from state on the next plan/apply.

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the identity portion and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_iam_role_policy_attachment.example ROLE_NAME:POLICY_ARN
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_iam_role_policy_attachment.example "ACCESS_KEY:SECRET_KEY:ROLE_NAME:POLICY_ARN"
```
