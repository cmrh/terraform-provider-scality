# scality_console_account - Manage Scality Accounts via Console API

Manages Scality accounts using the Console API with JWT token authentication. Optionally generates random passwords for Console access. Persistent S3 credentials are generated automatically.

## Prerequisites

Console superadmin credentials must be created during Scality deployment via Ansible:

```bash
ansible-playbook -i env/s3config/inventory \
  tooling-playbooks/create-superadmin-console-user.yml \
  -e ui_username=admin -e ui_password=mySuperPassword
```

## Example Usage

### Basic Usage (Recommended)

**Always use environment variables for credentials** (never hardcode in `.tf` files):

```bash
export SCALITY_CONSOLE_ENDPOINT="http://scality.example.com:8080"
export SCALITY_CONSOLE_USERNAME="admin"           # From Ansible deployment
export SCALITY_CONSOLE_PASSWORD="mySuperPassword" # From Ansible deployment
```

```hcl
provider "scality" {
  # Credentials loaded from environment variables
}

resource "scality_console_account" "example" {
  account_name = "my-app"
  email        = "myapp@example.com"
  quota        = 53687091200  # 50GB
}

output "s3_credentials" {
  value = {
    access_key = scality_console_account.example.access_key
    secret_key = scality_console_account.example.secret_key
  }
  sensitive = true
}
```

### With Console Password Generation

Generate a random password for Console UI access:

```hcl
resource "scality_console_account" "with_password" {
  account_name             = "admin-user"
  email                    = "admin@example.com"
  quota                    = 53687091200  # 50GB
  generate_random_password = true
  password_length          = 20  # Optional, default 16, minimum 16
}

output "console_credentials" {
  value = {
    account_name = scality_console_account.with_password.account_name
    password     = scality_console_account.with_password.password
    access_key   = scality_console_account.with_password.access_key
    secret_key   = scality_console_account.with_password.secret_key
  }
  sensitive = true
}
```

### Multiple Accounts

```hcl
locals {
  console_accounts = {
    app1 = { email = "app1@example.com", quota = 10000000000 }
    app2 = { email = "app2@example.com", quota = 20000000000 }
    app3 = { email = "app3@example.com", quota = 30000000000 }
  }
}

resource "scality_console_account" "apps" {
  for_each = local.console_accounts

  account_name = "console-${each.key}"
  email        = each.value.email
  quota        = each.value.quota
}

output "app_credentials" {
  value = {
    for k, v in scality_console_account.apps : k => {
      access_key = v.access_key
      secret_key = v.secret_key
    }
  }
  sensitive = true
}
```

### Save Credentials to File

```hcl
resource "scality_console_account" "app" {
  account_name = "backend-service"
  email        = "backend@example.com"
  quota        = 100000000000
}

resource "local_file" "credentials" {
  content = <<-EOT
    [${scality_console_account.app.account_name}]
    aws_access_key_id = ${scality_console_account.app.access_key}
    aws_secret_access_key = ${scality_console_account.app.secret_key}
  EOT

  filename        = "./credentials/${scality_console_account.app.account_name}.ini"
  file_permission = "0600"
}
```


## Argument Reference

### Required Arguments

- `account_name` (String, Required, Forces new resource) - Name of the account. Changing this forces a new resource to be created.
- `email` (String, Required, Forces new resource) - Email address for the account. Changing this forces a new resource to be created.

### Optional Arguments

- `quota` (Number, Optional, Default: 0, Forces new resource) - Maximum amount of bytes storable by the account. `0` means unlimited. Changing this forces a new resource to be created.
- `generate_random_password` (Boolean, Optional, Default: false, Forces new resource) - Generate a random password for Console access. Changing this forces a new resource to be created.
- `password_length` (Number, Optional, Default: 16, Forces new resource) - Length of generated password. Minimum 16 characters. Only used if `generate_random_password` is true. Changing this forces a new resource to be created.

## Attribute Reference

In addition to the arguments, the following attributes are exported:

- `id` (String) - Account identifier (same as `account_name`).
- `created_at` (String) - Account creation timestamp in ISO 8601 format.
- `password` (String, Sensitive, Computed) - Generated Console password. Only available if `generate_random_password` is true. This value is only known after creation and cannot be retrieved later.
- `access_key` (String, Sensitive) - S3 API access key (persistent credentials, generated automatically). This value is only known after creation and cannot be retrieved later.
- `secret_key` (String, Sensitive) - S3 API secret key (persistent credentials, generated automatically). This value is only known after creation and cannot be retrieved later.

## Import

Console accounts can be imported using the account name:

```bash
terraform import scality_console_account.example my-account-name
```

**Note**: After import, the `access_key` and `secret_key` attributes will be unknown since they cannot be retrieved from the API. You'll need to generate new keys or maintain existing credentials separately.

