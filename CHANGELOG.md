# Changelog

All notable changes to the Scality Terraform Provider are documented in this file.

## [Unreleased]

## [1.0.0] - 2026-06-24

First stable release. API surface unchanged from v0.6.4; the bump reflects the security/CI baseline below and a stable commitment to semver going forward.

### Added
- Dependabot config for Go modules (daily) and GitHub Actions (weekly). (#4)
- CodeQL workflow on PR + weekly schedule. (#4)
- `govulncheck` step in `test.yml`. (#4)
- `gitleaks` workflow on PR + push to main; `.gitleaks.toml` allowlists test fixtures. (#13)
- `gosec` workflow on PR + push to main; 11 false-positive findings annotated inline with `// #nosec <ID> -- <reason>`. (#13)

### Security
- Bumped `go` directive from 1.25.0 to 1.25.11, picking up stdlib patches across `crypto/tls`, `crypto/x509`, `net/http`, `net/url`, `encoding/asn1`, `encoding/pem`, `html/template`, `os`, and `net/textproto`. (#4)
- Bumped `golang.org/x/net` from v0.49.0 to v0.56.0 (covers GO-2026-4918, GO-2026-5026). (#4)

### Changed
- `test.yml` and the build/release-manifest jobs in `release.yml` run on `ubuntu-latest`. `acceptance.yml` stays on `self-host-cm-tf-provider` and is invoked from `release.yml` via `workflow_call`. (#2)
- Workflow-level `permissions: contents: read` on `test.yml`, `acceptance.yml`, and `release.yml`. `release.yml`'s `build` and `release-manifest` jobs declare `contents: write` per-job for artifact upload and signing. (#4)
- `SECURITY.md` points to GitHub Private Vulnerability Reporting. (#4)

## [0.6.4] - 2026-06-18

### Fixed
- `scality_account.Read` preserves prior state's `custom_attributes` instead of overwriting from `GetAccount`. Eliminates phantom in-place drift from eventual-consistency races on `UpdateAccountAttributes`. Out-of-band edits via the Vault Console are reconciled by the next apply that touches the attribute. (scality/terraform-provider-scality#72)

## [0.6.3] - 2026-06-17

Tagged, never shipped — acceptance gate held when the polling-based fix didn't survive a multi-replica API. Polling helper removed; actual fix in v0.6.4. (scality/terraform-provider-scality#72)

## [0.6.2] - 2026-06-17

### Fixed
- YAML syntax error in `acceptance.yml` that blocked v0.6.1's acceptance run. (scality/terraform-provider-scality#70)
- `acceptance.yml` is a reusable workflow gating `build` in `release.yml`; release artifacts no longer publish when acceptance fails. `workflow_dispatch` on `acceptance.yml` still runs ad-hoc.

## [0.6.1] - 2026-06-17

### Fixed
- `scality_account` no longer phantom-replaces on plan after a `custom_attributes` write. `quota_max` gains `int64planmodifier.UseStateForUnknown()`; `Read` preserves prior `custom_attributes` when the API returns empty. (scality/terraform-provider-scality#68)

### Changed
- `acceptance.yml` uses `${TF_ACC_TERRAFORM_PATH}` for the runtime binary, hoisted to job-level `env`. Swap to terraform with a one-line change. (scality/terraform-provider-scality#68)

## [0.6.0] - 2026-06-17

### Added
- Secret-free import mode: when `SCALITY_ACCOUNT_ACCESS_KEY` + `SCALITY_ACCOUNT_SECRET_KEY` are set, `terraform import` reads creds from env and the import ID carries only the resource identity. Legacy `ACCESS_KEY:SECRET_KEY:IDENTITY` form unchanged when env vars are unset. Covers all per-account resources. (scality/terraform-provider-scality#58)

### Changed
- Replaced internal endpoint and example admin credentials in `QUICKSTART.md`, `examples/main.tf`, and provider schema descriptions with `https://vault.example.com` placeholders. (scality/terraform-provider-scality#56)
- `terraform import` on `scality_user_access_key` and `scality_account_access_key` emits a warning that the secret is unrecoverable from the API; recommends creating a new managed key. Docs gain an "Adoption: rotate, don't import" section. (scality/terraform-provider-scality#64)

### Fixed
- `scality_bucket_lifecycle` no longer phantom-updates on rules with explicit zero-value fields (`prefix = ""`, `expiration_days = 0`, etc.). `clientRulesToModel` matches each Read rule against prior state by `id` and preserves the user's empty-vs-null representation. (scality/terraform-provider-scality#62)
- JSON policy documents (`scality_user_policy.policy_document`, `scality_iam_policy.policy_document`, `scality_bucket_policy.policy`, `scality_iam_role.assume_role_policy`, and matching data sources) use `jsontypes.Normalized` for semantic equality. Whitespace/key-order differences no longer surface as phantom plan diffs. Adds `terraform-plugin-framework-jsontypes` dependency. (scality/terraform-provider-scality#60)

## [0.5.0] - 2026-05-28

### Added
- `region` provider attribute (defaults to `us-east-1`, also `SCALITY_REGION`). Overrides SigV4 signing region for the IAM and S3 clients. (scality/terraform-provider-scality#46)
- `## Data Sources` section in `docs/index.md`. (scality/terraform-provider-scality#48)
- `data.scality_account` and `data.scality_bucket` — singular lookup by name.
- `data.scality_accounts` and `data.scality_buckets` — paginated enumeration. Drill down to per-entry detail via `for_each` over the singular.
- `data.scality_user[s]`, `data.scality_group[s]`, `data.scality_iam_policy[ies]`, `data.scality_iam_role[s]`. Singulars look up by name; plurals enumerate with pagination. (scality/terraform-provider-scality#50)
- `IAMClient.ListUsers`, `ListGroups`, `ListPolicies` (scope `Local`), `ListRoles` with corresponding list-entry types. (scality/terraform-provider-scality#50)
- Eight new `docs/data-sources/*.md` pages. (scality/terraform-provider-scality#50)

### Changed
- `examples/*.tf` version pins bumped to `~> 0.4`. (scality/terraform-provider-scality#48)
- `interface{}` → `any` in `AccountCreateResponse.AccountData.CustomAttributes` (lint hygiene).
- IAM client returns typed `*APIError` from `doSignedRequest`; callers use `client.IsNotFound(err)` via `errors.As` instead of `strings.Contains`. Hardens against upstream error-wording drift. (scality/terraform-provider-scality#52)

### Fixed
- `scality_console_account.Read` probes Vault via `IAMClient.GetAccount` to detect out-of-band deletion. Requires IAM admin credentials configured; Console-only configurations retain prior state-preserve behavior. (scality/terraform-provider-scality#54)

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
