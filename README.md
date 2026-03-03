# OpenTofu/Terraform Provider for Scality

OpenTofu/Terraform provider for managing Scality accounts. Supports both IAM-style API (AWS Signature Version 4) and Console API (JWT authentication).

> **Note**: This provider is fully compatible with both [OpenTofu](https://opentofu.org/) (recommended) and Terraform. We recommend using OpenTofu due to its open-source licensing.

## Documentation

Comprehensive documentation has been split into separate files for readability:

### Getting Started
- **[Quick Start Guide](docs/QUICKSTART.md)** - Get started in minutes with step-by-step instructions
- **[Main Documentation Hub](docs/README.md)** - Overview of all resources and features

### Resource Documentation
- **[scality_account](docs/scality_account.md)** - IAM API resource (AWS Signature V4)
  - Create/delete accounts
  - Generate S3 credentials
  - Manage quotas and metadata
  - Import existing accounts

- **[scality_console_account](docs/scality_console_account.md)** - Console API resource (JWT)
  - Password-free account creation
  - Persistent S3 keys
  - Token caching
  - Two-step deletion

### Additional Resources
- **[Examples](docs/EXAMPLES.md)** - Common patterns and advanced use cases
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions

## Features

### IAM API (scality_account)
- ✅ Create Scality accounts with automatic S3 API credential generation
- ✅ Manage account quotas and metadata
- ✅ Delete accounts with comprehensive error handling
- ✅ AWS Signature Version 4 authentication

### Console API (scality_console_account)
- ✅ Create accounts without passwords (security best practice)
- ✅ Generate persistent S3 access keys
- ✅ JWT token authentication with automatic caching (23.5hr)
- ✅ Two-step account deletion process

### Common Features
- ✅ State management and drift detection
- ✅ Import existing accounts
- ✅ Sensitive credential handling in state

## Requirements

- [OpenTofu](https://opentofu.org/docs/intro/install/) >= 1.6 or [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for building from source)
- Access to a Scality storage system
- Admin credentials for the Scality IAM API

## Installation Options

### Option 1: Use Pre-Built Binaries (Recommended for Testing)

**No build tools required!** Download pre-built binaries from our Gitea CI/CD system.

See **[TESTING_FROM_ARTIFACTS.md](TESTING_FROM_ARTIFACTS.md)** for complete instructions on:
- Downloading provider binaries from Gitea releases
- Installing without local builds
- Testing the latest development versions
- Automated installation scripts

**Quick Install:**
```bash
# Download the latest release for your platform
wget https://gitea.example.com/scality/terraform-provider-scality/releases/download/v0.1.0/terraform-provider-scality_v0.1.0_linux_amd64.tar.gz

# Extract and install
tar -xzf terraform-provider-scality_v0.1.0_linux_amd64.tar.gz
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/
mv terraform-provider-scality_v0.1.0_linux_amd64 ~/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/terraform-provider-scality_v0.1.0
chmod +x ~/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/terraform-provider-scality_v0.1.0
```

### Option 2: Build from Source

If you need to build locally or contribute to development:

```bash
git clone https://github.com/scality/terraform-provider-scality
cd terraform-provider-scality
go build -o terraform-provider-scality
```

## Local Development Installation

1. Build the provider:
   ```bash
   go build -o terraform-provider-scality
   ```

2. Create the local plugin directory:
   ```bash
   mkdir -p ~/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/
   ```

3. Copy the built provider:
   ```bash
   cp terraform-provider-scality ~/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/
   ```

4. Create a development override file:

   **For OpenTofu** (`~/.tofurc`):
   ```hcl
   provider_installation {
     dev_overrides {
       "scality/scality" = "/home/YOUR_USER/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/"
     }
     direct {}
   }
   ```

   **For Terraform** (`~/.terraformrc`): Same configuration as above

## Usage

### Basic Example

```hcl
terraform {
  required_providers {
    scality = {
      source  = "scality/scality"
      version = "~> 0.1"
    }
  }
}

provider "scality" {
  endpoint   = "http://10.164.169.247"
  access_key = "your-admin-access-key"
  secret_key = "your-admin-secret-key"
}

resource "scality_account" "example" {
  name          = "myapp"
  email_address = "myapp@example.com"
  quota_max     = 1000000000  # 1GB
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
}
```

## Provider Configuration

The provider supports two authentication methods. You can configure one or both depending on which resources you need.

### IAM API Configuration (for scality_account)

- `endpoint` (Optional) - Scality IAM API endpoint URL (e.g., `http://10.164.169.247`). Can also be set via `SCALITY_ENDPOINT` environment variable.
- `access_key` (Optional, Sensitive) - Admin access key for IAM authentication. Can also be set via `SCALITY_ACCESS_KEY` environment variable.
- `secret_key` (Optional, Sensitive) - Admin secret key for IAM authentication. Can also be set via `SCALITY_SECRET_KEY` environment variable.

### Console API Configuration (for scality_console_account)

- `console_endpoint` (Optional) - Scality Console API endpoint URL (e.g., `http://10.164.169.247:8080`). Can also be set via `SCALITY_CONSOLE_ENDPOINT` environment variable.
- `console_username` (Optional, Sensitive) - Console username for JWT authentication. Can also be set via `SCALITY_CONSOLE_USERNAME` environment variable.
- `console_password` (Optional, Sensitive) - Console password for JWT authentication. Can also be set via `SCALITY_CONSOLE_PASSWORD` environment variable.

**Note**: At least one complete set of credentials (either IAM or Console) must be configured.

## Resources

### scality_account

Manages a Scality account with automatically generated S3 API credentials.

#### Example Usage

```hcl
resource "scality_account" "production" {
  name               = "production-app"
  email_address      = "prod@example.com"
  quota_max          = 100000000000  # 100GB
  external_account_id = "prod-001"
}
```

#### Argument Reference

- `name` (Required, ForceNew) - Name of the account. Changing this forces a new resource.
- `email_address` (Required) - Email address for the account. Must be a valid email format.
- `quota_max` (Optional) - Maximum amount of bytes storable by the account. Default: `0` (unlimited).
- `external_account_id` (Optional) - External account ID for integration with other systems.

#### Attribute Reference

In addition to the arguments, the following attributes are exported:

- `id` - Account ID
- `arn` - Amazon Resource Name (ARN) of the account
- `canonical_id` - Canonical ID of the account
- `create_date` - Account creation timestamp
- `access_key` (Sensitive) - S3 API access key (generated automatically)
- `secret_key` (Sensitive) - S3 API secret key (generated automatically)

### scality_console_account

Manages a Scality account via Console API with JWT authentication. Accounts are created **without passwords** (security best practice) and persistent S3 credentials are generated automatically.

#### Example Usage

```hcl
provider "scality" {
  console_endpoint = "http://10.164.169.247:8080"
  console_username = "admin"
  console_password = "admin-password"
}

resource "scality_console_account" "app" {
  account_name = "my-console-app"
  email        = "app@example.com"
  quota        = 50000000000  # 50GB
}

output "console_access_key" {
  value     = scality_console_account.app.access_key
  sensitive = true
}

output "console_secret_key" {
  value     = scality_console_account.app.secret_key
  sensitive = true
}
```

#### Argument Reference

- `account_name` (Required, ForceNew) - Name of the account. Changing this forces a new resource.
- `email` (Required) - Email address for the account. Must be a valid email format.
- `quota` (Optional) - Maximum amount of bytes storable by the account. Default: `0` (unlimited).

#### Attribute Reference

In addition to the arguments, the following attributes are exported:

- `id` - Account identifier (same as account_name)
- `created_at` - Account creation timestamp
- `access_key` (Sensitive) - S3 API access key (persistent credentials, generated automatically)
- `secret_key` (Sensitive) - S3 API secret key (persistent credentials, generated automatically)

#### Key Differences from scality_account

1. **Authentication**: Uses JWT tokens instead of AWS Signature V4
2. **Security**: Accounts created without passwords by design
3. **Credentials**: Generates persistent keys (not password-linked)
4. **Token Caching**: JWT tokens cached for 23.5 hours for performance
5. **Deletion**: Two-step process (account + user)

## Advanced Examples

### Using Both IAM and Console APIs

```hcl
provider "scality" {
  # IAM API credentials
  endpoint   = "http://10.164.169.247"
  access_key = var.iam_access_key
  secret_key = var.iam_secret_key

  # Console API credentials
  console_endpoint = "http://10.164.169.247:8080"
  console_username = var.console_username
  console_password = var.console_password
}

# Create account via IAM API
resource "scality_account" "iam_account" {
  name          = "iam-managed"
  email_address = "iam@example.com"
  quota_max     = 10000000000
}

# Create account via Console API
resource "scality_console_account" "console_account" {
  account_name = "console-managed"
  email        = "console@example.com"
  quota        = 10000000000
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

resource "scality_account" "accounts" {
  for_each = local.accounts

  name          = "${each.key}-account"
  email_address = each.value.email
  quota_max     = each.value.quota
}

output "account_credentials" {
  value = {
    for k, v in scality_account.accounts : k => {
      access_key = v.access_key
      secret_key = v.secret_key
    }
  }
  sensitive = true
}
```

### Saving Credentials to AWS Secrets Manager

```hcl
resource "scality_account" "app" {
  name          = "myapp"
  email_address = "app@example.com"
}

resource "aws_secretsmanager_secret" "scality_credentials" {
  name = "scality/${scality_account.app.name}/credentials"
}

resource "aws_secretsmanager_secret_version" "scality_credentials" {
  secret_id = aws_secretsmanager_secret.scality_credentials.id
  secret_string = jsonencode({
    access_key = scality_account.app.access_key
    secret_key = scality_account.app.secret_key
    account_id = scality_account.app.id
  })
}
```

### Import Existing Account

```bash
terraform import scality_account.existing account-name
```

## Error Handling

### DeleteConflict (HTTP 409)

When attempting to delete an account that still contains resources:

```
Error: Cannot delete account 'myaccount' - the account contains resources that must be removed first.

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

**Resolution**: Clean up all resources in the account before destroying it in Terraform.

### Account Already Exists (HTTP 409 on Create)

If you try to create an account that already exists:

```
Error: account already exists
```

**Resolution**: Import the existing account or use a different name.

## State Management

### Sensitive Data

The provider marks the following as sensitive in the Terraform state:
- `access_key`
- `secret_key`
- Provider configuration `access_key` and `secret_key`

**Important**: Even though these are marked sensitive, the Terraform state file itself should be protected:
- Use remote state with encryption (S3, Terraform Cloud, etc.)
- Restrict access to state files
- Never commit state files to version control

### Drift Detection

Terraform will detect if an account is deleted outside of Terraform and will remove it from state on the next refresh.

## Authentication

The provider supports two authentication methods:

### IAM API - AWS Signature Version 4

Used by `scality_account` resource:

- Service: `iam`
- Region: `us-east-1`
- Algorithm: `AWS4-HMAC-SHA256`
- Signed headers: `host`, `x-amz-content-sha256`, `x-amz-date`

API calls made:
- **CreateAccount**: `POST /` with `Action=CreateAccount&Version=2010-05-08`
- **GenerateAccountAccessKey**: `POST /` with `Action=GenerateAccountAccessKey&Version=2010-05-08`
- **GetAccount**: `POST /` with `Action=GetAccount&Version=2010-05-08` (for drift detection)
- **DeleteAccount**: `POST /` with `Action=DeleteAccount&Version=2010-05-08`

### Console API - JWT Token Authentication

Used by `scality_console_account` resource:

- **Authentication**: `POST /_/console/authenticate` - Returns JWT token (24hr lifetime)
- **Token Caching**: Tokens cached in `/tmp/.scality_console_token_<hash>` for 23.5 hours
- **Cache Security**: Cache files have 0600 permissions (owner-only access)
- **Authorization**: All requests include `x-access-token` header with JWT

API calls made:
- **Authenticate**: `POST /_/console/authenticate` (cached for 23.5hrs)
- **CreateAccount**: `POST /_/console/vault/accounts` (creates without password)
- **GenerateAccessKey**: `POST /_/console/vault/accounts/{name}/keys` (persistent credentials)
- **GetAccount**: `GET /_/console/vault/accounts/{name}` (for drift detection)
- **DeleteAccount**: Two-step process:
  1. `DELETE /_/console/vault/accounts/{name}` (delete account)
  2. `DELETE /_/console/vault/accounts/{name}/user` (delete user)

## Limitations

- Account updates may require replacement (check API documentation)
- Credentials are generated once during creation and cannot be rotated via this provider
- Maximum 10 custom attributes per account (if using account attributes)

## Development

### Running Tests

```bash
go test ./...
```

### Building Documentation

```bash
go generate ./...
```

### Running Acceptance Tests

```bash
TF_ACC=1 go test ./... -v -count 1 -timeout 120m
```

## CI/CD and Automated Builds

This provider uses Gitea Actions for continuous integration and delivery. Every commit triggers automated builds for multiple platforms.

### Build Pipeline

The CI/CD pipeline (`.gitea/workflows/build.yml`) includes:

1. **Testing Phase**
   - Runs unit tests across the codebase
   - Executes `go vet` for code quality
   - Validates code formatting with `gofmt`

2. **Build Phase**
   - Builds provider binaries for multiple platforms:
     - Linux (amd64, arm64)
     - macOS/Darwin (amd64, arm64)
     - Windows (amd64)
   - Embeds version information in binaries
   - Creates compressed archives (`.tar.gz` for Unix, `.zip` for Windows)
   - Uploads artifacts with 30-day retention

3. **Release Phase** (on version tags)
   - Creates GitHub/Gitea releases
   - Attaches all platform binaries
   - Generates SHA256 checksums for verification
   - Produces provider manifest for registry

### Accessing Build Artifacts

**From Gitea Actions:**
1. Navigate to the [Actions tab](https://gitea.example.com/scality/terraform-provider-scality/actions)
2. Click on a successful workflow run
3. Download artifacts for your platform from the "Artifacts" section

**From Releases:**
1. Navigate to [Releases](https://gitea.example.com/scality/terraform-provider-scality/releases)
2. Download the binary matching your platform
3. Follow the [TESTING_FROM_ARTIFACTS.md](TESTING_FROM_ARTIFACTS.md) guide

### Supported Platforms

| Platform | Architecture | File Extension |
|----------|-------------|----------------|
| Linux    | amd64       | `.tar.gz`      |
| Linux    | arm64       | `.tar.gz`      |
| macOS    | amd64 (Intel) | `.tar.gz`    |
| macOS    | arm64 (Apple Silicon) | `.tar.gz` |
| Windows  | amd64       | `.zip`         |

### Version Management

**Development Versions:**
- Built on every commit to `main` or `develop` branches
- Version format: `dev-<git-short-hash>` (e.g., `dev-abc123`)
- Available as workflow artifacts (30-day retention)

**Release Versions:**
- Triggered by version tags (e.g., `v0.1.0`)
- Version format: Follows semantic versioning
- Permanently available in the Releases section

### Creating a New Release

To trigger a release build:

```bash
# Tag the commit
git tag -a v0.2.0 -m "Release version 0.2.0"

# Push the tag
git push origin v0.2.0
```

This automatically:
1. Runs full test suite
2. Builds binaries for all platforms
3. Creates a release in Gitea
4. Uploads all artifacts and checksums

### Verifying Build Integrity

All releases include SHA256 checksums:

```bash
# Download the checksum file
wget https://gitea.example.com/scality/terraform-provider-scality/releases/download/v0.1.0/terraform-provider-scality_SHA256SUMS

# Verify your downloaded binary
sha256sum -c terraform-provider-scality_SHA256SUMS --ignore-missing
```

## Contributing

Contributions are welcome! Please:
- Follow existing code patterns
- Add tests for new features
- Update documentation
- Ensure all tests pass

## License

This provider is provided as-is for use with Scality storage systems.

## Support

For issues or questions:
- Review this documentation
- Check example configurations in `examples/`
- Verify API connectivity and credentials
- Report bugs via GitHub issues

## API Reference

Based on Scality IAM API specifications:
- CreateAccount API
- GenerateAccountAccessKey API
- DeleteAccount API

## Version History

### 0.2.0 (Current)
- Added Console API support with JWT authentication
- New `scality_console_account` resource
- Token caching for performance (23.5hr cache)
- Dual-client architecture supporting both IAM and Console APIs
- Security-first approach (accounts without passwords)

### 0.1.0 (Initial Release)
- Account resource with CRUD operations
- AWS Signature V4 authentication
- Automatic access key generation
- Comprehensive error handling
