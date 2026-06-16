---
page_title: "scality_account_access_key Resource - scality"
subcategory: "Accounts"
description: |-
  Generates an additional S3 access key pair for an account. Useful for creating a stable credential that Terraform owns exclusively, independent of credentials that may be rotated externally.
---

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

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the access-key ID and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_account_access_key.example ACCESS_KEY_ID
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_account_access_key.example "ACCOUNT_ACCESS_KEY:ACCOUNT_SECRET_KEY:ACCESS_KEY_ID"
```

### Adoption: rotate, don't import

Importing an existing key binds its identity but cannot recover the secret — the Vault IAM API returns `secret_key` only at creation and never again. After import, `secret_key` is empty in state, so any downstream resource that reads it (a `scality_bucket` configured with these credentials, a derived `local_file`, a chained provider) will fail.

To bring an existing manually-created access key under Terraform management, create a new managed key and retire the manually-created one:

1. Add a `scality_account_access_key` resource for the account. `terraform apply` creates a new key and records the secret in state.
2. Update downstream resources to consume the new credentials.
3. Delete the manually-created key out-of-band (or import-then-`terraform destroy` it).

An account supports up to 4 access keys. If already at the cap, delete an unused one before applying.
