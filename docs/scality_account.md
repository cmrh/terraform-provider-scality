# scality_account

Manages a Scality account via the IAM API (AWS Signature V4 auth). Automatically generates S3 credentials on creation.

## Example

```hcl
resource "scality_account" "app" {
  name          = "my-app"
  email_address = "myapp@example.com"
  quota_max     = 53687091200  # 50 GB

  custom_attributes = {
    environment = "production"
    team        = "platform"
  }
}

output "s3_credentials" {
  value = {
    access_key = scality_account.app.access_key
    secret_key = scality_account.app.secret_key
  }
  sensitive = true
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | Account name. Forces replacement. |
| `email_address` | String | Yes | Account email. Forces replacement. |
| `quota_max` | Int | No | Max bytes storable. `0` = unlimited. Forces replacement. |
| `external_account_id` | String | No | External ID for integration with other systems. Forces replacement. |
| `custom_attributes` | Map(String) | No | Key-value metadata. Max 10 per account. Updated in-place. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `id` | Scality account ID. |
| `arn` | Account ARN. |
| `canonical_id` | Canonical ID (used in bucket policies). |
| `create_date` | Creation timestamp (ISO 8601). |
| `access_key` | S3 access key. Sensitive. Only available at creation. |
| `secret_key` | S3 secret key. Sensitive. Only available at creation. |

## Import

```bash
tofu import scality_account.app my-account-name
```

After import, `access_key` and `secret_key` will be unknown. Generate new keys with `scality_account_access_key` if needed.

## Notes

- Uses admin credentials from the provider configuration.
- All arguments except `custom_attributes` force replacement on change.
- `custom_attributes` uses `UpdateAccountAttributes`, which does a full overwrite of all attributes.
- Accounts with existing resources (users, buckets, policies) cannot be deleted. Clean up child resources first.
