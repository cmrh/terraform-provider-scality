# scality_account_access_key - Manage Access Keys for Scality Accounts

Manages additional S3 API access keys for Scality accounts. Useful for implementing key rotation strategies and managing multiple credentials for the same account.

## Example Usage

### Basic Key Rotation

Generate an additional access key for an existing account:

```hcl
resource "scality_account" "app" {
  name          = "my-app"
  email_address = "myapp@example.com"
  quota_max     = 50000000000
}

# Generate additional access key for rotation
resource "scality_account_access_key" "rotation_key" {
  account_name = scality_account.app.name
}

output "primary_credentials" {
  value = {
    access_key = scality_account.app.access_key
    secret_key = scality_account.app.secret_key
  }
  sensitive = true
}

output "rotation_credentials" {
  value = {
    access_key = scality_account_access_key.rotation_key.access_key
    secret_key = scality_account_access_key.rotation_key.secret_key
  }
  sensitive = true
}
```

### Zero-Downtime Key Rotation

Implement zero-downtime key rotation by creating a new key before deleting the old one:

```hcl
resource "scality_account" "service" {
  name          = "backend-service"
  email_address = "backend@example.com"
  quota_max     = 100000000000
}

# Current active key
resource "scality_account_access_key" "current" {
  account_name = scality_account.service.name
}

# Uncomment when rotating:
# resource "scality_account_access_key" "new" {
#   account_name = scality_account.service.name
# }

output "active_key" {
  value = {
    access_key = scality_account_access_key.current.access_key
    secret_key = scality_account_access_key.current.secret_key
  }
  sensitive = true
}
```

**Rotation Process**:
1. Uncomment the `new` key resource and apply
2. Update application configuration to use the new credentials
3. Verify application is working with new credentials
4. Comment out or remove the `current` key resource and apply
5. The old key is deleted after the new key is in use

### Multiple Keys for Different Environments

Create separate keys for different deployment environments:

```hcl
resource "scality_account" "shared" {
  name          = "shared-account"
  email_address = "shared@example.com"
  quota_max     = 200000000000
}

# Development environment key
resource "scality_account_access_key" "dev" {
  account_name = scality_account.shared.name
}

# Staging environment key
resource "scality_account_access_key" "staging" {
  account_name = scality_account.shared.name
}

# Production environment key
resource "scality_account_access_key" "prod" {
  account_name = scality_account.shared.name
}

output "environment_keys" {
  value = {
    dev = {
      access_key = scality_account_access_key.dev.access_key
      secret_key = scality_account_access_key.dev.secret_key
    }
    staging = {
      access_key = scality_account_access_key.staging.access_key
      secret_key = scality_account_access_key.staging.secret_key
    }
    prod = {
      access_key = scality_account_access_key.prod.access_key
      secret_key = scality_account_access_key.prod.secret_key
    }
  }
  sensitive = true
}
```

### Time-Based Rotation with Lifecycle

Automate key rotation using Terraform lifecycle rules:

```hcl
resource "scality_account" "app" {
  name          = "rotation-app"
  email_address = "app@example.com"
  quota_max     = 50000000000
}

# Primary key with timestamp
resource "scality_account_access_key" "primary" {
  account_name = scality_account.app.name

  lifecycle {
    create_before_destroy = true
  }
}

# Use timestamp to force rotation
# Change the timestamp value to trigger rotation
locals {
  rotation_timestamp = "2026-03-01"  # Update this to rotate
}

resource "scality_account_access_key" "rotated" {
  count = local.rotation_timestamp != "" ? 1 : 0

  account_name = scality_account.app.name

  lifecycle {
    create_before_destroy = true
  }
}
```

## Argument Reference

### Required Arguments

- `account_name` (String, Required, Forces new resource) - Name of the account this key belongs to. Changing this forces a new resource to be created.

## Attribute Reference

In addition to the arguments, the following attributes are exported:

- `id` (String) - Access key ID (same as `access_key`).
- `access_key` (String, Sensitive) - S3 API access key. This value is only known after creation and cannot be retrieved later.
- `secret_key` (String, Sensitive) - S3 API secret key. This value is only known after creation and cannot be retrieved later.
- `status` (String) - Status of the access key (typically "Active").
- `create_date` (String) - Key creation timestamp in ISO 8601 format.

## Import

Access keys can be imported using the access key ID:

```bash
terraform import scality_account_access_key.example <access-key-id>
```

**Note**: After import, the `secret_key` attribute will be unknown since it cannot be retrieved from the API. Import is primarily useful for managing the lifecycle of existing keys, but you won't have access to the secret key value.

## Authentication Details

### AWS Signature Version 4

The provider uses AWS Signature Version 4 authentication for all IAM API requests. Each request is individually signed with the admin credentials configured in the provider.

### API Endpoints Used

- **GenerateAccountAccessKey**: `POST /` with `Action=GenerateAccountAccessKey&Version=2010-05-08`
- **DeleteAccessKey**: `POST /` with `Action=DeleteAccessKey&Version=2010-05-08`

All requests use form-encoded parameters with `Content-Type: application/x-www-form-urlencoded`.

## Key Rotation Strategies

### Strategy 1: Blue-Green Rotation

Create a new key before deleting the old one:

```hcl
# Step 1: Create new key (green)
resource "scality_account_access_key" "green" {
  account_name = scality_account.app.name
}

# Step 2: Update application to use green credentials
# Step 3: Delete blue key after verification
# resource "scality_account_access_key" "blue" {
#   account_name = scality_account.app.name
# }
```

