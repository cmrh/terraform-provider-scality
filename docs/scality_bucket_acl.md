# scality_bucket_acl

Sets a canned ACL on an S3 bucket.

## Example

```hcl
resource "scality_bucket_acl" "data" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = scality_bucket.data.bucket
  acl                = "private"
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `bucket` | String | Yes | Target bucket name. Changing this replaces the resource. |
| `acl` | String | Yes | Canned ACL to apply. |

### Supported ACL Values

| ACL | Description |
|-----|-------------|
| `private` | Owner gets full control. No public access. |
| `public-read` | Owner gets full control. Everyone else gets read access. |
| `public-read-write` | Everyone gets read and write access. |
| `authenticated-read` | Owner gets full control. Authenticated users get read access. |

## Notes

- Deleting this resource resets the bucket ACL to `private`.
- The Read operation preserves state from the last apply. The raw ACL XML from the server is not reverse-mapped to a canned ACL name.
