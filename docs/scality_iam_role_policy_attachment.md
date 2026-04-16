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
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Changing this replaces the resource. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Changing this replaces the resource. |
| `role_name` | String | Yes | Name of the IAM role to attach the policy to. Changing this replaces the resource. |
| `policy_arn` | String | Yes | ARN of the IAM managed policy to attach. Changing this replaces the resource. |

## Notes

- All attributes force resource replacement. To change the attached policy, Terraform will detach the old policy and attach the new one.
- If the attachment is removed outside of Terraform, the resource will be removed from state on the next plan/apply.
