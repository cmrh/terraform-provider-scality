---
page_title: "scality_bucket_encryption Resource - scality"
subcategory: "S3 Buckets"
description: |-
  Configures server-side encryption for an S3 bucket.
---

# scality_bucket_encryption

Configures server-side encryption for an S3 bucket.

## Example

```hcl
resource "scality_bucket_encryption" "data" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = scality_bucket.data.bucket
  sse_algorithm      = "AES256"
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `bucket` | String | Yes | Target bucket name. Changing this replaces the resource. |
| `sse_algorithm` | String | Yes | Encryption algorithm: `AES256` (SSE-S3) or `aws:kms` (SSE-KMS). |
| `kms_master_key_id` | String | No | KMS key ID. Only used when `sse_algorithm` is `aws:kms`. |

## Notes

- `AES256` (SSE-S3) works in all deployments.
- `aws:kms` (SSE-KMS) requires an external KMS server configured on the Scality side. This is uncommon.
- Deleting this resource removes the encryption configuration from the bucket.
