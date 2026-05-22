# Changelog

All notable changes to the Scality Terraform Provider are documented in this file.

## [Unreleased]

### Added
- `data.scality_account` data source. Looks up an existing account by name and exposes `id`, `email_address`, `quota_max`, `custom_attributes`, `arn`, `canonical_id`, `create_date`. Does not expose `access_key`/`secret_key` (IAM API returns those only at creation).
- `data.scality_bucket` data source. Looks up an existing bucket by name within an account and exposes `id`, `arn`, `versioning`, `object_lock_enabled`, `tags`.
- `data.scality_accounts` data source. Lists all accounts in the cluster (paginated under the hood). Returns a `accounts` list of objects with `id`, `name`, `email_address`, `arn`, `canonical_id`, `create_date`, `quota_max`. No `custom_attributes` per entry — use `data.scality_account` with `for_each` for drill-down.
- `data.scality_buckets` data source. Lists all buckets owned by the supplied account credentials. Returns a `buckets` list of objects with `name`, `arn`, `creation_date`. No versioning/tags/object-lock per entry — use `data.scality_bucket` with `for_each` for drill-down.
- `docs/data-sources/scality_account.md`, `docs/data-sources/scality_bucket.md`, `docs/data-sources/scality_accounts.md`, `docs/data-sources/scality_buckets.md`.

### Changed
- Replaced legacy `interface{}` style with `any` in `AccountCreateResponse.AccountData.CustomAttributes` (lint hygiene; no functional change).

## [0.4.0] - 2026-04-30

First general-availability release. Aside from the doc and CI changes below, the resource set and provider schema match `0.1.0-rc.1`.

### Added
- `terraform-registry-manifest.json` at repo root for Terraform/OpenTofu Registry protocol detection.
- Documentation restructured to Terraform Registry format: `docs/index.md` (provider overview) and `docs/resources/*.md` with YAML frontmatter (`page_title`, `subcategory`, `description`).
- `## Import` section added to all 17 resource docs.
- `.golangci.yml` configuration (golangci-lint v2 schema).
- `golangci-lint` step in CI (`test.yml`), pinned to `golangci-lint-action@v8`.

### Changed
- Acceptance test `TestAccBucketReplication_crr` simplified to single-endpoint, removing the requirement for two Scality clusters in CI.
- `CheckResourceDestroyed` test helper now verifies parent accounts are destroyed for all child resource types, catching cascading-delete regressions.

### Removed
- Dead test helpers `PreCheckCRR()` and `DestProviderBlock()` from `internal/acctest/`.
- `SCALITY_DEST_*` environment variables from acceptance workflow (no longer referenced by any test).
- Unused `createPolicyVersionResponse` type in `internal/client/iam_managed_policy.go`.

### Fixed
- `staticcheck` S1016: simplified struct conversion in `internal/client/s3_encryption.go`.
- `errcheck` warnings in test HTTP handlers via `.golangci.yml` exclusion of `fmt.Fprint`/`fmt.Fprintf`/`fmt.Fprintln`.

## [0.1.0-rc.1] - 2026-04-26

### Added
- Core resources: `scality_account`, `scality_console_account`, `scality_account_access_key`, `scality_user`, `scality_user_access_key`, `scality_user_policy`, `scality_group`, `scality_group_membership`, `scality_bucket`
- Bucket sub-resources: `scality_bucket_encryption`, `scality_bucket_lifecycle`, `scality_bucket_object_lock`, `scality_bucket_policy`, `scality_bucket_replication`
- IAM resources: `scality_iam_policy`, `scality_iam_role`, `scality_iam_role_policy_attachment`
- Three-client architecture: IAMClient (SigV4), S3Client (SigV4), ConsoleClient (JWT)
- `insecure_skip_verify` provider attribute for self-signed TLS certificates
- Custom account attributes support
- Console password generation for `scality_console_account`
- Input validation on all user-facing schema attributes (account names, bucket names, emails, IAM names, policy ARNs, JSON documents)
- Unit tests for validators, schema wiring, client layer, and client concurrency
- `ARCHITECTURE.md` documenting provider design, client layer, and resource patterns
- Multi-platform release workflow (linux/darwin/windows, amd64/arm64)

### Fixed
- Race condition in ConsoleClient token management under concurrent resource creation
- URL path escaping for account names and resource identifiers containing special characters
- Swallowed `io.ReadAll` errors in `DeleteConsoleAccount` error paths
- `w.Write()` return values checked in test HTTP handlers

### Changed
- Provider registers all resources through a single `Resources()` method
- Release workflow: `.zip` archives, protocol version `6.0`, Terraform-convention binary naming
- Makefile: version injection via `-ldflags`, dynamic platform detection

### Security
- Bumped `golang.org/x/net` from 0.17.0 to 0.38.0
