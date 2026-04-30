---
page_title: "scality_bucket_object_lock Resource - scality"
subcategory: "S3 Buckets"
description: |-
  Configures default object lock retention for an S3 bucket. Prevents objects from being deleted or overwritten for a fixed retention period.
---

# scality_bucket_object_lock

Configures default object lock retention for an S3 bucket. Prevents objects from being deleted or overwritten for a fixed retention period.

## Example

```hcl
resource "scality_bucket" "immutable" {
  account_access_key  = local.ak
  account_secret_key  = local.sk
  bucket              = "compliance-data"
  object_lock_enabled = true
  versioning          = true
}

resource "scality_bucket_object_lock" "compliance" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = scality_bucket.immutable.bucket
  retention_mode     = "COMPLIANCE"
  retention_days     = 365
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `bucket` | String | Yes | Target bucket name. Changing this replaces the resource. |
| `retention_mode` | String | Yes | `GOVERNANCE` or `COMPLIANCE`. |
| `retention_days` | Int | No | Retention period in days. Mutually exclusive with `retention_years`. |
| `retention_years` | Int | No | Retention period in years. Mutually exclusive with `retention_days`. |

### Retention Modes

| Mode | Description |
|------|-------------|
| `GOVERNANCE` | Users with special permissions (`s3:BypassGovernanceRetention`) can override or delete locks. |
| `COMPLIANCE` | No one can override or delete locks, including the root account. Objects cannot be deleted until the retention period expires. |

## Prerequisites

The bucket must be created with `object_lock_enabled = true`. Object lock can only be enabled at bucket creation time -- it cannot be added to an existing bucket. When object lock is enabled, S3 automatically enables versioning.

## Notes

- Specify either `retention_days` or `retention_years`, not both.
- Deleting this resource removes the default retention configuration but does not disable object lock on the bucket (object lock cannot be disabled once enabled).
- Objects already locked retain their individual lock settings regardless of changes to the default configuration.
