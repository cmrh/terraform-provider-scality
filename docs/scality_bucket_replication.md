# scality_bucket_replication

Configures cross-region replication (CRR) for an S3 bucket. CRR spans two independent Scality clusters and requires versioned buckets on both source and destination, IAM roles with a backbeat trust policy, and managed policies granting replication permissions.

## Example

CRR requires two provider instances — one per cluster. Use the `alias` argument to define a second provider pointing at the destination cluster.

Create a `.env` file with credentials for both clusters:

```bash
# Source cluster
SCALITY_ENDPOINT="http://source-cluster:8080"
SCALITY_ACCESS_KEY="<source-admin-access-key>"
SCALITY_SECRET_KEY="<source-admin-secret-key>"

# Destination cluster
TF_VAR_dest_endpoint="http://dest-cluster:8080"
TF_VAR_dest_access_key="<dest-admin-access-key>"
TF_VAR_dest_secret_key="<dest-admin-secret-key>"
```

The default provider reads `SCALITY_*` variables automatically. The destination provider uses `TF_VAR_*` variables which Terraform/OpenTofu maps to input variables:

```hcl
variable "dest_endpoint" {}
variable "dest_access_key" { sensitive = true }
variable "dest_secret_key" { sensitive = true }

provider "scality" {
  insecure_skip_verify = true
}

provider "scality" {
  alias                = "dest"
  endpoint             = var.dest_endpoint
  access_key           = var.dest_access_key
  secret_key           = var.dest_secret_key
  insecure_skip_verify = true
}

# --- Source side ---

resource "scality_account" "source" {
  name          = "crr-source"
  email_address = "crr-source@example.com"
}

resource "scality_user" "source" {
  account_access_key = scality_account.source.access_key
  account_secret_key = scality_account.source.secret_key
  username           = "operator"
}

resource "scality_user_policy" "source" {
  account_access_key = scality_account.source.access_key
  account_secret_key = scality_account.source.secret_key
  username           = scality_user.source.username
  policy_name        = "full-access"
  policy_document    = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "*"
      Resource = "*"
    }]
  })
}

resource "scality_user_access_key" "source" {
  account_access_key = scality_account.source.access_key
  account_secret_key = scality_account.source.secret_key
  username           = scality_user.source.username
}

resource "scality_account_access_key" "source" {
  account_access_key = scality_account.source.access_key
  account_secret_key = scality_account.source.secret_key
}

resource "scality_bucket" "source" {
  account_access_key = scality_account_access_key.source.access_key
  account_secret_key = scality_account_access_key.source.secret_key
  bucket             = "crr-source-bucket"
  versioning         = true

  depends_on = [scality_user_policy.source]
}

resource "scality_iam_policy" "source" {
  account_access_key = scality_account.source.access_key
  account_secret_key = scality_account.source.secret_key
  policy_name        = "crr-policy"
  policy_document    = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:GetObjectVersion", "s3:GetObjectVersionAcl", "s3:GetObjectTagging", "s3:ReplicateObject", "s3:ReplicateTags", "s3:ReplicateDelete"]
        Resource = "arn:aws:s3:::${scality_bucket.source.bucket}/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket", "s3:GetReplicationConfiguration"]
        Resource = "arn:aws:s3:::${scality_bucket.source.bucket}"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ReplicateObject", "s3:ReplicateDelete", "s3:ReplicateTags", "s3:PutObject"]
        Resource = "arn:aws:s3:::${scality_bucket.dest.bucket}/*"
      },
    ]
  })
}

resource "scality_iam_role" "source" {
  account_access_key = scality_account.source.access_key
  account_secret_key = scality_account.source.secret_key
  role_name          = "crr-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "backbeat" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "scality_iam_role_policy_attachment" "source" {
  account_access_key = scality_account.source.access_key
  account_secret_key = scality_account.source.secret_key
  role_name          = scality_iam_role.source.role_name
  policy_arn         = scality_iam_policy.source.arn
}

# --- Destination side ---

resource "scality_account" "dest" {
  provider      = scality.dest
  name          = "crr-dest"
  email_address = "crr-dest@example.com"
}

resource "scality_user" "dest" {
  provider           = scality.dest
  account_access_key = scality_account.dest.access_key
  account_secret_key = scality_account.dest.secret_key
  username           = "operator"
}

resource "scality_user_policy" "dest" {
  provider           = scality.dest
  account_access_key = scality_account.dest.access_key
  account_secret_key = scality_account.dest.secret_key
  username           = scality_user.dest.username
  policy_name        = "full-access"
  policy_document    = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "*"
      Resource = "*"
    }]
  })
}

resource "scality_user_access_key" "dest" {
  provider           = scality.dest
  account_access_key = scality_account.dest.access_key
  account_secret_key = scality_account.dest.secret_key
  username           = scality_user.dest.username
}

resource "scality_account_access_key" "dest" {
  provider           = scality.dest
  account_access_key = scality_account.dest.access_key
  account_secret_key = scality_account.dest.secret_key
}

resource "scality_bucket" "dest" {
  provider           = scality.dest
  account_access_key = scality_account_access_key.dest.access_key
  account_secret_key = scality_account_access_key.dest.secret_key
  bucket             = "crr-dest-bucket"
  versioning         = true

  depends_on = [scality_user_policy.dest]
}

resource "scality_iam_policy" "dest" {
  provider           = scality.dest
  account_access_key = scality_account.dest.access_key
  account_secret_key = scality_account.dest.secret_key
  policy_name        = "crr-policy"
  policy_document    = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:GetObjectVersion", "s3:GetObjectVersionAcl", "s3:GetObjectTagging", "s3:ReplicateObject", "s3:ReplicateTags", "s3:ReplicateDelete"]
        Resource = "arn:aws:s3:::${scality_bucket.source.bucket}/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket", "s3:GetReplicationConfiguration"]
        Resource = "arn:aws:s3:::${scality_bucket.source.bucket}"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ReplicateObject", "s3:ReplicateDelete", "s3:ReplicateTags", "s3:PutObject"]
        Resource = "arn:aws:s3:::${scality_bucket.dest.bucket}/*"
      },
    ]
  })
}

resource "scality_iam_role" "dest" {
  provider           = scality.dest
  account_access_key = scality_account.dest.access_key
  account_secret_key = scality_account.dest.secret_key
  role_name          = "crr-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "backbeat" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "scality_iam_role_policy_attachment" "dest" {
  provider           = scality.dest
  account_access_key = scality_account.dest.access_key
  account_secret_key = scality_account.dest.secret_key
  role_name          = scality_iam_role.dest.role_name
  policy_arn         = scality_iam_policy.dest.arn
}

# --- Replication: source → dest ---

resource "scality_bucket_replication" "source_to_dest" {
  account_access_key = scality_account_access_key.source.access_key
  account_secret_key = scality_account_access_key.source.secret_key
  bucket             = scality_bucket.source.bucket
  role               = "${scality_iam_role.source.arn},${scality_iam_role.dest.arn}"

  rule {
    status             = "Enabled"
    prefix             = ""
    destination_bucket = "arn:aws:s3:::${scality_bucket.dest.bucket}"
  }

  depends_on = [
    scality_iam_role_policy_attachment.source,
    scality_iam_role_policy_attachment.dest,
  ]
}
```

