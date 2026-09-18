---
page_title: "scality_bucket_logging Resource - scality"
subcategory: "S3 Buckets"
description: |-
  Configures server access logging for an S3 bucket.
---

# scality_bucket_logging

Configures server access logging for an S3 bucket. When enabled, S3 API requests
to the source bucket are recorded and delivered as log objects to a target bucket.

~> **Server access logging must be enabled at the cluster level** before this
resource can be used. When the feature is disabled, the API returns
`501 NotImplemented` and this resource errors with a message telling you to have
an administrator enable it. Ask your Scality administrator to enable server access
logging cluster-wide.

## Example

```hcl
resource "scality_bucket" "logs" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = "app-logs"
}

resource "scality_bucket_logging" "app" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = scality_bucket.app.bucket
  target_bucket      = scality_bucket.logs.bucket
  target_prefix      = "app/"
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | No | Access key of the owning account. Sensitive. Omit to use the provider's assume_role credentials. |
| `account_secret_key` | String | No | Secret key of the owning account. Sensitive. Omit to use the provider's assume_role credentials. |
| `bucket` | String | Yes | Source bucket to log requests for. Changing this replaces the resource. |
| `target_bucket` | String | Yes | Bucket where log objects are delivered. Must be owned by the same account as the source bucket. |
| `target_prefix` | String | Yes | Key prefix for log objects in the target bucket. May be an empty string. |

## Notes

- The target bucket must grant `s3:PutObject` to the internal logging service
  user (`scality-internal/service-access-logging-user` by default), or log
  delivery silently fails even though this resource applies successfully. Attach a
  [`scality_bucket_policy`](scality_bucket_policy.md) to the target bucket
  granting that permission.
- Always use a dedicated target bucket. If the target and source are the same
  bucket, each delivered log object generates new log records, creating a
  feedback loop.
- Deleting this resource disables logging on the source bucket.

## Import

Set `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` so the import ID carries only the bucket name and no credentials leak into shell history or CI logs:

```bash
export SCALITY_ACCOUNT_ACCESS_KEY=...
export SCALITY_ACCOUNT_SECRET_KEY=...
tofu import scality_bucket_logging.example BUCKET_NAME
```

When those env vars are unset, the import ID embeds the account credentials directly:

```bash
tofu import scality_bucket_logging.example "ACCESS_KEY:SECRET_KEY:BUCKET_NAME"
```
