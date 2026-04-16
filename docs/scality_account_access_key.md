# scality_account_access_key

Generates an additional S3 access key pair for an account. Useful for creating a stable credential that Terraform owns exclusively, independent of credentials that may be rotated externally.

## Example

```hcl
resource "scality_console_account" "app" {
  account_name             = "my-app"
  email                    = "app@example.com"
  generate_random_password = true
}

# Create a second key pair that external processes won't touch
resource "scality_account_access_key" "stable" {
  account_access_key = scality_console_account.app.access_key
  account_secret_key = scality_console_account.app.secret_key
}

output "stable_credentials" {
  value = {
    access_key = scality_account_access_key.stable.access_key
    secret_key = scality_account_access_key.stable.secret_key
  }
  sensitive = true
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Existing access key for the account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Existing secret key for the account. Sensitive. Forces replacement. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `id` | Access key ID (same as `access_key`). |
| `access_key` | The new access key. Sensitive. |
| `secret_key` | The new secret key. Sensitive. Only available at creation. |

## Notes

- The `secret_key` cannot be retrieved after creation. If state is lost, create a new key.
- The Read operation preserves state without querying the API (the secret key is not retrievable).
- Protect your state file -- it contains the secret key in plain text.
