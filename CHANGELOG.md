# Changelog

All notable changes to the Scality Terraform Provider are documented in this file.

## [Unreleased]

### Added
- `scality_iam_group_policy_attachment` attaches a managed policy to a group. Groups with attached policies can't be deleted, so `destroy` can now remove them and their account. (#115)

### Fixed
- Account-tier IAM resources (`scality_account_access_key`, `scality_user`, `scality_user_policy`, `scality_user_access_key`, `scality_group`, `scality_group_membership`, `scality_iam_policy`, `scality_iam_role`, `scality_iam_role_policy_attachment`, `scality_iam_group_policy_attachment`) no longer get replaced when `account_access_key` / `account_secret_key` change. Switching to another key of the same account, or between explicit keys and the provider's `assume_role`, now updates in place. Each resource records its account, and credentials of a different account are refused instead of moving the resource there and leaving the original behind. (#118)
- JSON documents (`scality_iam_policy.policy_document`, `scality_user_policy.policy_document`, `scality_bucket_policy.policy`, `scality_iam_role.assume_role_policy`) no longer plan a change when the configured JSON differs from the stored one only in formatting, such as after import or when reformatting the config. Previously this planned an update, which for `scality_iam_policy` created a new policy version, and for `scality_iam_role` replaced the role. (#117)
- `scality_bucket` import now records Object Lock, versioning, and tags. Import left them null, so a config with `object_lock_enabled = true` planned a replacement of the bucket, and configured versioning or tags planned an in-place update. Only settings that are on server-side are recorded, so a config that omits them still plans clean. `object_lock_enabled` now treats unset and `false` as the same value, so switching between them no longer forces replacement. (#108)
- `scality_bucket` with `object_lock_enabled = true` no longer fails to apply `versioning = true` with `InvalidBucketState`. Object Lock already enables versioning and the server rejects any versioning change; Update now skips the call when versioning is already enabled. (#108)
- `scality_account` import with `external_account_id` configured no longer plans a replacement. Vault uses the external ID as the account ID and returns no separate field, so import left it null. `external_account_id` is now computed from the account ID when unset; existing states pick it up on refresh with no plan change. (#109)
- `scality_console_account` import no longer plans a replacement of the account. Import recorded only the name, so the required `email` and any configured `quota` forced replacement. Read now fills them from the Vault IAM API, and import requires the provider's IAM admin credentials. `generate_random_password` and `password_length` can't be read back; setting them when previously unset is now recorded in place (with a warning that no password was generated) instead of replacing the account. (#109)
- `scality_console_account` docs no longer say that every argument forces replacement; they now note the exception for the password inputs added in #109. (#109)
- `scality_console_account` in-place updates no longer plan `created_at`, `password`, `access_key` and `secret_key` as known after apply. Update never changes them; they now keep their state values in the plan. (#113)
- The provider's debug-mode address is now `registry.terraform.io/cmrh/scality`, matching the published namespace. It was still `scality/scality`, so `-debug` reattach did not match configs using `source = "cmrh/scality"`.

## [1.1.6] - 2026-10-19

### Fixed
- `scality_bucket_object_lock` no longer fails `terraform validate` with a false "Conflicting retention period" when `retention_days`/`retention_years` come from variables. `ValidateConfig` treated an unknown value as set; it now skips the exactly-one check until both values are known. (#103)

### Changed
- `CONTRIBUTING.md` and the pull-request template now document the resource lifecycle as an enforced gate: every resource must run through `acctest.LifecycleTest` (round-trip, idempotency, import-verify) and be tested with unknown, null, and empty inputs, out-of-band deletion, and — for shared-record resources — parallel apply; plus a "verify server behavior, don't assume it" rule. Codifies the lessons from the v1.1.2–v1.1.5 bug-fix work. (#101)
- Added unknown-value `ValidateConfig` unit tests for `scality_bucket`, `scality_bucket_encryption`, and `scality_account`, so the "validators must tolerate unknown inputs" rule is enforced for every validator rather than asserted. (#105)

## [1.1.5] - 2026-10-19

### Fixed
- `scality_bucket` with `tags = {}` (an empty, non-null map) no longer drifts to `null` on refresh. Create saved the empty map to state but Read forced it to null when the server reported no tags, giving a perpetual `+ tags = {}` plan and a no-op tag delete on every apply. Read now maps "no tags" to an empty map so a configured `{}` round-trips. (#90)
- `scality_user_access_key` and `scality_iam_role_policy_attachment` no longer block all plans when their parent (the user or role) is deleted outside Terraform. Listing a deleted parent's children returns `NoSuchEntity`, which Read treated as a hard error; Read now removes the resource from state so the next plan recreates it. (#95)
- `scality_bucket` no longer orphans a bucket when a post-creation step fails. If setting versioning or tags failed after the bucket was created, Create returned without saving state, leaving the bucket on the server untracked so the next apply failed with `BucketAlreadyOwnedByYou`. Create now persists state on those failures so Terraform tracks (and on the next apply replaces) the bucket. (#92)

### Added
- Acceptance coverage for previously untested lifecycle paths: multiple `scality_bucket_lifecycle` rules in a non-sorted order (round-trip in configured order) and `expiration_date` (round-trips verbatim) — both confirmed stable against the API. (#90)
- Automatic retry with exponential backoff and jitter for transient failures on S3 and IAM/account requests. Only requests the server provably did not process are retried — network errors, throttling (HTTP 429/503), and the retryable 409 conflict codes (`OperationAborted`, `ConcurrentModification`) — so a non-idempotent create is never duplicated (HTTP 500 is not retried). Retries respect context cancellation. (#93)

### Changed
- `CONTRIBUTING.md` and the pull-request template now document the resource lifecycle as an enforced gate: every resource must run through `acctest.LifecycleTest` (round-trip, idempotency, import-verify) and be tested with unknown, null, and empty inputs, out-of-band deletion, and — for shared-record resources — parallel apply; plus a "verify server behavior, don't assume it" rule. Codifies the lessons from the v1.1.2–v1.1.5 bug-fix work. (#101)
- Added unknown-value `ValidateConfig` unit tests for `scality_bucket`, `scality_bucket_encryption`, and `scality_account`, so the "validators must tolerate unknown inputs" rule is enforced for every validator rather than asserted. (#105)

## [1.1.4] - 2026-10-16

### Fixed
- `scality_bucket_lifecycle` no longer breaks `terraform validate` when `rule` is built from a `dynamic` block over a value that isn't known at validate time (e.g. a module whose rules come from input variables). The `ValidateConfig` added in 1.1.2 (#70) decoded the config into a Go slice, which cannot hold an unknown list; it now reads `rule` as a list and skips validation until the value is known, and ignores rules whose action values are still unknown. Regression in 1.1.2/1.1.3. (#87)

## [1.1.3] - 2026-10-16

### Fixed
- Applying several bucket-configuration resources to the same bucket in parallel (`scality_bucket_policy`, `scality_bucket_encryption`, `scality_bucket_lifecycle`, and the other `scality_bucket_*` resources) no longer silently drops one of the writes. The backend persists all bucket subresources into a single record via read-modify-write, so concurrent writes raced; the provider now serializes mutating requests per bucket within an apply. Users no longer need `-parallelism=1` or a `depends_on` chain to work around it; writes to different buckets still run in parallel. (#84)

### Changed
- `CONTRIBUTING.md` now documents the resource-lifecycle contract (round-trip, Create/Update symmetry, `null` semantics, no replace on mutable fields, plan-time validation) and a per-resource audit checklist, codifying the invariants behind the v1.1.2 bug-fix release. (#80)
- Added a shared `acctest.LifecycleTest` helper (apply → idempotency re-plan → import-verify) enforcing the lifecycle contract at the acceptance gate, and backfilled `ImportStateVerify` for `scality_account` and `scality_bucket_replication` (test-only; `scality_console_account` is excluded because the Console API has no read-back for import to verify against). (#81)

## [1.1.2] - 2026-10-15

### Fixed
- `scality_bucket` no longer suspends versioning on a newly created bucket. `versioning = false` (or unset) now leaves a new bucket **Unversioned** instead of issuing `PutBucketVersioning(Suspended)` — Suspended and Unversioned are distinct S3 states, and Suspended is a one-way door (there is no API back to Unversioned). Versioning is only suspended when turning it off on a bucket that is currently Enabled; removing the attribute entirely now leaves the bucket's versioning untouched (previously it could suspend an Enabled bucket). Reads map both `Suspended` and Unversioned to `false` (Enabled is the only `true` state). (#63)
- `scality_iam_policy` no longer fails with `LimitExceeded` after repeated `policy_document` edits. IAM managed policies cap at 5 versions; Update now prunes the oldest non-default version before creating the new one, so a policy can be edited indefinitely. (#67)
- `scality_account` now changes `quota_max` in place via the Vault `UpdateAccountQuota` API instead of forcing a destructive replacement. Previously a quota change destroyed and recreated the account — a new ARN and root credentials, or a `409` on a non-empty account. `scality_console_account` quota remains replace-only (Console API path). (#68) Read preserves the state quota across refreshes to avoid phantom drift from the IAM API's read-replica lag, matching the `custom_attributes` handling. (#78)
- `scality_bucket_replication` no longer fails with "inconsistent result after apply" when adding a rule to an existing configuration. Update now reads the config back after applying (mirroring Create), and an omitted rule `id` is planned as unknown so the server-assigned value is accepted, including on newly added rules. (#69)
- `scality_account` no longer plans a perpetual replacement when `email_address` contains uppercase letters. The server lowercases the email; Read now keeps the configured casing when it differs only by case. (#71)

### Added
- `scality_bucket` now rejects `object_lock_enabled = true` combined with `versioning = false` at plan time. Object Lock requires versioning, so the combination is contradictory and previously drifted or failed mid-apply; the error now names the fix (set `versioning = true` or omit it). (#64)
- Plan-time validation for co-dependent attributes, catching at plan what previously failed at apply or drifted: `scality_bucket_encryption` requires `sse_algorithm = "aws:kms"` when `kms_master_key_id` is set; `scality_bucket_object_lock` requires exactly one of `retention_days`/`retention_years`; `scality_account` rejects more than 10 `custom_attributes`; `scality_bucket_lifecycle` requires each rule to set at least one action and rejects both `expiration_days` and `expiration_date`. (#70)

## [1.1.1] - 2026-10-12

### Fixed
- `scality_account` data source now retries briefly when an account is found but its custom attributes have not yet propagated across the Vault IAM API's load-balanced read replicas. A same-apply lookup of an account whose attributes were just set could previously return them empty. The genuinely-empty case is unchanged. (#58)

### Security
- Bumped `golang.org/x/net` v0.58.0 → v0.60.0 and the `go` directive to 1.26.9, clearing all reachable advisories reported by govulncheck. The x/net upgrade fixes five HTTP/2 advisories in that module (GO-2026-6617/-6612/-6611/-6610/-6603) and requires `go` ≥ 1.26.0; pinning 1.26.9 (rather than 1.26.0) additionally clears the net/http, net/textproto, and crypto/tls advisories present in the go 1.26.0 standard library. CI reads the version from `go.mod`, so no workflow changes are needed.

## [1.1.0] - 2026-09-18

### Added
- `scality_bucket_logging` resource for S3 server access logging (PUT/GET Bucket Logging). Requires server access logging to be enabled at the cluster level; when it is not, the API returns `501 NotImplemented` and the resource errors with actionable guidance rather than a raw API error. Deleting the resource disables logging. (#54)
- Provider-level `assume_role { role_arn, session_name }` block for delegated cross-account management. A management account's credentials are exchanged once, at provider configuration, for temporary role credentials via STS; per-account resources fall back to those credentials whenever they omit their own `account_access_key`/`account_secret_key`. Use one aliased provider per target account. Credentials are held in memory only and never written to state; automatic refresh is not yet supported. `account_access_key`/`account_secret_key` are now optional on all per-account resources (explicit values still take precedence). STS shares the IAM endpoint and is an account-level operation — the superadmin account-management APIs do not support it. (#52)

## [1.0.2] - 2026-09-16

### Removed
- Dead code with no production callers: `validators.JSONDocument()` (superseded by the `jsontypes.Normalized` type) and `IAMClient.DeleteAccountAccessKey` (superseded by the per-account key-delete path), plus their tests. (#48)

### Changed
- `govulncheck` moved off PR CI. It now blocks in `release.yml` (a release can't ship a reachable vuln) and runs weekly, non-blocking, in `security-scan.yml`. PRs no longer go red when the vulnerability DB updates on unchanged code. (#42)

### Security
- Bumped indirect deps `golang.org/x/text` (v0.38.0 → v0.39.0, GO-2026-5970) and `google.golang.org/grpc` (v1.80.0 → v1.82.1, GO-2026-6061). (#43)
- Bumped the `go` directive from 1.25.12 to 1.25.13, picking up stdlib fixes: GO-2026-6218 (`net/url`), GO-2026-6090 (`crypto/tls`), GO-2026-6088 (`encoding/xml`), GO-2026-5972 (`encoding/asn1`), GO-2026-5026 (`net/http`). (#51)

## [1.0.1] - 2026-07-09

### Fixed
- `scality_account` now persists the generated root access key to state immediately after creation, before setting custom attributes. Previously, if `UpdateAccountAttributes` failed, Create returned without writing the keys — leaving a live, unrecoverable root credential on the account with no record in state. (#18)
- `scality_console_account` no longer fails at apply when `generate_random_password` is unset or false. The Computed `password` attribute is now resolved to null in that case instead of being left unknown. (#19)
- `GetGroup` now paginates. `scality_group_membership` previously truncated the member set at one page (~100 users), causing permanent drift and no-op re-adds on groups larger than that. (#20)
- `scality_group_membership` Delete now skips users that are already gone (`NoSuchEntity`) and continues removing the rest, instead of aborting the loop on the first one. Previously an out-of-band removal of one member left the remaining members in the group while Terraform dropped the resource from state. (#23)
- Release-asset provider manifest now emits the correct schema (`{"version": 1, "metadata": {"protocol_versions": [...]}}`, copied from the repo-root manifest) and its checksum is included in `SHA256SUMS`. The previous hand-built manifest used an invalid schema and was absent from `SHA256SUMS`, which the registry rejected. (#16)

### Changed
- Release workflow hardened. `release.yml` triggers only on tag push (the duplicate `release: [created]` trigger is removed) and carries a `concurrency` group so a tag can't run two racing pipelines. The `build` job now hands archives to the signing job via `actions/upload-artifact`/`download-artifact` (immutable within a run) instead of round-tripping through mutable release assets, and re-verifies every checksum before signing — closing the window where a tampered or stale asset could receive a valid GPG signature. `build` no longer needs `contents: write`. (#22)
- Pinned `govulncheck` (v1.5.0), `gosec` (v2.27.1), and `golangci-lint` (v2.12.2) to explicit versions instead of `@latest`, so a tool release can't red an unrelated PR. A weekly `tool-bumps.yml` workflow watches these pins plus the `go` directive and opens a PR when any is stale. (#31)
- `.gitleaks.toml` no longer exempts all `_test.go` files by path; the placeholder-credential allowlist is anchored to exact values so a real secret in a test file is still flagged. (#31)

### Security
- The Console JWT is no longer written to a predictable file in `os.TempDir()`. The token is now held in memory for the client's lifetime only, removing the symlink/pre-creation and cross-run exposure on shared hosts. (#21)
- The release workflow passes only the `SCALITY_*` secrets to the acceptance job (explicit `secrets:` mapping) instead of `secrets: inherit`, so `GPG_PRIVATE_KEY`/`GPG_PASSPHRASE` are no longer exposed to the self-hosted acceptance runner. (#30)
- Bumped the `go` directive from 1.25.11 to 1.25.12 for the `crypto/tls` fix (GO-2026-5856). (#30)

### Removed
- Untracked stale build artifacts committed at the repo root (`coverage.txt`, `terraform-provider-scality_1.0.0_SHA256SUMS`, `_SHA256SUMS.sig`, `_manifest.json`); `.gitignore` now covers release outputs. (#30)

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
