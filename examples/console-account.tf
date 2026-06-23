terraform {
  required_providers {
    scality = {
      source  = "cmrh/scality"
      version = "~> 1.0"
    }
  }
}

provider "scality" {
  # Console API credentials are read from environment variables:
  # export SCALITY_CONSOLE_ENDPOINT="http://scality.example.com:8080"
  # export SCALITY_CONSOLE_USERNAME="admin"
  # export SCALITY_CONSOLE_PASSWORD="your-password"
}

# Create account without password (default - for service accounts)
resource "scality_console_account" "service" {
  account_name = "service-account"
  email        = "service@example.com"
  quota        = 10737418240 # 10GB
}

# Create account with random password (for human Console UI access)
resource "scality_console_account" "admin" {
  account_name             = "admin-user"
  email                    = "admin@example.com"
  quota                    = 53687091200 # 50GB
  generate_random_password = true
  password_length          = 20 # Optional, default 16, minimum 16
}

# Output service account credentials
output "service_credentials" {
  description = "S3 credentials for service account"
  value = {
    access_key = scality_console_account.service.access_key
    secret_key = scality_console_account.service.secret_key
  }
  sensitive = true
}

# Output admin account credentials (including Console password)
output "admin_credentials" {
  description = "Console and S3 credentials for admin user"
  value = {
    account_name = scality_console_account.admin.account_name
    password     = scality_console_account.admin.password
    access_key   = scality_console_account.admin.access_key
    secret_key   = scality_console_account.admin.secret_key
  }
  sensitive = true
}

output "account_info" {
  description = "Account metadata"
  value = {
    service = {
      id         = scality_console_account.service.id
      created_at = scality_console_account.service.created_at
    }
    admin = {
      id         = scality_console_account.admin.id
      created_at = scality_console_account.admin.created_at
    }
  }
}
