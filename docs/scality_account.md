# scality_account - Manage Scality Accounts via IAM API

Manages Scality accounts using the IAM-style API with AWS Signature Version 4 authentication. Automatically generates S3 API credentials upon account creation.

## Example Usage

### Basic Usage

```hcl
provider "scality" {
  endpoint   = "http://10.164.169.247"
  access_key = var.scality_access_key
  secret_key = var.scality_secret_key
}

resource "scality_account" "example" {
  name          = "my-app"
  email_address = "myapp@example.com"
  quota_max     = 50000000000  # 50GB
}

output "access_key" {
  value     = scality_account.example.access_key
  sensitive = true
}

output "secret_key" {
  value     = scality_account.example.secret_key
  sensitive = true
}
```

### Using Environment Variables

```bash
export SCALITY_ENDPOINT="http://10.164.169.247"
export SCALITY_ACCESS_KEY="your-admin-access-key"
export SCALITY_SECRET_KEY="your-admin-secret-key"
```

```hcl
provider "scality" {
  # Configuration loaded from environment variables
}

resource "scality_account" "example" {
  name          = "myapp"
  email_address = "myapp@example.com"
  quota_max     = 10000000000
}
```

### Multiple Accounts with Different Quotas

```hcl
locals {
  accounts = {
    dev = {
      email = "dev@example.com"
      quota = 5000000000  # 5GB
    }
    staging = {
      email = "staging@example.com"
      quota = 10000000000  # 10GB
    }
    prod = {
      email = "prod@example.com"
      quota = 100000000000  # 100GB
    }
  }
}

resource "scality_account" "environments" {
  for_each = local.accounts

  name          = "${each.key}-account"
  email_address = each.value.email
  quota_max     = each.value.quota
}

output "account_credentials" {
  value = {
    for k, v in scality_account.environments : k => {
      account_id = v.id
      access_key = v.access_key
      secret_key = v.secret_key
    }
  }
  sensitive = true
}
```

### With External Account ID

```hcl
resource "scality_account" "integrated" {
  name                = "external-system-account"
  email_address       = "external@example.com"
  quota_max           = 50000000000
  external_account_id = "ext-sys-12345"
}
```

## Argument Reference

### Required Arguments

- `name` (String, Required, Forces new resource) - Name of the account. Changing this forces a new resource to be created.
- `email_address` (String, Required) - Email address for the account. Must be a valid email format.

### Optional Arguments

- `quota_max` (Number, Optional, Default: 0) - Maximum amount of bytes storable by the account. `0` means unlimited.
- `external_account_id` (String, Optional) - External account ID for integration with other systems.

## Attribute Reference

In addition to the arguments, the following attributes are exported:

- `id` (String) - Scality account ID.
- `arn` (String) - Amazon Resource Name (ARN) of the account.
- `canonical_id` (String) - Canonical ID of the account (used for bucket policies).
- `create_date` (String) - Account creation timestamp in ISO 8601 format.
- `access_key` (String, Sensitive) - S3 API access key (generated automatically). This value is only known after creation and cannot be retrieved later.
- `secret_key` (String, Sensitive) - S3 API secret key (generated automatically). This value is only known after creation and cannot be retrieved later.

## Import

IAM accounts can be imported using the account name:

```bash
terraform import scality_account.example my-account-name
```

**Note**: After import, the `access_key` and `secret_key` attributes will be unknown since they cannot be retrieved from the API. You'll need to generate new keys or maintain existing credentials separately.

## Authentication Details

### AWS Signature Version 4

The provider uses AWS Signature Version 4 authentication for all IAM API requests:

- **Service**: `iam`
- **Region**: `us-east-1`
- **Algorithm**: `AWS4-HMAC-SHA256`
- **Signed Headers**: `host`, `x-amz-content-sha256`, `x-amz-date`

Each request is individually signed with the admin credentials configured in the provider.

### API Endpoints Used

