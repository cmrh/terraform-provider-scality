# Example: Creating multiple accounts with different quotas

terraform {
  required_providers {
    scality = {
      source  = "scality/scality"
      version = "~> 0.4"
    }
  }
}

provider "scality" {
  # Configuration can also be set via environment variables:
  # SCALITY_ENDPOINT, SCALITY_ACCESS_KEY, SCALITY_SECRET_KEY
}

# Development account
resource "scality_account" "dev" {
  name          = "development"
  email_address = "dev-team@example.com"
  quota_max     = 5000000000 # 5GB
}

# Staging account
resource "scality_account" "staging" {
  name          = "staging"
  email_address = "staging@example.com"
  quota_max     = 10000000000 # 10GB
}

# Production account
resource "scality_account" "prod" {
  name          = "production"
  email_address = "production@example.com"
  quota_max     = 100000000000 # 100GB
}

# Save credentials to local files (be careful with this in production!)
resource "local_file" "dev_credentials" {
  filename = "${path.module}/credentials/dev.env"
  content  = <<-EOT
    AWS_ACCESS_KEY_ID=${scality_account.dev.access_key}
    AWS_SECRET_ACCESS_KEY=${scality_account.dev.secret_key}
  EOT
  file_permission = "0600"
}

resource "local_file" "staging_credentials" {
  filename = "${path.module}/credentials/staging.env"
  content  = <<-EOT
    AWS_ACCESS_KEY_ID=${scality_account.staging.access_key}
    AWS_SECRET_ACCESS_KEY=${scality_account.staging.secret_key}
  EOT
  file_permission = "0600"
}

resource "local_file" "prod_credentials" {
  filename = "${path.module}/credentials/prod.env"
  content  = <<-EOT
    AWS_ACCESS_KEY_ID=${scality_account.prod.access_key}
    AWS_SECRET_ACCESS_KEY=${scality_account.prod.secret_key}
  EOT
  file_permission = "0600"
}

# Output account IDs
output "account_ids" {
  value = {
    dev     = scality_account.dev.id
    staging = scality_account.staging.id
    prod    = scality_account.prod.id
  }
}
