---
page_title: "scality_user_access_key Resource - scality"
subcategory: "IAM"
description: |-
  Generates an S3 access key pair for an IAM user. The secret key is only available at creation time and is stored in state.
---

# scality_user_access_key

Generates an S3 access key pair for an IAM user. The secret key is only available at creation time and is stored in state.

## Example

```hcl
resource "scality_user_access_key" "operator" {
  account_access_key = local.ak
  account_secret_key = local.sk
  username           = scality_user.operator.username
}

output "operator_credentials" {
  value = {
    access_key = scality_user_access_key.operator.access_key_id
    secret_key = scality_user_access_key.operator.secret_access_key
  }
  sensitive = true
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `username` | String | Yes | IAM username to create the key for. Forces replacement. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `access_key_id` | The access key ID. Sensitive. |
| `secret_access_key` | The secret access key. Sensitive. Only available at creation. |
| `status` | Key status (typically `Active`). |

## Notes

- The `secret_access_key` cannot be retrieved after creation. If state is lost, create a new key.
- Protect your state file -- it contains the secret key in plain text.
