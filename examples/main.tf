terraform {
  required_providers {
    scality = {
      source = "scality/scality"
    }
  }
}

provider "scality" {
  endpoint = "http://10.164.169.247"
  # Credentials are read from environment variables — never hardcode them here.
  # export AWS_ACCESS_KEY_ID="your-admin-access-key"
  # export AWS_SECRET_ACCESS_KEY="your-admin-secret-key"
}

# Create a basic account
resource "scality_account" "example" {
  name          = "terraform-example"
  email_address = "terraform@example.com"
  quota_max     = 1000000000 # 1GB
}

# Output the generated credentials
output "access_key" {
  value     = scality_account.example.access_key
  sensitive = true
}

output "secret_key" {
  value     = scality_account.example.secret_key
  sensitive = true
}

output "account_id" {
  value = scality_account.example.id
}

output "canonical_id" {
  value = scality_account.example.canonical_id
}