## Authentication Details

### JWT Token Authentication

The provider uses JWT token authentication with the Console API:

1. **Initial Authentication**:
   - Endpoint: `POST /_/console/authenticate`
   - Sends username and password
   - Receives JWT token (24-hour lifetime)

2. **Token Caching**:
   - Tokens cached in `/tmp/.scality_console_token_<hash>`
   - Hash based on MD5 of `endpoint:username`
   - Cache lifetime: 23.5 hours (safety margin before expiry)
   - File permissions: 0600 (owner-only access)
   - Reduces authentication overhead for multiple operations

3. **Resource Operations**:
   - All API calls include `x-access-token` header with JWT
   - Automatic re-authentication if token expires

### API Endpoints Used

- **Authenticate**: `POST /_/console/authenticate`
- **Create Account**: `POST /_/console/vault/accounts` (optionally includes password field)
- **Generate Access Key**: `POST /_/console/vault/accounts/{name}/keys` (persistent credentials)
- **Get Account**: `GET /_/console/vault/accounts/{name}` (for state refresh)
- **Delete Account**: Two-step process:
  1. `DELETE /_/console/vault/accounts/{name}` (delete account)
  2. `DELETE /_/console/vault/accounts/{name}/user` (delete associated user)

## Key Differences from scality_account (IAM API)

| Feature | scality_account (IAM) | scality_console_account (Console) |
|---------|----------------------|-----------------------------------|
| **Authentication** | AWS Signature V4 (per-request) | JWT Token (cached 23.5hrs) |
| **Account Creation** | Standard process | With or without password (optional) |
| **Password Generation** | Not supported | Optional random password generation |
| **Credentials Type** | Standard S3 access keys | **Persistent keys** (not password-linked) |
| **Performance** | Per-request signing overhead | Cached token reduces overhead |
| **Deletion Process** | Single-step | Two-step (account + user) |
| **Account Attribute** | `name` | `account_name` |
| **Email Attribute** | `email_address` | `email` |
| **Quota Attribute** | `quota_max` | `quota` |
| **Provider Config** | `endpoint`, `access_key`, `secret_key` | `console_endpoint`, `console_username`, `console_password` |
| **Use Case** | Traditional IAM-style management | Console UI integration with optional password |

## Security Considerations

### 1. Optional Password Generation

Accounts can be created with or without Console passwords:

**Without Password (Default)**:
- More secure for service accounts and automation
- Persistent S3 keys won't be invalidated by password changes
- Eliminates password management overhead

**With Random Password** (`generate_random_password = true`):
- Generates cryptographically secure random passwords (minimum 16 characters)
- Useful for human Console UI access
- Password includes uppercase, lowercase, digits, and special characters
- Excludes ambiguous characters (0, O, 1, l, I) for clarity
- Password stored in Terraform state as sensitive value

**Note**: According to API documentation, password-linked credentials are automatically rotated when passwords change. For persistent access keys, the provider generates separate credentials via the `POST /_/console/vault/accounts/{name}/keys` endpoint.

### 2. Token Caching Security

- Cache files stored in `/tmp` with `0600` permissions (owner-only access)
- Each endpoint/username combination has a unique cache file
- Tokens automatically expire after 23.5 hours
- Failed cache operations don't affect resource operations (automatic re-authentication)

### 3. Credential Protection in State

The Terraform state file contains sensitive credentials:

```hcl
# Use remote state with encryption
terraform {
  backend "s3" {
    bucket  = "terraform-state"
    key     = "scality/terraform.tfstate"
    encrypt = true
  }
}
```

**Always:**
- Use remote state with encryption
- Restrict access to state files
- Never commit state files to version control
- Use sensitive outputs

### 4. Sensitive Output Handling

```hcl
output "access_key" {
  value     = scality_console_account.example.access_key
  sensitive = true
}

output "secret_key" {
  value     = scality_console_account.example.secret_key
  sensitive = true
}
```

View sensitive outputs:
```bash
terraform output -json | jq '.access_key.value'
terraform output -json | jq '.secret_key.value'
```

### 5. Use HTTPS in Production

```hcl
provider "scality" {
  console_endpoint = "https://scality.example.com:8443"
  # ... other config
}
```

## Error Handling

### HTTP 200/201 - Success
Account created or deleted successfully.

### HTTP 400 - Bad Request
```
Error: Unable to create Console account: unexpected status 400: Bad Request
```
**Resolution**: Verify email format is valid (`user@domain.tld`) and all required fields are provided.

### HTTP 401 - Unauthorized
```
Error: Authentication failed with status 401
```
**Resolution**:
- Verify Console username and password are correct
- Check that Console API is accessible at the specified endpoint
- Ensure credentials have admin permissions

### HTTP 409 - Conflict
```
Error: account already exists
```
**Resolution**: The account name is already in use. Choose a different name or import the existing account.

