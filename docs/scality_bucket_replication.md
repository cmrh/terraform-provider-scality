# scality_bucket_replication

Configures cross-region or cross-bucket replication for an S3 bucket.

## Example

```hcl
resource "scality_bucket_replication" "backup" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = scality_bucket.primary.bucket
  role               = "arn:aws:iam::${local.account_id}:role/replication-role"

  rule {
    id                 = "replicate-all"
    status             = "Enabled"
    prefix             = ""
    destination_bucket = "arn:aws:s3:::${scality_bucket.replica.bucket}"
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `bucket` | String | Yes | Source bucket name. Changing this replaces the resource. |
| `role` | String | Yes | IAM role ARN that S3 assumes to replicate objects. |
| `rule` | Block (list) | Yes | One or more replication rules. See below. |

### Rule Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `id` | String | No | Unique identifier for the rule. |
| `status` | String | Yes | `Enabled` or `Disabled`. |
| `prefix` | String | Yes | Object key prefix filter. Use `""` for all objects. |
| `destination_bucket` | String | Yes | Destination bucket ARN (e.g. `arn:aws:s3:::my-replica`). |
| `destination_storage_class` | String | No | Storage class for replicated objects. |

## Prerequisites

- Both source and destination buckets must have versioning enabled.
- The IAM role must have permission to read from the source and write to the destination.

## Notes

- All rules are submitted as a single replication configuration.
- Deleting this resource removes the replication configuration entirely.
