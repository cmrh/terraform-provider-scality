---
page_title: "scality_bucket_lifecycle Resource - scality"
subcategory: "S3 Buckets"
description: |-
  Configures lifecycle rules for an S3 bucket. Rules can expire objects, expire noncurrent versions, and abort incomplete multipart uploads.
---

# scality_bucket_lifecycle

Configures lifecycle rules for an S3 bucket. Rules can expire objects, expire noncurrent versions, and abort incomplete multipart uploads.

## Example

```hcl
resource "scality_bucket_lifecycle" "data" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = scality_bucket.data.bucket

  rule {
    id              = "expire-old-logs"
    status          = "Enabled"
    prefix          = "logs/"
    expiration_days = 90
  }

  rule {
    id                                    = "abort-incomplete-uploads"
    status                                = "Enabled"
    prefix                                = ""
    abort_incomplete_multipart_upload_days = 7
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `bucket` | String | Yes | Target bucket name. Changing this replaces the resource. |
| `rule` | Block (list) | Yes | One or more lifecycle rules. See below. |

### Rule Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `id` | String | Yes | Unique identifier for the rule. |
| `status` | String | Yes | `Enabled` or `Disabled`. |
| `prefix` | String | No | Object key prefix filter. Use `""` for all objects. |
| `expiration_days` | Int | No | Delete objects after this many days. |
| `expiration_date` | String | No | Delete objects after this date (ISO 8601). |
| `noncurrent_version_expiration_days` | Int | No | Delete noncurrent versions after this many days. |
| `abort_incomplete_multipart_upload_days` | Int | No | Abort incomplete multipart uploads after this many days. |

## Notes

- Each rule must have at least one action (`expiration_days`, `expiration_date`, `noncurrent_version_expiration_days`, or `abort_incomplete_multipart_upload_days`).
- All rules are submitted as a single configuration. Updating any rule replaces the entire lifecycle configuration.
- Deleting this resource removes all lifecycle rules from the bucket.

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the bucket name and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_bucket_lifecycle.example BUCKET_NAME
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_bucket_lifecycle.example "ACCESS_KEY:SECRET_KEY:BUCKET_NAME"
```
