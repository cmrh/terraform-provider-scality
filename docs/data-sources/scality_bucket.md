---
page_title: "scality_bucket Data Source - scality"
subcategory: "Buckets"
description: |-
  Looks up an existing S3 bucket by name within a Scality account. Useful for attaching Terraform-managed sub-resources to a bucket created outside the current Terraform configuration.
---

# scality_bucket (Data Source)

Looks up an existing S3 bucket within an account. Use this when you want to attach Terraform-managed sub-resources — bucket policies, lifecycle rules, replication, etc. — to a bucket that was created outside the current Terraform configuration (manually, by another Terraform project, or via other tooling).

## Example

```hcl
data "scality_bucket" "legacy" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  bucket             = "legacy-data"
}

# Attach a managed lifecycle to an existing bucket:

resource "scality_bucket_lifecycle" "legacy" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  bucket             = data.scality_bucket.legacy.bucket

  rule {
    id                                 = "expire-noncurrent"
    status                             = "Enabled"
    prefix                             = ""
    noncurrent_version_expiration_days = 90
  }
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account that owns the bucket.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account that owns the bucket.
- `bucket` (Required, String) — Name of the bucket to look up.

## Attribute Reference

- `id` (String) — Bucket identifier (same value as `bucket`).
- `arn` (String) — ARN of the bucket (`arn:aws:s3:::<bucket>`).
- `versioning` (Boolean) — Whether versioning is enabled.
- `object_lock_enabled` (Boolean) — Whether Object Lock is enabled.
- `tags` (Map of String) — Tags attached to the bucket.

## Notes

- If the bucket doesn't exist within the supplied account's namespace, the data source errors at plan time.
- The bucket is scoped per account — the credentials decide which namespace is consulted.
- Sub-feature lookups (encryption configuration, lifecycle rules, replication configuration of the looked-up bucket) are **not** included here. If you need those, open an issue on the provider repository — they can be added as separate data sources following the same per-feature modular pattern as the resources.
