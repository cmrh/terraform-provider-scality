# Changelog

All notable changes to the Scality Terraform Provider are documented in this file.

## [Unreleased]

### Added
- Input validation on all user-facing schema attributes (account names, bucket names, emails, IAM names, policy ARNs, JSON documents)
- Unit tests for validators, schema wiring, and client concurrency
- `ARCHITECTURE.md` documenting provider design, client layer, and resource patterns

### Fixed
- Race condition in ConsoleClient token management under concurrent resource creation
- Swallowed `io.ReadAll` errors in `DeleteConsoleAccount` error paths
- `w.Write()` return values checked in test HTTP handlers

### Changed
- Release workflow: archives now use `.zip` format (Terraform/OpenTofu compatible)
- Release workflow: protocol version corrected from `5.0` to `6.0` (matches ProtoV6)
- Release workflow: binary naming inside archives follows Terraform convention
- Makefile: `build` target now injects version via `-ldflags`
- Makefile: `install` target uses dynamic version and platform detection

### Security
- Bumped `golang.org/x/net` from 0.17.0 to 0.38.0

## [0.6.0] - 2026-04-17

### Added
- Bucket sub-resources: `scality_bucket_acl`, `scality_bucket_encryption`, `scality_bucket_lifecycle`, `scality_bucket_object_lock`, `scality_bucket_policy`, `scality_bucket_replication`
- IAM resources: `scality_iam_policy`, `scality_iam_role`, `scality_iam_role_policy_attachment`
- Unit tests for client layer

### Fixed
- URL path escaping for account names and resource identifiers containing special characters
- CVE mitigations in dependencies

### Changed
- Refactored provider to register all resources through a single `Resources()` method

## [0.5.0] - 2026-03-31

### Added
- `insecure_skip_verify` provider attribute for self-signed TLS certificates

## [0.4.1] - 2026-03-27

### Fixed
- Go formatting issues in source files

## [0.4.0] - 2026-03-27

### Added
- Custom account attributes support
- QA improvements across existing resources

## [0.3.0] - 2026-03-27

### Added
- Console password generation for `scality_console_account`
- Documentation cleanup

## [0.2.0] - 2026-03-23

### Added
- Initial release
- Core resources: `scality_account`, `scality_console_account`, `scality_account_access_key`, `scality_user`, `scality_user_access_key`, `scality_user_policy`, `scality_group`, `scality_group_membership`, `scality_bucket`
- Three-client architecture: IAMClient (SigV4), S3Client (SigV4), ConsoleClient (JWT)
- Multi-platform release workflow (linux/darwin/windows, amd64/arm64)
