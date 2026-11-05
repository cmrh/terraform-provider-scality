---
page_title: "scality_iam_user_policy_attachment Resource - scality"
subcategory: "IAM"
description: |-
  Attaches an IAM managed policy to an IAM user within a Scality account.
---

# scality_iam_user_policy_attachment

Attaches an IAM managed policy to an IAM user within a Scality account.

## Example

```hcl
resource "scality_iam_user_policy_attachment" "app" {
  account_access_key = local.ak
  account_secret_key = local.sk
  username           = scality_user.app.username
  policy_arn         = scality_iam_policy.read_only.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | No | Access key of the owning account. Sensitive. Switching to another key of the same account updates in place. Omit to use the provider's assume_role credentials. |
| `account_secret_key` | String | No | Secret key of the owning account. Sensitive. Switching to another key of the same account updates in place. Omit to use the provider's assume_role credentials. |
| `username` | String | Yes | Name of the IAM user to attach the policy to. Changing this replaces the resource. |
| `policy_arn` | String | Yes | ARN of the IAM managed policy to attach. Changing this replaces the resource. |

## Notes

- Changing `username` or `policy_arn` replaces the attachment: Terraform detaches the old policy and attaches the new one.
- If the attachment or its user is removed outside of Terraform, the resource is removed from state on the next plan/apply.
- A user with attached policies cannot be deleted. Referencing the user from this resource makes Terraform detach the policy before deleting the user.

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the identity portion and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_iam_user_policy_attachment.example USERNAME:POLICY_ARN
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_iam_user_policy_attachment.example "ACCESS_KEY:SECRET_KEY:USERNAME:POLICY_ARN"
```
