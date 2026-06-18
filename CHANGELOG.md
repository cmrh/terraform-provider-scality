# Changelog

All notable changes to the Scality Terraform Provider are documented in this file.

## [Unreleased]

## [0.6.4] - 2026-06-18

### Fixed
- `scality_account.Read` no longer overwrites prior state's `custom_attributes` from `GetAccount`. The Vault IAM API serves reads from load-balanced replicas with non-uniform write propagation, so a post-apply `GetAccount` can land on a replica that hasn't seen a recent `UpdateAccountAttributes` and return either empty or the prior values — both surface as phantom in-place drift. `Read` now populates `custom_attributes` from the API only on the initial-read path (state null/unknown, e.g. post-import); when state already holds a value, it's preserved. Trade-off: out-of-band edits via the Vault Console are not surfaced on `terraform refresh`. The next apply that touches the attribute reconciles state to config — standard plan-vs-config diff is still shown before that apply runs. Same trade-off pattern as #62. Supersedes the v0.6.3 polling attempt, which was correct against a single-node fixture but couldn't bridge the load-balanced multi-replica case. (#72)

## [0.6.3] - 2026-06-17

v0.6.3 was tagged but no binaries were published — the new release-pipeline gate held when acceptance tests caught that the polling-based fix below didn't work on a load-balanced API. Superseded by the next release. The polling helper (`IAMClient.WaitForCustomAttributes`) and its tests have been removed.

### Fixed
- *(attempted, did not ship)* `scality_account` Create and Update polled `GetAccount` until `UpdateAccountAttributes` was visible, intended to fix the eventual-consistency race observed in v0.6.2's acceptance run. The poll succeeded against one replica but the framework's subsequent post-apply `Read` could land on a different (still-stale) replica — same phantom drift. See [Unreleased] for the actual fix. (#72)

## [0.6.2] - 2026-06-17

### Fixed
- YAML syntax error in `.github/workflows/acceptance.yml` that prevented the v0.6.1 acceptance run from starting. The diagnostic step's `run:` value mixed a double-quoted scalar with trailing content on the same line (`run: "${TF_ACC_TERRAFORM_PATH}" version`), which YAML rejects after the closing `"`. Rewritten as a block scalar so the shell still quotes the binary path defensively. (#70)
- Release binaries are no longer published when acceptance tests fail. Previously `.github/workflows/release.yml` and `.github/workflows/acceptance.yml` ran in parallel on a `v*` tag push, so `build` + `release-manifest` uploaded artifacts (binaries, signed `SHA256SUMS.sig`, manifest) regardless of whether the acceptance suite passed — exactly how v0.6.0 and v0.6.1 shipped with broken acceptance runs. `acceptance.yml` is now a reusable workflow invoked by `release.yml` as a gate between `test` and `build`; tag-push artifacts only publish when acceptance is green. `workflow_dispatch` on `acceptance.yml` still works for ad-hoc validation runs.

## [0.6.1] - 2026-06-17

### Fixed
- `scality_account` no longer reports a phantom replacement on the next plan after a `custom_attributes` write when `GetAccount` races the attribute persist. Two changes: (a) `quota_max` (`Optional+Computed+RequiresReplace`) now carries `int64planmodifier.UseStateForUnknown()` so it doesn't get re-marked Unknown when any other attribute drifts (matches the pattern used by the resource's other Computed attributes); (b) `Read` preserves prior state's `custom_attributes` when the API returns an empty map, mirroring the `bucket_lifecycle` Read fix from #62. Trade-off: out-of-band attribute deletions via the Vault Console are no longer surfaced on `terraform refresh`. Surfaced on a new self-hosted GitHub Actions runner whose latency exposed the eventual-consistency window; passes locally and on the previous runner. (#68)

### Changed
- `.github/workflows/acceptance.yml` now references the acceptance runtime binary through `${TF_ACC_TERRAFORM_PATH}` everywhere (including a new "Acceptance runtime version" diagnostic step), with the value hoisted to a job-level `env:` block. Swap the path to `/usr/bin/terraform` in one spot to validate against upstream Terraform — no step-level edits needed. (#68)

## [0.6.0] - 2026-06-17

### Added
- Env-gated, secret-free import mode for every per-account resource. When `SCALITY_ACCOUNT_ACCESS_KEY` and `SCALITY_ACCOUNT_SECRET_KEY` are both set, `terraform import` reads the credentials from the environment and the import ID carries only the resource identity (`BUCKET_NAME`, `USERNAME:POLICY_NAME`, `ROLE_NAME:POLICY_ARN`, etc.) — no more credentials in `import {}` blocks, shell history, or CI logs. When the env vars are unset, the legacy `ACCESS_KEY:SECRET_KEY:IDENTITY` form continues to parse byte-for-byte unchanged. Covers `bucket`, `bucket_encryption`, `bucket_lifecycle`, `bucket_policy`, `bucket_object_lock`, `bucket_replication`, `user`, `user_policy`, `user_access_key`, `group`, `group_membership`, `account_access_key`, `iam_policy`, `iam_role`, `iam_role_policy_attachment`. Pure additive change — no schema change, no state migration. (#58)

### Changed
- Replaced internal lab endpoint and example admin credentials baked into `QUICKSTART.md`, `examples/main.tf`, and the `endpoint` / `console_endpoint` provider schema descriptions with the generic `https://vault.example.com` placeholder pattern already used in `README.md`. Pre-public-release hygiene. (#56)
- `terraform import` on `scality_user_access_key` and `scality_account_access_key` now emits a warning explaining that `secret_access_key` / `secret_key` won't be populated from the IAM API (it's only available at creation) and that the supported way to bring an existing key under Terraform management is to create a new managed key and retire the manually-created one. The import still succeeds — the warning is non-blocking. Docs pages for both resources gained an "Adoption: rotate, don't import" subsection covering the same rotation procedure and the 4-key-per-principal cap. (#64)

### Fixed
- Phantom in-place update on `scality_bucket_lifecycle` when a rule sets `prefix = ""` (or `expiration_days = 0`, `expiration_date = ""`, `noncurrent_version_expiration_days = 0`, `abort_incomplete_multipart_upload_days = 0`). The Read path collapsed every zero-valued field to `null`, but the prior state stored the user's explicit empty value, so the next refresh diff-flapped between `null` and `""`/`0` forever. `clientRulesToModel` now matches each Read rule against prior state by `id` and preserves the user's empty-vs-null representation when the API returned the zero value. No schema change; no state migration. (#62)
- Phantom plan diff on JSON policy documents. `scality_user_policy.policy_document`, `scality_iam_policy.policy_document`, `scality_bucket_policy.policy`, and `scality_iam_role.assume_role_policy` (plus the matching `data.scality_iam_policy` and `data.scality_iam_role` attributes) now use the `jsontypes.Normalized` custom type. JSON-equivalent values (whitespace, key order) compare equal via `StringSemanticEquals`, so `terraform plan` after a clean apply no longer reports `~ policy_document = jsonencode( # whitespace changes )`. No schema change on the wire and no state migration — the on-disk representation is still a string; only in-memory equality changed. Adds dependency `github.com/hashicorp/terraform-plugin-framework-jsontypes`. (#60)

## [0.5.0] - 2026-05-28

### Added
- `region` provider attribute (defaults to `us-east-1`, also settable via `SCALITY_REGION`) — overrides the SigV4 signing region used by the IAM and S3 clients. Previously hardcoded. (#46)
- `## Data Sources` section in `docs/index.md` listing `scality_account`, `scality_accounts`, `scality_bucket`, `scality_buckets`. They were already documented per-page but absent from the provider overview. (#48)
- `data.scality_account` data source. Looks up an existing account by name and exposes `id`, `email_address`, `quota_max`, `custom_attributes`, `arn`, `canonical_id`, `create_date`. Does not expose `access_key`/`secret_key` (IAM API returns those only at creation).
- `data.scality_bucket` data source. Looks up an existing bucket by name within an account and exposes `id`, `arn`, `versioning`, `object_lock_enabled`, `tags`.
- `data.scality_accounts` data source. Lists all accounts in the cluster (paginated under the hood). Returns a `accounts` list of objects with `id`, `name`, `email_address`, `arn`, `canonical_id`, `create_date`, `quota_max`. No `custom_attributes` per entry — use `data.scality_account` with `for_each` for drill-down.
- `data.scality_buckets` data source. Lists all buckets owned by the supplied account credentials. Returns a `buckets` list of objects with `name`, `arn`, `creation_date`. No versioning/tags/object-lock per entry — use `data.scality_bucket` with `for_each` for drill-down.
- `docs/data-sources/scality_account.md`, `docs/data-sources/scality_bucket.md`, `docs/data-sources/scality_accounts.md`, `docs/data-sources/scality_buckets.md`.
- `data.scality_user`, `data.scality_users`, `data.scality_group`, `data.scality_groups`, `data.scality_iam_policy`, `data.scality_iam_policies`, `data.scality_iam_role`, `data.scality_iam_roles` data sources. Singular look up by name (errors at plan time when not found); plurals enumerate the account with pagination handled internally. Singular drill-down from a plural via `for_each` mirrors the pattern shipped for `account`/`bucket`. (#50)
- `IAMClient.ListUsers`, `ListGroups`, `ListPolicies` (scope `Local`), `ListRoles` client methods with corresponding `UserListEntry`, `GroupListEntry`, `PolicyListEntry`, `RoleListEntry` types. Pagination follows the `ListAccounts` marker + `IsTruncated` pattern. (#50)
- `docs/data-sources/scality_user.md` and 7 sibling pages covering all eight new data sources; `docs/index.md` Data Sources table extended. (#50)

### Changed
- `examples/*.tf` version pins bumped to `~> 0.4` (was `0.2.1` in two files, missing in `multiple-accounts.tf`). (#48)
- Replaced legacy `interface{}` style with `any` in `AccountCreateResponse.AccountData.CustomAttributes` (lint hygiene; no functional change).
- IAM client now returns a typed `*APIError{Code, Message, StatusCode}` from `doSignedRequest` when the response body parses as an XML `ErrorResponse`; callers (`GetUser`, `GetUserPolicy`, `GetGroup`, `GetRole`, `DeleteRole`, `DetachRolePolicy`, `GetManagedPolicy`, `DeleteManagedPolicy` paths) now use `client.IsNotFound(err)` via `errors.As` instead of `strings.Contains(err.Error(), "NoSuchEntity")`. No user-visible behavior change; refactor hardens against upstream error-wording drift. (#52)

### Fixed
- `scality_console_account` Read now probes Vault via `IAMClient.GetAccount` to surface out-of-band deletion. When the provider is configured with IAM admin credentials (the common case), `terraform refresh` removes deleted accounts from state instead of silently preserving them. Console-only provider configurations retain the previous state-preserve behavior; drift detection is best-effort. (#54)

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