### Strategy 2: Scheduled Rotation

Use a consistent naming pattern with timestamps:

```hcl
locals {
  current_rotation_cycle = "2026-q1"  # Update quarterly
}

resource "scality_account_access_key" "current" {
  account_name = scality_account.app.name

  # Tag with rotation cycle in resource name
  # Example: scality_account_access_key.rotation_2026_q1
}
```

### Strategy 3: Application-Specific Keys

Create separate keys for different applications or services:

```hcl
resource "scality_account" "shared_storage" {
  name          = "shared-storage"
  email_address = "storage@example.com"
  quota_max     = 500000000000
}

resource "scality_account_access_key" "web_app" {
  account_name = scality_account.shared_storage.name
}

resource "scality_account_access_key" "mobile_app" {
  account_name = scality_account.shared_storage.name
}

resource "scality_account_access_key" "batch_processor" {
  account_name = scality_account.shared_storage.name
}
```

Benefits:
- Revoke access for specific applications without affecting others
- Track usage by application
- Implement least-privilege access patterns

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

Always:
- Use remote state with encryption
- Restrict access to state files
- Never commit state files to version control
- Use sensitive outputs

### 2. Sensitive Output Handling

```hcl
output "access_key" {
  value     = scality_account_access_key.rotation_key.access_key
  sensitive = true
}

output "secret_key" {
  value     = scality_account_access_key.rotation_key.secret_key
  sensitive = true
}
```

View sensitive outputs:
```bash
terraform output -json | jq '.access_key.value'
terraform output -json | jq '.secret_key.value'
```

### 3. Rotation Best Practices

- **Regular Rotation**: Rotate keys on a regular schedule (e.g., every 90 days)
- **Incident Response**: Rotate immediately if credentials are compromised
- **Zero-Downtime**: Always create new keys before deleting old ones
- **Verification**: Test new credentials thoroughly before revoking old ones
- **Audit Trail**: Track key creation and deletion in your version control

### 4. Key Lifecycle Management

```hcl
resource "scality_account_access_key" "managed" {
  account_name = scality_account.app.name

  lifecycle {
    # Prevent accidental deletion
    prevent_destroy = true

    # Create new key before destroying old one
    create_before_destroy = true
  }
}
```

## Limitations

### Updates Not Supported

Access keys cannot be updated - they are immutable. To rotate a key:
1. Create a new `scality_account_access_key` resource
2. Update application configuration to use the new key
3. Delete the old `scality_account_access_key` resource

Attempting to update an access key will result in a warning.

### Secret Key Retrieval

The `secret_key` is only available in the API response during creation. It cannot be retrieved later:
- Store credentials securely immediately after creation
- Use Terraform outputs to extract credentials
- If lost, generate a new key

### Read Operations

The API doesn't provide a way to retrieve secret keys or list keys after creation. The Read operation preserves the state without making API calls. This means:
- Drift detection is limited for access keys
- Keys deleted outside Terraform may not be detected until the next apply

## Error Handling

### HTTP 201 - Success
Access key generated successfully.

### HTTP 400 - Bad Request
```
Error: Unable to generate access key: unexpected status 400
```
**Resolution**: Verify the account name is correct and the account exists.

### HTTP 404 - Not Found
```
Error: Unable to generate access key: account not found
```
**Resolution**: The specified account doesn't exist. Create the account first or verify the account name.

### HTTP 500 - Service Failure
```
Error: Unable to generate access key: unexpected status 500
```
**Resolution**: The Scality IAM service encountered an internal error. Check service logs and health.

## Troubleshooting

### Provider Configuration Issues

**Error**: `Missing IAM Client Configuration`

**Solution**: Ensure IAM API credentials are configured:
```bash
export SCALITY_ENDPOINT="http://10.164.169.247"
export SCALITY_ACCESS_KEY="your-access-key"
export SCALITY_SECRET_KEY="your-secret-key"
```

### Account Not Found

**Error**: `Unable to generate access key`

**Checklist**:
1. Verify the account exists: Check `scality_account` resource
2. Ensure account creation completed successfully
3. Verify account name spelling is correct

### Deletion Failures

**Error**: Access key deletion returns 403 or fails

**Note**: The current implementation attempts to delete access keys, but the API may not support this operation in all configurations. If deletion fails:
- The provider treats 404 responses as successful (key already deleted)
- Other errors are reported but may not prevent resource removal from state

## State Management

### Credential Limitations

**Important**: The `access_key` and `secret_key` attributes cannot be retrieved after creation:
- These values are only available in the create response
- Subsequent state refreshes don't update these values
- If state is lost, you'll need to generate new credentials

### State File Security

Protect your state files:
```bash
# Add to .gitignore
echo "terraform.tfstate*" >> .gitignore
echo "*.tfvars" >> .gitignore
echo ".terraform/" >> .gitignore
```

## See Also

- [scality_account](scality_account.md) - Parent account resource
- [Provider Configuration](provider.md) - Detailed provider configuration
- [AWS Access Key Rotation Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html#Using_RotateAccessKey)

## API Reference

Based on Scality IAM API specifications:
- [GenerateAccountAccessKey API](../../requirements/iam/generate_account_access_key.rst)
- [DeleteAccessKey API](../../requirements/iam/delete_access_key.rst)
