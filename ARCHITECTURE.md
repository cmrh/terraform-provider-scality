# Architecture

## Overview

```
OpenTofu / Terraform
        │
        │  Plugin Protocol (gRPC)
        ▼
┌─────────────────────────────────────────┐
│            Scality Provider             │
│                                         │
│  provider.go ── config, env vars,       │
│                 resource registration   │
│                                         │
│  ┌────────────────────────────────────┐ │
│  │         Client Layer               │ │
│  │                                    │ │
│  │  IAMClient     (SigV4, service     │ │
│  │                 "iam")             │ │
│  │  S3Client      (SigV4, service     │ │
│  │                 "s3")              │ │
│  │  ConsoleClient (JWT via            │ │
│  │                 x-access-token)    │ │
│  └────────────────────────────────────┘ │
│                                         │
│  ┌────────────────────────────────────┐ │
│  │         Resources (18)             │ │
│  │  Each: model.go + resource.go      │ │
│  └────────────────────────────────────┘ │
└─────────────────────────────────────────┘
        │              │             │
        ▼              ▼             ▼
   Scality IAM    Scality S3    Console API
```

## Directory Layout

```
internal/
├── acctest/
│   └── acctest.go              # Acceptance test helpers (provider factories, PreCheck funcs)
├── client/
│   ├── provider_clients.go     # ProviderClients bundle (IAM + S3 + Console)
│   ├── iam.go                  # IAM client: SigV4 signing, account/user/group ops
│   ├── iam_managed_policy.go   # IAM managed policy CRUD (CreatePolicy, GetPolicy, etc.)
│   ├── iam_role.go             # IAM role CRUD and role policy attachment
│   ├── console.go              # Console client: JWT auth, token caching, mutex-guarded
│   ├── s3.go                   # S3 client: SigV4 signing (service "s3"), doRequest
│   ├── s3_bucket.go            # CreateBucket, HeadBucket, DeleteBucket
│   ├── s3_versioning.go        # Get/PutBucketVersioning
│   ├── s3_tagging.go           # Get/Put/DeleteBucketTagging
│   ├── s3_policy.go            # Get/Put/DeleteBucketPolicy (JSON, not XML)
│   ├── s3_encryption.go        # Get/Put/DeleteBucketEncryption
│   ├── s3_acl.go               # Get/PutBucketACL (canned ACL via x-amz-acl header)
│   ├── s3_lifecycle.go         # Get/Put/DeleteBucketLifecycle
│   ├── s3_object_lock.go       # Get/PutObjectLockConfiguration
│   └── s3_replication.go       # Get/Put/DeleteBucketReplication
├── provider/
│   └── provider.go             # Schema, Configure, Resources()
├── validators/
│   └── validators.go           # Shared input validators (AccountName, BucketName, IAMName, etc.)
└── resources/
    ├── account/                # scality_account
    ├── console_account/        # scality_console_account
    ├── account_access_key/     # scality_account_access_key
    ├── bucket/                 # scality_bucket
    ├── bucket_policy/          # scality_bucket_policy
    ├── bucket_encryption/      # scality_bucket_encryption
    ├── bucket_acl/             # scality_bucket_acl
    ├── bucket_lifecycle/       # scality_bucket_lifecycle
    ├── bucket_object_lock/     # scality_bucket_object_lock
    ├── bucket_replication/     # scality_bucket_replication
    ├── iam_policy/             # scality_iam_policy
    ├── iam_role/               # scality_iam_role
    ├── iam_role_policy_attachment/ # scality_iam_role_policy_attachment
    ├── user/                   # scality_user
    ├── user_access_key/        # scality_user_access_key
    ├── user_policy/            # scality_user_policy
    ├── group/                  # scality_group
    └── group_membership/       # scality_group_membership
```

Each resource is a package with:
- `model.go` -- struct with `tfsdk` tags
- `resource.go` -- schema, Configure, CRUD, import
- `resource_acc_test.go` -- acceptance tests (gated behind `TF_ACC=1`)

## Three Clients, Three Auth Models

| Client | Auth | Wire Format | Used By |
|--------|------|-------------|---------|
| `IAMClient` | SigV4 (service `iam`) | form-encoded actions | account, user, user_access_key, user_policy, group, group_membership, iam_policy, iam_role, iam_role_policy_attachment |
| `S3Client` | SigV4 (service `s3`) | XML (JSON for bucket policy) | bucket, bucket_acl, bucket_encryption, bucket_lifecycle, bucket_object_lock, bucket_policy, bucket_replication |
| `ConsoleClient` | JWT (`x-access-token` header) | JSON/REST | console_account, account_access_key |

The provider bundles all three in `ProviderClients`. Each resource extracts the client it needs in its `Configure` method.

### IAM vs S3 signing

Both use AWS SigV4, but differ in the service name and URL structure:

- **IAM**: service `iam`, all requests POST to `/` with form-encoded `Action=...` body
- **S3**: service `s3`, path-style URLs (`/bucket-name?subresource`), XML request/response bodies

Scality's nginx routes requests based on the service field in the SigV4 `Authorization` header.

### Per-account credentials

Account-level resources (buckets, users, groups) use per-account credentials passed as resource attributes, not the provider-level admin credentials. This is because standard AWS IAM/S3 operations require the owning account's keys.

## Input Validation

The `internal/validators` package provides reusable schema validators following AWS IAM/S3 naming rules:

| Validator | Rules | Used On |
|-----------|-------|---------|
| `AccountName()` | 1-128 chars, alphanumeric + hyphens | account name, console account name |
| `BucketName()` | 3-63 chars, lowercase + numbers + hyphens + periods | bucket fields across all bucket resources |
| `Email()` | Standard email syntax | account email |
| `IAMName(maxLen)` | 1-maxLen chars, alphanumeric + `_+=,.@-` | user, group, role, policy names |
| `PolicyARN()` | `arn:aws:iam::*:policy/*` pattern | policy_arn on role_policy_attachment |
| `JSONDocument()` | Valid JSON | policy documents, trust policies |
| `OneOf(values...)` | Exact string match | ACL types, SSE algorithms, retention modes, rule status |
| `Int64AtLeast(min)` | Minimum integer value | password_length |

Schema wiring is verified by `internal/provider/schema_validators_test.go`, which checks that every validated attribute has its validators attached.

## Testing

Unit tests (`*_test.go`) exist for the client layer and validators package. Acceptance tests (`resource_acc_test.go`) exist in every resource package and are gated behind `TF_ACC=1`. Test infrastructure lives in `internal/acctest/`.

## Resource Patterns

### Atomic Create

Account resources save state immediately after creation, before generating access keys. If key generation fails, the account is still tracked in state and can be destroyed or retried.

### RequiresReplace

Fields that the API cannot update in-place use `stringplanmodifier.RequiresReplace()`. Terraform destroys and recreates the resource when these change.

### State-only Read

Resources where the API cannot return secrets after creation (access keys) preserve state as-is in Read. No API call is made.

### Bucket sub-resources

Bucket features (policy, encryption, ACL, lifecycle, object lock, replication) are separate resources rather than inline attributes on `scality_bucket`. This keeps each resource focused and allows independent lifecycle management. The `bucket` attribute on each sub-resource uses `RequiresReplace`.
