# scality_bucket

Manages an S3 bucket with optional versioning and tags. Uses per-account credentials (not provider-level admin credentials).

## Example

```hcl
resource "scality_bucket" "data" {
  account_access_key = scality_account_access_key.stable.access_key
  account_secret_key = scality_account_access_key.stable.secret_key
  bucket             = "application-data"

  versioning = true

  tags = {
    environment = "production"
    managed_by  = "terraform"
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `bucket` | String | Yes | Bucket name. Changing this replaces the resource. |
| `versioning` | Bool | No | Enable versioning on the bucket. |
| `tags` | Map(String) | No | Key-value tags for the bucket. |

## Notes

- Bucket operations use S3 API signing (service `s3`), distinct from IAM operations.
- Versioning and tags are managed inline. For other bucket features (encryption, lifecycle, etc.), use the dedicated sub-resources.
- Deleting this resource deletes the bucket. The bucket must be empty.
