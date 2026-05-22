---
page_title: "scality_buckets Data Source - scality"
subcategory: "Buckets"
description: |-
  Lists all S3 buckets owned by a given Scality account. Useful for inventory, audit, and dashboard tooling.
---

# scality_buckets (Data Source)

Lists every S3 bucket owned by the account whose credentials are supplied. Use this for inventory, audit, or to feed a `for_each` over all buckets — combine with `data.scality_bucket` for per-bucket drill-down (versioning, tags, object-lock state).

The list does **not** include versioning, tags, or object-lock state on each entry — those require per-bucket API calls. If you need them, iterate over `data.scality_buckets.buckets` with `for_each` into `data.scality_bucket` (composition pattern shown below).

Empty account returns an empty `buckets` list, not an error.

## Example

```hcl
data "scality_buckets" "all" {
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
}

output "bucket_names" {
  value = [for b in data.scality_buckets.all.buckets : b.name]
}

# Composition pattern — drill down into each bucket for full details:

data "scality_bucket" "per_bucket" {
  for_each           = { for b in data.scality_buckets.all.buckets : b.name => b }
  account_access_key = var.account_ak
  account_secret_key = var.account_sk
  bucket             = each.value.name
}

output "versioning_state" {
  value = {
    for name, bucket in data.scality_bucket.per_bucket :
    name => bucket.versioning
  }
}
```

## Argument Reference

- `account_access_key` (Required, String, Sensitive) — Access key of the account whose buckets to list.
- `account_secret_key` (Required, String, Sensitive) — Secret key of the account whose buckets to list.

## Attribute Reference

- `id` (String) — Synthetic identifier for the data source instance.
- `buckets` (List of Object) — All buckets owned by the account. Each element has:
  - `name` (String) — Bucket name.
  - `arn` (String) — ARN of the bucket (`arn:aws:s3:::<bucket>`).
  - `creation_date` (String) — Bucket creation date.

## Notes

- Buckets are scoped per account — the credentials supplied decide which namespace is listed.
- S3 `ListBuckets` is a single-call API; no pagination.
- Filtering inputs are not currently supported. Use HCL `for` expressions on the result to filter client-side.
- For per-bucket versioning, tags, or object-lock state, use `data.scality_bucket` as shown in the composition pattern above.
