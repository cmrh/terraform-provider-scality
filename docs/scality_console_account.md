# scality_console_account - Manage Scality Accounts via Console API

Manages Scality accounts using the Console API with JWT token authentication. Accounts are created **without passwords** (security best practice) and persistent S3 credentials are generated automatically.

## Example Usage

### Basic Usage

```hcl
provider "scality" {
  console_endpoint = "http://10.164.169.247:8080"
  console_username = var.console_username
  console_password = var.console_password
}

resource "scality_console_account" "example" {
  account_name = "my-app"
  email        = "myapp@example.com"
  quota        = 50000000000  # 50GB
}

output "access_key" {
  value     = scality_console_account.example.access_key
  sensitive = true
}

output "secret_key" {
  value     = scality_console_account.example.secret_key
  sensitive = true
}
```

### Using Environment Variables

```bash
export SCALITY_CONSOLE_ENDPOINT="http://10.164.169.247:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="admin-password"
```

```hcl
provider "scality" {
  # Configuration loaded from environment variables
}

resource "scality_console_account" "example" {
  account_name = "myapp"
  email        = "myapp@example.com"
  quota        = 10000000000
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

### Integration with AWS Secrets Manager

```hcl
resource "scality_console_account" "app" {
  account_name = "production-app"
  email        = "prod@example.com"
  quota        = 500000000000
}

resource "aws_secretsmanager_secret" "scality_credentials" {
  name = "scality/${scality_console_account.app.account_name}/credentials"
}

resource "aws_secretsmanager_secret_version" "scality_credentials" {
  secret_id = aws_secretsmanager_secret.scality_credentials.id
  secret_string = jsonencode({
    account_name = scality_console_account.app.account_name
    access_key   = scality_console_account.app.access_key
    secret_key   = scality_console_account.app.secret_key
    endpoint     = var.scality_endpoint
  })
}
```

## Argument Reference

### Required Arguments

- `account_name` (String, Required, Forces new resource) - Name of the account. Changing this forces a new resource to be created.
- `email` (String, Required) - Email address for the account. Must be a valid email format.

### Optional Arguments

- `quota` (Number, Optional, Default: 0) - Maximum amount of bytes storable by the account. `0` means unlimited.

## Attribute Reference

In addition to the arguments, the following attributes are exported:

- `id` (String) - Account identifier (same as `account_name`).
- `created_at` (String) - Account creation timestamp in ISO 8601 format.
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
- **Create Account**: `POST /_/console/vault/accounts` (creates account without password)
- **Generate Access Key**: `POST /_/console/vault/accounts/{name}/keys` (persistent credentials)
- **Get Account**: `GET /_/console/vault/accounts/{name}` (for state refresh)
- **Delete Account**: Two-step process:
  1. `DELETE /_/console/vault/accounts/{name}` (delete account)
  2. `DELETE /_/console/vault/accounts/{name}/user` (delete associated user)

## Key Differences from scality_account (IAM API)

| Feature | scality_account (IAM) | scality_console_account (Console) |
|---------|----------------------|-----------------------------------|
| **Authentication** | AWS Signature V4 (per-request) | JWT Token (cached 23.5hrs) |
| **Account Creation** | Standard process | **Without password** (security best practice) |
| **Credentials Type** | Standard S3 access keys | **Persistent keys** (not password-linked) |
| **Performance** | Per-request signing overhead | Cached token reduces overhead |
| **Deletion Process** | Single-step | Two-step (account + user) |
| **Account Attribute** | `name` | `account_name` |
| **Email Attribute** | `email_address` | `email` |
| **Quota Attribute** | `quota_max` | `quota` |
| **Provider Config** | `endpoint`, `access_key`, `secret_key` | `console_endpoint`, `console_username`, `console_password` |
| **Use Case** | Traditional IAM-style management | Console UI integration, no password management |

## Security Considerations

### 1. Password-Free Account Creation

Accounts are created **without passwords** by design:
- **More secure** than password-linked credentials
- Persistent keys won't be invalidated by password changes
- Eliminates password management overhead
- Follows modern security best practices

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
  account_name = "myapp"
  email        = "myapp@example.com"
  quota        = 10000000000
}
```

On `terraform apply`:
1. Provider authenticates to Console API (or uses cached token)
2. Creates account without password
3. Generates persistent S3 access keys
4. Returns credentials in state (encrypted if using remote state)

### Updates

Changing the `email` or `quota` attributes triggers an update:
```hcl
resource "scality_console_account" "example" {
  account_name = "myapp"
  email        = "newemail@example.com"  # Updated
  quota        = 20000000000              # Updated
}
```

**Note**: Console API update support may be limited. Check API documentation for update capabilities. Some changes may fail or require replacement.

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

**Solution**: Ensure Console API credentials are configured:
```hcl
provider "scality" {
  console_endpoint = "http://10.164.169.247:8080"
  console_username = var.console_username
  console_password = var.console_password
}
```

Or via environment variables:
```bash
export SCALITY_CONSOLE_ENDPOINT="http://10.164.169.247:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="admin-password"
```

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
