# OpenTofu/Terraform Provider for Scality

Manage Scality on-premises S3 storage accounts with OpenTofu/Terraform. Supports both IAM API and Console API authentication methods.

> **Note**: This provider is fully compatible with both [OpenTofu](https://opentofu.org/) (recommended) and Terraform.

## Quick Links

- **[scality_account](docs/scality_account.md)** - IAM API resource for account management
- **[scality_console_account](docs/scality_console_account.md)** - Console API resource for account management
- **[Quick Start Guide](docs/QUICKSTART.md)** - Step-by-step setup instructions

## Features

**Two Resource Types:**
- **scality_account** - IAM API with Signature V4 authentication
- **scality_console_account** - Console API with JWT authentication

**Core Capabilities:**
- Automatic S3 credential generation
- Account quota management
- State management and drift detection
- Import existing accounts
- Secure credential handling

## Requirements

- OpenTofu >= 1.6 or Terraform >= 1.0
- Go >= 1.21 (for building from source)
- Scality on-premises storage system with admin credentials

## Installation

### From Release (Recommended)

Download pre-built binaries from [GitHub Releases](https://github.com/scality/terraform-provider-scality/releases):

```bash
# Download and extract (replace version and platform as needed)
VERSION=v0.2.0
PLATFORM=linux_amd64
wget https://github.com/scality/terraform-provider-scality/releases/download/${VERSION}/terraform-provider-scality_${VERSION}_${PLATFORM}.tar.gz
tar -xzf terraform-provider-scality_${VERSION}_${PLATFORM}.tar.gz

# Install to local plugin directory
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/scality/scality/${VERSION#v}/${PLATFORM}/
mv terraform-provider-scality_${VERSION}_${PLATFORM} ~/.terraform.d/plugins/registry.terraform.io/scality/scality/${VERSION#v}/${PLATFORM}/terraform-provider-scality_${VERSION}
chmod +x ~/.terraform.d/plugins/registry.terraform.io/scality/scality/${VERSION#v}/${PLATFORM}/terraform-provider-scality_${VERSION}
```

### Build from Source

```bash
git clone https://github.com/scality/terraform-provider-scality
cd terraform-provider-scality
make build
make install
```

### Development Setup

For local development with automatic rebuilds:

```bash
# Build and install
make build
make install

# Optional: Create dev override in ~/.tofurc or ~/.terraformrc
provider_installation {
  dev_overrides {
    "scality/scality" = "/home/YOUR_USER/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/"
  }
  direct {}
}
```

## Usage

### Basic Example - IAM API

**Recommended: Use environment variables**:

```bash
export SCALITY_ENDPOINT="http://scality.example.com"
export SCALITY_ACCESS_KEY="your-admin-access-key"
export SCALITY_SECRET_KEY="your-admin-secret-key"
```

```hcl
terraform {
  required_providers {
    scality = {
      source  = "scality/scality"
      version = "~> 0.2"
    }
  }
}

provider "scality" {
  # Credentials loaded from environment variables
}

resource "scality_account" "app" {
  name          = "myapp"
  email_address = "myapp@example.com"
  quota_max     = 1073741824  # 1GB in bytes
}

output "s3_credentials" {
  value = {
    access_key = scality_account.app.access_key
    secret_key = scality_account.app.secret_key
  }
  sensitive = true
}
```

### Console API Example

> **Note**: Console superadmin credentials are created during Scality deployment via Ansible:
> ```bash
> ansible-playbook -i env/s3config/inventory \
>   tooling-playbooks/create-superadmin-console-user.yml \
>   -e ui_username=admin -e ui_password=mySuperPassword
> ```

**Recommended: Use environment variables** (never commit credentials to git):

```bash
# Set credentials from Ansible deployment
export SCALITY_CONSOLE_ENDPOINT="http://scality.example.com:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="mySuperPassword"
```

```hcl
provider "scality" {
  # Credentials loaded from environment variables
}

resource "scality_console_account" "app" {
  account_name             = "myapp"
  email                    = "myapp@example.com"
  quota                    = 1073741824  # 1GB in bytes
  generate_random_password = true        # Optional: for Console UI access
  password_length          = 20          # Optional: default 16, minimum 16
}
```

### Environment Variables (Recommended)

```bash
# IAM API
export SCALITY_ENDPOINT="http://scality.example.com"
export SCALITY_ACCESS_KEY="admin-access-key"
export SCALITY_SECRET_KEY="admin-secret-key"

# Console API
export SCALITY_CONSOLE_ENDPOINT="http://scality.example.com:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="mySuperPassword"
```

> **Security Best Practice**: Always use environment variables or a secure secret management system. Never hardcode credentials in `.tf` files.

## Provider Configuration

> **Recommended**: Use environment variables for all credentials. See examples above.

Configure one or both authentication methods depending on which resources you need.

### IAM API (scality_account)

| Argument | Environment Variable (Recommended) | Description |
|----------|-----------------------------------|-------------|
| `endpoint` | `SCALITY_ENDPOINT` | IAM API endpoint (e.g., `http://scality.example.com`) |
| `access_key` | `SCALITY_ACCESS_KEY` | Admin access key (sensitive) |
| `secret_key` | `SCALITY_SECRET_KEY` | Admin secret key (sensitive) |

### Console API (scality_console_account)

| Argument | Environment Variable (Recommended) | Description |
|----------|-----------------------------------|-------------|
| `console_endpoint` | `SCALITY_CONSOLE_ENDPOINT` | Console API endpoint (e.g., `http://scality.example.com:8080`) |
| `console_username` | `SCALITY_CONSOLE_USERNAME` | Admin username from Ansible deployment (sensitive) |
| `console_password` | `SCALITY_CONSOLE_PASSWORD` | Admin password from Ansible deployment (sensitive) |

> **Console Setup**: Superadmin credentials are created during deployment:
> ```bash
> ansible-playbook -i env/s3config/inventory \
>   tooling-playbooks/create-superadmin-console-user.yml \
>   -e ui_username=admin -e ui_password=mySuperPassword
> ```

## Resources

### scality_account

Manages accounts using the IAM API with Signature V4 authentication.

**Arguments:**
- `name` (Required) - Account name
- `email_address` (Required) - Email address
- `quota_max` (Optional) - Storage quota in bytes (default: unlimited)
- `external_account_id` (Optional) - External ID for integrations

**Exported Attributes:**
- `id`, `arn`, `canonical_id`, `create_date`
- `access_key`, `secret_key` (sensitive, auto-generated)

[Full Documentation →](docs/scality_account.md)

### scality_console_account

Manages accounts using the Console API with JWT authentication.

**Arguments:**
- `account_name` (Required) - Account name
- `email` (Required) - Email address
- `quota` (Optional) - Storage quota in bytes (default: unlimited)
- `generate_random_password` (Optional) - Generate random Console password (default: false)
- `password_length` (Optional) - Password length, minimum 16 (default: 16)

**Exported Attributes:**
- `id`, `created_at`
- `password` (sensitive, computed) - Generated Console password if enabled
- `access_key`, `secret_key` (sensitive, auto-generated)

**Key Features:**
- Optional random password generation for Console UI access
- Persistent S3 credentials (not password-linked)
- JWT token caching (23.5 hours)

[Full Documentation →](docs/scality_console_account.md)

## Examples

### Multiple Accounts with For-Each

```hcl
locals {
  environments = {
    dev     = { email = "dev@example.com", quota = 5368709120 }    # 5GB
    staging = { email = "staging@example.com", quota = 10737418240 } # 10GB
    prod    = { email = "prod@example.com", quota = 107374182400 }  # 100GB
  }
}

resource "scality_account" "env" {
  for_each = local.environments

  name          = "${each.key}-storage"
  email_address = each.value.email
  quota_max     = each.value.quota
}

output "credentials" {
  value = {
    for k, v in scality_account.env : k => {
      access_key = v.access_key
      secret_key = v.secret_key
    }
  }
  sensitive = true
}
```

### Import Existing Account

```bash
# IAM account
terraform import scality_account.existing account-name

# Console account
terraform import scality_console_account.existing account-name
```

## Common Issues

**DeleteConflict (409)**: Account contains resources (buckets, users, policies)
- **Solution**: Delete all resources in the account first, then retry

**Account Already Exists (409)**: Account name is taken
- **Solution**: Import the existing account or use a different name

**Import Syntax**:
```bash
terraform import scality_account.name account-name
terraform import scality_console_account.name account-name
```

## State Management

**Sensitive Data**: Generated S3 credentials are marked sensitive in state files
- Use remote state with encryption (S3, Consul, Terraform Cloud)
- Restrict state file access with proper IAM/permissions
- Never commit state files to version control
- **Use environment variables for provider credentials** (never hardcode in `.tf` files)

**Drift Detection**: Automatically detects accounts deleted outside Terraform

## Technical Details

### IAM API Authentication
- **Method**: Signature Version 4 (compatible with AWS SDKs)
- **Endpoints**: CreateAccount, GenerateAccountAccessKey, GetAccount, DeleteAccount
- **Signing**: AWS4-HMAC-SHA256 with `host`, `x-amz-content-sha256`, `x-amz-date` headers

### Console API Authentication
- **Method**: JWT tokens (24-hour lifetime)
- **Token Caching**: Automatic caching for 23.5 hours in `/tmp` (0600 permissions)
- **Endpoints**: Authenticate, CreateAccount, GenerateAccessKey, GetAccount, DeleteAccount
- **Deletion**: Two-step process (account + user)

## Limitations

- Credentials cannot be rotated (generate new account for rotation)
- Account name changes force resource replacement
- Some account updates may require replacement

## Development

```bash
# Run tests
make test

# Run acceptance tests (requires Scality instance)
make testacc

# Format code
make fmt

# Build locally
make build
make install
```

## Release Process

GitHub Actions automatically builds and releases binaries for all platforms when you push a version tag:

```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

**Supported Platforms**: Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)

**Verify Downloads**:
```bash
wget https://github.com/scality/terraform-provider-scality/releases/download/v0.2.0/terraform-provider-scality_v0.2.0_SHA256SUMS
sha256sum -c terraform-provider-scality_v0.2.0_SHA256SUMS --ignore-missing
```

## Contributing

Contributions welcome! Please:
- Add tests for new features
- Run `make fmt` before committing
- Ensure `make test` passes

## License

Apache 2.0 - See LICENSE file

## Version History

**v0.2.0** - Console API support, JWT authentication, token caching
**v0.1.0** - Initial release with IAM API support
