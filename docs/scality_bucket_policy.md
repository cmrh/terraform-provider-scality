# scality_bucket_policy

Attaches a JSON bucket policy to an S3 bucket. The policy controls access permissions for the bucket and its objects.

## Example

```hcl
resource "scality_bucket_policy" "allow_user" {
  account_access_key = local.ak
  account_secret_key = local.sk
  bucket             = scality_bucket.data.bucket

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "AllowUserAccess"
      Effect    = "Allow"
      Principal = { AWS = scality_user.operator.arn }
      Action    = "s3:*"
      Resource = [
        "arn:aws:s3:::${scality_bucket.data.bucket}",
        "arn:aws:s3:::${scality_bucket.data.bucket}/*"
      ]
    }]
  })
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. |
| `bucket` | String | Yes | Target bucket name. Changing this replaces the resource. |
| `policy` | String | Yes | JSON policy document. Use `jsonencode()` for readability. |

## Notes

- The policy document follows the standard AWS IAM policy syntax.
- Updating the `policy` attribute applies the new policy in-place (no replacement).
- Deleting this resource removes the bucket policy entirely.