- **CreateAccount**: `POST /` with `Action=CreateAccount&Version=2010-05-08`
- **GenerateAccountAccessKey**: `POST /` with `Action=GenerateAccountAccessKey&Version=2010-05-08`
- **GetAccount**: `POST /` with `Action=GetAccount&Version=2010-05-08` (for state refresh)
- **DeleteAccount**: `POST /` with `Action=DeleteAccount&Version=2010-05-08`

All requests use form-encoded parameters with `Content-Type: application/x-www-form-urlencoded`.

## Key Differences from scality_console_account (Console API)

| Feature | scality_account (IAM) | scality_console_account (Console) |
|---------|----------------------|-----------------------------------|
| **Authentication** | AWS Signature V4 (per-request signing) | JWT Token (cached 23.5hrs) |
| **Account Creation** | Standard process | **Without password** |
| **Credentials Type** | Standard S3 access keys | **Persistent keys** (not password-linked) |
| **Performance** | Per-request signing overhead | Cached token reduces overhead |
| **Deletion Process** | Single-step | Two-step (account + user) |
| **Account Attribute** | `name` | `account_name` |
| **Email Attribute** | `email_address` | `email` |
| **Quota Attribute** | `quota_max` | `quota` |
| **Provider Config** | `endpoint`, `access_key`, `secret_key` | `console_endpoint`, `console_username`, `console_password` |
| **Exported Attributes** | `arn`, `canonical_id`, `create_date` | `created_at` |
| **Use Case** | Traditional IAM-style management | Console UI integration |

## Error Handling

### HTTP 201 - Success
Account created successfully.

### HTTP 200 - Success (Read/Delete)
Account read or deleted successfully.

### HTTP 400 - Bad Request
```
Error: Unable to create account: unexpected status 400: Bad Request
```
**Resolution**: Verify email format is valid (`user@domain.tld`) and all required fields are provided.

### HTTP 404 - Not Found (Read)
Account doesn't exist. Terraform will remove it from state and recreate on next apply if still defined.

### HTTP 409 - Conflict (Create)
```
Error: account already exists
```
**Resolution**: The account name is already in use. Choose a different name or import the existing account.

### HTTP 409 - DeleteConflict (Delete)
```
Error: cannot delete account 'myaccount' - the account contains resources that must be removed first.

The account may contain:
  • IAM users
  • IAM policies
  • S3 buckets (empty or with data)

Required actions before deletion:
  1. Delete all IAM users in the account
  2. Delete all IAM policies in the account
  3. Delete all objects from S3 buckets
  4. Delete all S3 buckets
  5. Retry account deletion
```
**Resolution**: The account has attached resources. Clean up all resources before deletion:

1. List and delete all IAM users
2. List and delete all IAM policies
3. Empty all S3 buckets (delete all objects)
4. Delete all S3 buckets
5. Retry `terraform destroy`

**Warning**: Attempting to delete an account with resources can strand data that cannot be recovered.

### HTTP 500 - Service Failure
```
Error: Unable to create account: unexpected status 500
```
**Resolution**: The Scality IAM service encountered an internal error. Check service logs and health.

## Security Considerations

### 1. Credential Protection in State

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

### 2. Sensitive Output Handling

```hcl
output "access_key" {
  value     = scality_account.example.access_key
  sensitive = true
}

output "secret_key" {
  value     = scality_account.example.secret_key
  sensitive = true
}
```

View sensitive outputs:
```bash
terraform output -json | jq '.access_key.value'
terraform output -json | jq '.secret_key.value'
```

### 3. Admin Credential Protection

Store admin credentials securely:

```bash
# Use environment variables
export SCALITY_ACCESS_KEY="admin-key"
export SCALITY_SECRET_KEY="admin-secret"

# Or use .tfvars with restricted permissions
chmod 600 terraform.tfvars
```

Never hardcode credentials in `.tf` files:
```hcl
# BAD - Don't do this
provider "scality" {
  access_key = "hardcoded-key"  # ❌ Never do this
  secret_key = "hardcoded-secret"
}

# GOOD - Use variables
provider "scality" {
  access_key = var.scality_access_key  # ✅ From variables
  secret_key = var.scality_secret_key
}
```

### 4. Use HTTPS in Production

