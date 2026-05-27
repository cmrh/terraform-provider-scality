---
page_title: "scality_console_account Resource - scality"
subcategory: "Accounts"
description: |-
  Manages a Scality account via the Console API (JWT auth). Optionally generates a random password for Console UI access. S3 credentials are generated automatically.
---

# scality_console_account

Manages a Scality account via the Console API (JWT auth). Optionally generates a random password for Console UI access. S3 credentials are generated automatically.

## Example

```hcl
resource "scality_console_account" "app" {
  account_name             = "my-app"
  email                    = "myapp@example.com"
  quota                    = 53687091200  # 50 GB
  generate_random_password = true
}

output "console_login" {
  value = {
    account_name = scality_console_account.app.account_name
    password     = scality_console_account.app.password
  }
  sensitive = true
}

output "s3_credentials" {
  value = {
    access_key = scality_console_account.app.access_key
    secret_key = scality_console_account.app.secret_key
  }
  sensitive = true
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Account name. Forces replacement. |
| `email` | String | Yes | Account email. Forces replacement. |
| `quota` | Int | No | Max bytes storable. `0` = unlimited. Forces replacement. |
| `generate_random_password` | Bool | No | Generate a random Console password. Default `false`. Forces replacement. |
| `password_length` | Int | No | Password length. Minimum and default is 16. Forces replacement. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `id` | Account identifier (same as `account_name`). |
| `created_at` | Creation timestamp (ISO 8601). |
| `password` | Generated Console password. Sensitive. Only set if `generate_random_password` is true. |
| `access_key` | S3 access key. Sensitive. Only available at creation. |
| `secret_key` | S3 secret key. Sensitive. Only available at creation. |

## Import

```bash
tofu import scality_console_account.app my-account-name
```

After import, credentials and password will be unknown.

## Prerequisites

Console superadmin credentials must exist (created during Scality deployment):

```bash
ansible-playbook -i env/s3config/inventory \
  tooling-playbooks/create-superadmin-console-user.yml \
  -e ui_username=admin -e ui_password=mySuperPassword
```

## Notes

- Uses Console API credentials from the provider configuration (`console_endpoint`, `console_username`, `console_password`).
- All arguments force replacement. The Console API does not support in-place updates.
- Deletion is a two-step process (account + associated user), handled automatically.
- The Console API has no per-account GET endpoint. When the provider is also configured with IAM admin credentials (`endpoint`, `access_key`, `secret_key`), Read probes Vault via `GetAccount` to detect out-of-band deletion and removes the resource from state if the account is gone. When the provider is configured Console-only, Read preserves state without probing — drift detection is best-effort.
- Generated passwords include uppercase, lowercase, digits, and special characters. Ambiguous characters (0, O, 1, l, I) are excluded.
