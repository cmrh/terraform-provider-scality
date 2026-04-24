# scality_bucket

Manages an S3 bucket with optional versioning, object lock, and tags. Uses per-account credentials (not provider-level admin credentials).

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

# Bucket with object lock (versioning is auto-enabled by S3)
resource "scality_bucket" "locked" {
  account_access_key  = scality_account_access_key.stable.access_key
  account_secret_key  = scality_account_access_key.stable.secret_key
  bucket              = "compliance-data"
  object_lock_enabled = true
  versioning          = true
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `bucket` | String | Yes | Bucket name. Changing this replaces the resource. |
| `object_lock_enabled` | Bool | No | Enable Object Lock on the bucket. Can only be set at creation time. Changing this replaces the resource. |
| `versioning` | Bool | No | Enable versioning on the bucket. |
| `tags` | Map(String) | No | Key-value tags for the bucket. |

## Notes

- Bucket operations use S3 API signing (service `s3`), distinct from IAM operations.
- Versioning and tags are managed inline. For other bucket features (encryption, lifecycle, etc.), use the dedicated sub-resources.
- When `object_lock_enabled` is true, S3 auto-enables versioning. Use `scality_bucket_object_lock` to configure the retention policy.
- Deleting this resource deletes the bucket. The bucket must be empty.

## Credential Dependencies

When using IAM user credentials, always declare a `depends_on` pointing at the user's access policy. This ensures Terraform destroys the bucket before the policy that grants permission to manage it.

```hcl
resource "scality_bucket" "data" {
  account_access_key = scality_user_access_key.operator.access_key_id
  account_secret_key = scality_user_access_key.operator.secret_access_key
  bucket             = "my-data"

  depends_on = [scality_user_policy.operator]
}
```

Without this, `terraform destroy` may remove the user's policy first, leaving Terraform unable to delete the bucket.