```hcl
provider "scality" {
  endpoint = "https://scality.example.com"
  # ... other config
}
```

## State Management

### Drift Detection

Terraform will detect if an account is deleted outside of Terraform:
- On `terraform plan` or `terraform apply`, the provider queries the IAM API
- If the account no longer exists (HTTP 404), it's removed from state
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
resource "scality_account" "example" {
  name          = "myapp"
  email_address = "myapp@example.com"
  quota_max     = 10000000000
}
```

On `terraform apply`:
1. Provider signs request with admin credentials (AWS Signature V4)
2. Creates account via IAM API
3. Automatically generates S3 access keys
4. Returns account details and credentials in state

### Updates

Changing the `email_address` or `quota_max` attributes triggers an update:
```hcl
resource "scality_account" "example" {
  name          = "myapp"
  email_address = "newemail@example.com"  # Updated
  quota_max     = 20000000000              # Updated
}
```

**Note**: IAM API update support may be limited. Check API documentation for update capabilities. Some changes may require replacement.

**Changing `name` forces replacement** (new account created, old account destroyed):
```hcl
resource "scality_account" "example" {
  name          = "myapp-v2"  # Changed - will force replacement
  email_address = "myapp@example.com"
  quota_max     = 10000000000
}
```

### Deletion

On `terraform destroy`:
1. Provider signs delete request with admin credentials
2. Deletes account via IAM API (single-step process)
3. Removes resource from state

**Important**: Accounts with buckets, users, or policies will fail deletion (HTTP 409). Clean up resources first.

## Troubleshooting

### Provider Configuration Issues

**Error**: `Missing IAM Client Configuration`

**Solution**: Ensure IAM API credentials are configured:
```hcl
provider "scality" {
  endpoint   = "http://10.164.169.247"
  access_key = var.scality_access_key
  secret_key = var.scality_secret_key
}
```

Or via environment variables:
```bash
export SCALITY_ENDPOINT="http://10.164.169.247"
export SCALITY_ACCESS_KEY="your-access-key"
export SCALITY_SECRET_KEY="your-secret-key"
```

### Authentication Failures

**Error**: `failed to sign request` or HTTP 403

**Checklist**:
1. Verify IAM API is accessible: `curl http://10.164.169.247`
2. Check access key and secret key are correct
3. Ensure credentials have IAM admin permissions
4. Verify endpoint URL is correct (no trailing slash)

### Account Already Exists

**Error**: `account already exists`

**Options**:
1. **Import existing account**:
   ```bash
   terraform import scality_account.example existing-account-name
   ```

2. **Use different name**:
   ```hcl
   name = "myapp-v2"  # Changed
   ```

### Deletion Failures

**Error**: `cannot delete account - the account contains resources`

**Resolution**: The account has attached resources. Clean up manually:

```bash
# List buckets (use AWS CLI or similar)
aws s3 ls --profile ACCOUNT_NAME --endpoint-url http://10.164.169.247

# Delete objects from buckets
aws s3 rm s3://BUCKET_NAME --recursive --profile ACCOUNT_NAME

# Delete buckets
aws s3 rb s3://BUCKET_NAME --profile ACCOUNT_NAME

# List IAM users (if IAM is available)
# Delete IAM users and policies

# Retry deletion
terraform destroy
```

### Enable Debug Logging

```bash
# Set Terraform log level
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform-debug.log

# Run terraform
terraform apply

# Review logs
cat terraform-debug.log | grep -i "scality"
```

## See Also

- [scality_console_account](scality_console_account.md) - Console API resource for comparison
- [Provider Configuration](provider.md) - Detailed provider configuration
- [Quick Start Guide](QUICKSTART.md) - Get started quickly
- [Examples](EXAMPLES.md) - More examples and patterns
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions

## API Reference

Based on Scality IAM API specifications:
- [CreateAccount API](../../requirements/iam/create_account.rst)
- [GenerateAccountAccessKey API](../../requirements/iam/generate_account_access_key.rst)
- [DeleteAccount API](../../requirements/iam/delete_account.rst)