### HTTP 500 - Service Failure
```
Error: Unable to create Console account: unexpected status 500
```
**Resolution**: The Console API service encountered an internal error. Check Console API logs and service health.

### Token Cache Errors
Token cache read/write failures are handled gracefully:
- Cache failures don't prevent resource operations
- Provider automatically re-authenticates if cache is unavailable
- No user intervention required

## State Management

### Drift Detection

Terraform will detect if an account is deleted outside of Terraform:
- On `terraform plan` or `terraform apply`, the provider queries the Console API
- If the account no longer exists, it's removed from state
- Next apply will recreate the resource if still defined in configuration

### Credential Limitations

**Important**: The `access_key` and `secret_key` attributes cannot be retrieved after creation:
- These values are only available in the create response
- Subsequent state refreshes don't update these values
- If state is lost, you'll need to generate new credentials or rotate keys

### State File Security

Protect your state files:
```bash
# Add to .gitignore
echo "terraform.tfstate*" >> .gitignore
echo "*.tfvars" >> .gitignore
echo ".terraform/" >> .gitignore
```

## Lifecycle Management

### Creation
```hcl
resource "scality_console_account" "example" {
  account_name             = "myapp"
  email                    = "myapp@example.com"
  quota                    = 10000000000
  generate_random_password = true  # Optional
  password_length          = 20    # Optional, defaults to 16
}
```

On `terraform apply`:
1. Provider authenticates to Console API (or uses cached token)
2. Generates random password if requested (cryptographically secure, minimum 16 chars)
3. Creates account with optional password
4. Saves account to state immediately (so it can be tracked even if subsequent steps fail)
5. Generates persistent S3 access keys
6. Updates state with credentials

If access key generation fails, the account is still tracked in state and can be destroyed or re-applied.

### Updates

All attribute changes force resource replacement (destroy + recreate). The Console API does not support in-place updates. Changing any attribute (`account_name`, `email`, `quota`, `generate_random_password`, `password_length`) will destroy the existing account and create a new one:

```hcl
resource "scality_console_account" "example" {
  account_name = "myapp"
  email        = "newemail@example.com"  # Changed - will force replacement
  quota        = 20000000000              # Changed - will force replacement
}
```

**Warning**: Replacement destroys the existing account and its credentials. Ensure dependent resources are updated accordingly.

### Deletion

On `terraform destroy`:
1. Provider authenticates (or uses cached token)
2. Executes two-step deletion:
   - Deletes account entity
   - Deletes associated user entity
3. Removes resource from state

**Important**: Accounts with existing buckets, data, or resources may fail to delete. Clean up resources first.

## Troubleshooting

### Provider Configuration Issues

**Error**: `Missing Console Client Configuration`

**Solution**: Configure Console API credentials via environment variables (recommended):
```bash
export SCALITY_CONSOLE_ENDPOINT="http://scality.example.com:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="mySuperPassword"
```

Then in your Terraform configuration:
```hcl
provider "scality" {
  # Credentials loaded from environment variables
}
```

> **Security**: Never hardcode credentials in `.tf` files that may be committed to git.

### Authentication Failures

**Error**: `authentication failed with status 401`

**Checklist**:
1. Verify Console API is accessible: `curl http://10.164.169.247:8080/_/console/authenticate`
2. Check username and password are correct
3. Ensure Console API service is running
4. Verify firewall allows access to port 8080

### Account Already Exists

**Error**: `account already exists`

**Options**:
1. **Import existing account**:
   ```bash
   terraform import scality_console_account.example existing-account-name
   ```

2. **Use different name**:
   ```hcl
   account_name = "myapp-v2"  # Changed
   ```

### Deletion Failures

**Error**: Resource deletion fails silently or with timeout

**Resolution**: The account may have attached resources. The Console API requires accounts to be empty before deletion. Manually check and clean up:
1. Delete all S3 buckets and their contents
2. Remove any IAM users or policies
3. Retry deletion: `terraform destroy`

### Enable Debug Logging

```bash
# Set Terraform log level
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform-debug.log

# Run terraform
terraform apply

# Review logs
cat terraform-debug.log | grep -i "console"
```

## See Also

- [scality_account](scality_account.md) - IAM API resource for comparison
- [Provider Configuration](provider.md) - Detailed provider configuration
- [Quick Start Guide](QUICKSTART.md) - Get started quickly
- [Examples](EXAMPLES.md) - More examples and patterns
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions

## API Reference

Based on Scality Console API specifications:
- [Authentication API](../../requirements/console/authenticate.rst)
- [Create Account API](../../requirements/console/create_new_account.rst)
- [Generate Account Key API](../../requirements/console/generate_account_key.rst)
- [Delete Account API](../../requirements/console/delete_account.rst)
