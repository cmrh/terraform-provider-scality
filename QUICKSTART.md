# OpenTofu/Terraform Provider for Scality - Quick Start Guide

This guide will help you get started with the Scality OpenTofu/Terraform provider in minutes.

> **Recommended**: Use [OpenTofu](https://opentofu.org/) instead of Terraform for open-source licensing benefits.

## Prerequisites

- Go 1.21 or later installed
- [OpenTofu](https://opentofu.org/docs/intro/install/) 1.6+ or Terraform 1.0+ installed
- Access to a Scality storage system
- Admin credentials (access key and secret key)

## Step 1: Build the Provider

```bash
cd /path/to/repo
make build
```

This will create a `terraform-provider-scality` binary.

## Step 2: Install Locally

```bash
make install
```

This installs the provider to `~/.terraform.d/plugins/` for local development.

## Step 3: Set Up Credentials

Choose the API you want to use:

### Option A: IAM API (Environment Variables - Recommended)

```bash
export SCALITY_ENDPOINT="http://10.164.169.247"
export SCALITY_ACCESS_KEY="UE7VGEFRLGHWQMDBUD23"
export SCALITY_SECRET_KEY="Ta37=vlEzH1d3tfoSIaqOPTbKfy1l2T=SlQzTwD6"
```

### Option B: Console API (Environment Variables)

```bash
export SCALITY_CONSOLE_ENDPOINT="http://10.164.169.247:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="admin-password"
```

### Option C: Both APIs (Environment Variables)

```bash
# IAM API
export SCALITY_ENDPOINT="http://10.164.169.247"
export SCALITY_ACCESS_KEY="UE7VGEFRLGHWQMDBUD23"
export SCALITY_SECRET_KEY="Ta37=vlEzH1d3tfoSIaqOPTbKfy1l2T=SlQzTwD6"

# Console API
export SCALITY_CONSOLE_ENDPOINT="http://10.164.169.247:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="admin-password"
```

### Option D: Provider Configuration File

Create a file `terraform.tfvars`:
```hcl
# IAM API
scality_endpoint   = "http://10.164.169.247"
scality_access_key = "UE7VGEFRLGHWQMDBUD23"
scality_secret_key = "Ta37=vlEzH1d3tfoSIaqOPTbKfy1l2T=SlQzTwD6"

# Console API (optional)
scality_console_endpoint = "http://10.164.169.247:8080"
scality_console_username = "admin"
scality_console_password = "admin-password"
```

## Step 4: Create Your First Configuration

Create a file `main.tf`:

```hcl
terraform {
  required_providers {
    scality = {
      source = "scality/scality"
    }
  }
}

provider "scality" {
  endpoint   = var.scality_endpoint
  access_key = var.scality_access_key
  secret_key = var.scality_secret_key
}

variable "scality_endpoint" {
  type = string
}

variable "scality_access_key" {
  type      = string
  sensitive = true
}

variable "scality_secret_key" {
  type      = string
  sensitive = true
}

resource "scality_account" "my_first_account" {
  name          = "my-first-account"
  email_address = "first@example.com"
  quota_max     = 1000000000  # 1GB
}

output "account_id" {
  value = scality_account.my_first_account.id
}

output "access_key" {
  value     = scality_account.my_first_account.access_key
  sensitive = true
}

output "secret_key" {
  value     = scality_account.my_first_account.secret_key
  sensitive = true
}
```

## Step 5: Initialize

**Using OpenTofu (recommended):**
```bash
tofu init
```

**Using Terraform:**
```bash
terraform init
```

## Step 6: Plan and Apply

**Using OpenTofu:**
```bash
# See what will be created
tofu plan

# Create the account
tofu apply
```

**Using Terraform:**
```bash
# See what will be created
terraform plan

# Create the account
terraform apply
```

When prompted, type `yes` to confirm.

## Step 7: View the Output

```bash
# Show all outputs (credentials will be hidden)
terraform output

# Show sensitive credentials
terraform output -json | jq '.access_key.value'
terraform output -json | jq '.secret_key.value'
```

## Step 8: Save Credentials

```bash
# Save to a file
cat > my-account-credentials.env <<EOF
AWS_ACCESS_KEY_ID=$(terraform output -json | jq -r '.access_key.value')
AWS_SECRET_ACCESS_KEY=$(terraform output -json | jq -r '.secret_key.value')
EOF

chmod 600 my-account-credentials.env
```

## Step 9: Test the Credentials

```bash
# Source the credentials
source my-account-credentials.env

# Test with AWS CLI (if compatible)
aws s3 ls --endpoint-url http://10.164.169.247
```

## Step 10: Clean Up

```bash
# Destroy the account (will fail if it has buckets/users)
terraform destroy
```

## Using Console API

If you configured Console API credentials, you can create accounts via the Console API:

```hcl
terraform {
  required_providers {
    scality = {
      source = "scality/scality"
    }
  }
}

provider "scality" {
  console_endpoint = var.scality_console_endpoint
  console_username = var.scality_console_username
  console_password = var.scality_console_password
}

variable "scality_console_endpoint" {
  type = string
}

variable "scality_console_username" {
  type      = string
  sensitive = true
}

variable "scality_console_password" {
  type      = string
  sensitive = true
}

resource "scality_console_account" "my_console_account" {
  account_name = "my-console-account"
  email        = "console@example.com"
  quota        = 5000000000  # 5GB
}

output "console_account_id" {
  value = scality_console_account.my_console_account.id
}

output "console_access_key" {
  value     = scality_console_account.my_console_account.access_key
  sensitive = true
}

output "console_secret_key" {
  value     = scality_console_account.my_console_account.secret_key
  sensitive = true
}
```

### Key Differences Between IAM and Console APIs

| Feature | `scality_account` (IAM) | `scality_console_account` (Console) |
|---------|-------------------------|-------------------------------------|
| Authentication | AWS Signature V4 | JWT Token |
| Account Creation | Standard | Without password (security best practice) |
| Credentials | S3 access keys | Persistent S3 access keys |
| Token Caching | N/A | 23.5 hours |
| Deletion | Single step | Two-step (account + user) |
| Use Case | IAM-style management | Console UI integration |

## Common Tasks

### Creating Multiple Accounts

```hcl
resource "scality_account" "accounts" {
  for_each = {
    dev  = "dev@example.com"
    prod = "prod@example.com"
  }

  name          = "${each.key}-account"
  email_address = each.value
  quota_max     = 1000000000
}
```

### Importing Existing Accounts

```bash
terraform import scality_account.existing existing-account-name
```

### Updating Account Email

```hcl
resource "scality_account" "my_account" {
  name          = "my-account"
  email_address = "new-email@example.com"  # Changed
  quota_max     = 1000000000
}
```

Then:
```bash
terraform plan
terraform apply
```

## Troubleshooting

### Provider Not Found

If you see "provider not found", ensure:
1. You ran `make install`
2. The binary is in `~/.terraform.d/plugins/registry.terraform.io/scality/scality/0.1.0/linux_amd64/`
3. You ran `terraform init`

### Authentication Errors

If you see authentication errors:
1. Verify endpoint is reachable: `curl http://10.164.169.247`
2. Check credentials are correct
3. Ensure admin credentials have IAM permissions

### DeleteConflict Error

If you can't delete an account:
```
Error: Cannot delete account - contains resources
```

This means the account has buckets, users, or policies. You must:
1. Delete all buckets (and their contents)
2. Delete all IAM users
3. Delete all IAM policies
4. Then retry `terraform destroy`

## Next Steps

- Review the [README.md](README.md) for detailed documentation
- Check [examples/](examples/) for more advanced configurations
- Read about [state management](README.md#state-management) for production use

## Getting Help

- Check error messages carefully - they often contain the solution
- Review the examples in the `examples/` directory
- Ensure your Scality API is accessible and credentials are valid
- Check the Terraform logs: `TF_LOG=DEBUG terraform apply`
