# Scality Terraform/OpenTofu Provider Documentation

Terraform/OpenTofu provider for managing Scality accounts. Supports both IAM-style API (AWS Signature V4) and Console API (JWT authentication).

> **Recommended**: Use [OpenTofu](https://opentofu.org/) instead of Terraform for open-source licensing benefits.

## Available Resources

### IAM API Resources (AWS Signature V4 Authentication)

- [scality_account](scality_account.md) - Manage Scality accounts via IAM API
  - Create accounts with S3 credentials
  - Manage quotas and metadata
  - Delete accounts with comprehensive error handling
  - Automatic credential generation

### Console API Resources (JWT Authentication)

- [scality_console_account](scality_console_account.md) - Manage Scality accounts via Console API
  - Create accounts without passwords (security best practice)
  - Generate persistent S3 access keys
  - JWT token authentication with caching (23.5hr)
  - Two-step account deletion

## Quick Comparison

| Feature | scality_account (IAM) | scality_console_account (Console) |
|---------|----------------------|-----------------------------------|
| Authentication | AWS Signature V4 | JWT Token |
| Account Creation | Standard | Without password |
| Credentials | S3 access keys | Persistent S3 access keys |
| Token Caching | N/A | 23.5 hours |
| Deletion | Single step | Two-step (account + user) |
| Attribute: Name | `name` | `account_name` |
| Attribute: Email | `email_address` | `email` |
| Attribute: Quota | `quota_max` | `quota` |
| Use Case | IAM-style management | Console UI integration |

## Requirements

- [OpenTofu](https://opentofu.org/docs/intro/install/) >= 1.6 or [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for building from source)
- Access to a Scality storage system
- Admin credentials (IAM or Console)

## Installation

See the [Quick Start Guide](QUICKSTART.md) for detailed installation instructions.

### Quick Install

```bash
cd /home/cyrus/Documents/Development/Scality/Terraform/Providers/terraform-provider-scality
make build
make install
```

## Quick Start

### Using IAM API

```hcl
provider "scality" {
  endpoint   = "http://10.164.169.247"
  access_key = var.scality_access_key
  secret_key = var.scality_secret_key
}

resource "scality_account" "example" {
  name          = "myapp"
  email_address = "myapp@example.com"
  quota_max     = 1000000000
}

output "access_key" {
  value     = scality_account.example.access_key
  sensitive = true
}
```

### Using Console API

```hcl
provider "scality" {
  console_endpoint = "http://10.164.169.247:8080"
  console_username = var.console_username
  console_password = var.console_password
}

resource "scality_console_account" "example" {
  account_name = "myapp"
  email        = "myapp@example.com"
  quota        = 1000000000
}

output "access_key" {
  value     = scality_console_account.example.access_key
  sensitive = true
}
```

## Documentation

### Getting Started
- [Quick Start Guide](QUICKSTART.md) - Get started in minutes with step-by-step instructions
- [Provider Configuration](provider.md) - Configure the provider for IAM and Console APIs

### Resource Documentation
- [scality_account](scality_account.md) - IAM API resource (AWS Signature V4)
  - Create/delete accounts
  - Generate S3 credentials
  - Manage quotas and metadata
  - Import existing accounts

- [scality_console_account](scality_console_account.md) - Console API resource (JWT)
  - Password-free account creation
  - Persistent S3 keys
  - Token caching
  - Two-step deletion

### Additional Resources
- [Examples](EXAMPLES.md) - Common patterns and advanced use cases
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions

## Provider Configuration

The provider supports two authentication methods. You can configure one or both.

### IAM API Configuration

```hcl
provider "scality" {
  endpoint   = "http://10.164.169.247"      # or SCALITY_ENDPOINT
  access_key = var.scality_access_key        # or SCALITY_ACCESS_KEY
  secret_key = var.scality_secret_key        # or SCALITY_SECRET_KEY
}
```

### Console API Configuration

```hcl
provider "scality" {
  console_endpoint = "http://10.164.169.247:8080"  # or SCALITY_CONSOLE_ENDPOINT
  console_username = var.console_username           # or SCALITY_CONSOLE_USERNAME
  console_password = var.console_password           # or SCALITY_CONSOLE_PASSWORD
}
```

### Using Both APIs

```hcl
provider "scality" {
  # IAM API credentials
  endpoint   = "http://10.164.169.247"
  access_key = var.scality_access_key
  secret_key = var.scality_secret_key

  # Console API credentials
  console_endpoint = "http://10.164.169.247:8080"
  console_username = var.console_username
  console_password = var.console_password
}

# Use IAM API
resource "scality_account" "iam_managed" {
  name          = "iam-account"
  email_address = "iam@example.com"
  quota_max     = 10000000000
}

# Use Console API
resource "scality_console_account" "console_managed" {
  account_name = "console-account"
  email        = "console@example.com"
  quota        = 10000000000
}
```

## Security Best Practices

1. **Use Environment Variables for Credentials**
   ```bash
   export SCALITY_ENDPOINT="http://10.164.169.247"
   export SCALITY_ACCESS_KEY="your-access-key"
   export SCALITY_SECRET_KEY="your-secret-key"

   export SCALITY_CONSOLE_ENDPOINT="http://10.164.169.247:8080"
   export SCALITY_CONSOLE_USERNAME="admin"
   export SCALITY_CONSOLE_PASSWORD="admin-password"
   ```

2. **Never Commit Credentials**
   ```bash
   echo "*.tfvars" >> .gitignore
   echo "terraform.tfstate*" >> .gitignore
   ```

3. **Use Remote State with Encryption**
   ```hcl
   terraform {
     backend "s3" {
       bucket         = "terraform-state"
       key            = "scality/terraform.tfstate"
       encrypt        = true
     }
   }
   ```

4. **Mark Outputs as Sensitive**
   ```hcl
   output "access_key" {
     value     = scality_account.example.access_key
     sensitive = true
   }
   ```

5. **Use HTTPS in Production**
   ```hcl
   provider "scality" {
     endpoint  = "https://scality.example.com"
     # ... other config
   }
   ```

6. **Console API: Password-Free Security**
   - Console API creates accounts WITHOUT passwords by design
   - Generates persistent keys not tied to passwords
   - More secure than password-linked credentials

## Common Patterns

### Pattern 1: Multiple Environments

```hcl
locals {
  accounts = {
    dev = {
      email = "dev@example.com"
      quota = 5000000000
    }
    staging = {
      email = "staging@example.com"
      quota = 10000000000
    }
    prod = {
      email = "prod@example.com"
      quota = 100000000000
    }
  }
}

resource "scality_account" "environments" {
  for_each = local.accounts

  name          = "${each.key}-account"
  email_address = each.value.email
  quota_max     = each.value.quota
}
```

### Pattern 2: Save Credentials to AWS Secrets Manager

```hcl
resource "scality_console_account" "app" {
  account_name = "myapp"
  email        = "app@example.com"
  quota        = 50000000000
}

resource "aws_secretsmanager_secret" "scality_creds" {
  name = "scality/${scality_console_account.app.account_name}/credentials"
}

resource "aws_secretsmanager_secret_version" "scality_creds" {
  secret_id = aws_secretsmanager_secret.scality_creds.id
  secret_string = jsonencode({
    access_key = scality_console_account.app.access_key
    secret_key = scality_console_account.app.secret_key
  })
}
```

### Pattern 3: Conditional Resources

```hcl
variable "use_console_api" {
  type    = bool
  default = false
}

resource "scality_account" "iam" {
  count = var.use_console_api ? 0 : 1

  name          = "myapp"
  email_address = "myapp@example.com"
  quota_max     = 1000000000
}

resource "scality_console_account" "console" {
  count = var.use_console_api ? 1 : 0

  account_name = "myapp"
  email        = "myapp@example.com"
  quota        = 1000000000
}
```

## Support

For issues or questions:
- Review the [documentation](.)
- Check the [Quick Start Guide](QUICKSTART.md)
- Review [Examples](EXAMPLES.md)
- Check [Troubleshooting](TROUBLESHOOTING.md)
- Verify API connectivity and credentials
- Report bugs via GitHub issues

## Contributing

Contributions are welcome! Please ensure:
- Code follows existing patterns
- Tests are included
- Documentation is updated
- All tests pass

## License

This provider is provided as-is for use with Scality storage systems.

## Version History

### 0.2.0 (Current)
- Added Console API support with JWT authentication
- New `scality_console_account` resource
- Token caching for performance (23.5hr cache)
- Dual-client architecture supporting both IAM and Console APIs
- Security-first approach (accounts without passwords)
- Comprehensive documentation split into multiple files

### 0.1.0
- Initial release with `scality_account` resource
- AWS Signature V4 authentication
- Automatic access key generation
- Comprehensive error handling
