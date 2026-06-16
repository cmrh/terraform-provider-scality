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

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the bucket name and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_bucket_encryption.example BUCKET_NAME
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_bucket_encryption.example "ACCESS_KEY:SECRET_KEY:BUCKET_NAME"
```