## Bilateral replication

Each bucket can replicate to the other, giving active-active synchronization across both clusters. Add a second `scality_bucket_replication` on the destination side that points back to the source bucket. The same roles and policies cover both directions since they already grant replication permissions on both buckets.

```hcl
resource "scality_bucket_replication" "dest_to_source" {
  provider           = scality.dest
  account_access_key = scality_account_access_key.dest.access_key
  account_secret_key = scality_account_access_key.dest.secret_key
  bucket             = scality_bucket.dest.bucket
  role               = "${scality_iam_role.dest.arn},${scality_iam_role.source.arn}"

  rule {
    status             = "Enabled"
    prefix             = ""
    destination_bucket = "arn:aws:s3:::${scality_bucket.source.bucket}"
  }

  depends_on = [
    scality_iam_role_policy_attachment.source,
    scality_iam_role_policy_attachment.dest,
  ]
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `bucket` | String | Yes | Source bucket name. Changing this replaces the resource. |
| `role` | String | Yes | Comma-separated pair of IAM role ARNs: `source_role_arn,dest_role_arn`. S3 assumes these roles to replicate objects. |
| `rule` | Block (list) | Yes | One or more replication rules. See below. |

### Rule Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `id` | String | No | Unique identifier for the rule. Auto-generated by the server if not specified. |
| `status` | String | Yes | `Enabled` or `Disabled`. |
| `prefix` | String | Yes | Object key prefix filter. Use `""` for all objects. |
| `destination_bucket` | String | Yes | Destination bucket ARN (e.g. `arn:aws:s3:::my-replica`). |
| `destination_storage_class` | String | No | Storage class for replicated objects. |

## Prerequisites

- Both source and destination buckets must have **versioning enabled**.
- IAM roles must have a trust policy allowing the `backbeat` service to assume them.
- Managed policies granting replication permissions must be attached to the roles.
- Use `depends_on` to ensure role-policy attachments are created before the replication configuration.

## Notes

- All rules are submitted as a single replication configuration.
- Deleting this resource removes the replication configuration entirely.
- The rule `id` is computed by the server if not provided and stored in state to prevent drift.
