# Releasing

This document describes how to cut a release of the Scality Terraform Provider.

## Prerequisites

- Push access to the `scality/terraform-provider-scality` repository
- GPG key registered as a GitHub Actions secret (see [GPG Setup](#gpg-setup) below)
- Test environment configured on the self-hosted runner (see [Test Environment](#test-environment) below)

## Release Flow

### 1. Prepare

- Ensure `main` is green: unit tests, formatting, and vet all pass in CI
- Run acceptance tests against real infrastructure: trigger the **Acceptance Tests** workflow manually via `Actions > Acceptance Tests > Run workflow`
- Update `CHANGELOG.md` — move items from `[Unreleased]` into a new version section

### 2. Internal / Pre-release

Tag with a pre-release suffix to build and test artifacts without publishing to any registry:

```bash
git tag v0.7.0-rc1
git push origin v0.7.0-rc1
```

This produces a full set of release artifacts (binaries, checksums, signed SHA256SUMS) but marks the GitHub release as **pre-release**. Both the Terraform and OpenTofu registries ignore pre-releases.

Valid pre-release suffixes: `-rc1`, `-beta.1`, `-dev.1` (any of `-rc`, `-beta`, `-dev`).

### 3. Public Release

Once the pre-release has been verified, tag the same commit without a suffix:

```bash
git tag v0.7.0
git push origin v0.7.0
```

This creates a full GitHub release that registries will discover and serve to users.

### 4. Verify

After the release workflow completes:

- [ ] GitHub release page has `.zip` archives for all 5 platforms
- [ ] `SHA256SUMS` file is present and contains all 5 checksums
- [ ] `SHA256SUMS.sig` file is present (GPG signature)
- [ ] Manifest JSON has correct version and `"protocols": ["6.0"]`
- [ ] Release is **not** marked as pre-release (for public releases)

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **Patch** (`v0.7.1`): bug fixes, dependency updates
- **Minor** (`v0.8.0`): new resources, new attributes, non-breaking changes
- **Major** (`v1.0.0`, `v2.0.0`): breaking changes to resource schemas or provider configuration

All tags must include the patch number (use `v0.7.0`, not `v0.7`).

## What the Release Workflow Does

```
Tag push
  └─ test job: unit tests, go vet, gofmt
       └─ build job (5 platforms in parallel):
            - Builds binary with version injected via -ldflags
            - Creates zip archive: terraform-provider-scality_{version}_{os}_{arch}.zip
            - Generates per-archive SHA256 checksum
            - Uploads archive + checksum to GitHub release
                └─ release-manifest job:
                     - Combines per-archive checksums into SHA256SUMS
                     - Signs SHA256SUMS with GPG key
                     - Generates provider manifest JSON
                     - Uploads SHA256SUMS, signature, and manifest
```

## GPG Setup

The release workflow signs checksums with a GPG key. This is required for publishing to the Terraform and OpenTofu registries.

### One-time setup

1. **Generate a key** (RSA 4096 — registries do not support ECC):

   ```bash
   gpg --full-generate-key
   # Select: (1) RSA and RSA
   # Key size: 4096
   # Expiry: 0 (does not expire) or 2y
   # Name/email: use the org identity
   ```

2. **Add secrets to GitHub** (Settings > Secrets and variables > Actions):

   ```bash
   # Private key — paste output as GPG_PRIVATE_KEY secret
   gpg --armor --export-secret-keys YOUR_EMAIL

   # Passphrase — paste as GPG_PASSPHRASE secret
   ```

3. **Register the public key with registries** (when ready for public publishing):

   ```bash
   gpg --armor --export YOUR_EMAIL
   ```

   - **Terraform Registry**: registry.terraform.io > Sign in > Settings > GPG Keys
   - **OpenTofu Registry**: registry.opentofu.org > Sign in > Settings > GPG Keys

### Key rotation

When rotating the GPG key:

1. Generate a new key
2. Update both GitHub secrets (`GPG_PRIVATE_KEY`, `GPG_PASSPHRASE`)
3. Add the new public key to both registries (keep the old key — it's needed to verify past releases)

## Test Environment

Acceptance tests run on the self-hosted runner against real Scality infrastructure. The runner must have a `~/.scality-test.env` file that the acceptance workflow sources before running tests.

### `~/.scality-test.env`

```bash
# Use local tofu binary instead of downloading terraform
export TF_ACC_TERRAFORM_PATH="/usr/bin/tofu"

# Lab1 (source / primary)
export SCALITY_ENDPOINT="http://<lab1-ip>:8080"
export SCALITY_ACCESS_KEY="<admin-access-key>"
export SCALITY_SECRET_KEY="<admin-secret-key>"
export SCALITY_CONSOLE_ENDPOINT="http://<lab1-ip>:8080"
export SCALITY_CONSOLE_USERNAME="admin"
export SCALITY_CONSOLE_PASSWORD="<admin-password>"

# Lab2 (destination — CRR tests only)
export SCALITY_DEST_ENDPOINT="http://<lab2-ip>:8080"
export SCALITY_DEST_ACCESS_KEY="<dest-admin-access-key>"
export SCALITY_DEST_SECRET_KEY="<dest-admin-secret-key>"
export SCALITY_DEST_CONSOLE_ENDPOINT="http://<lab2-ip>:8080"
export SCALITY_DEST_CONSOLE_USERNAME="admin"
export SCALITY_DEST_CONSOLE_PASSWORD="<dest-admin-password>"
```

### Variable reference

| Variable | Used by | Purpose |
|----------|---------|---------|
| `TF_ACC_TERRAFORM_PATH` | test framework | Path to tofu/terraform binary — prevents the test harness from downloading one |
| `SCALITY_ENDPOINT` | all tests | S3/IAM API endpoint for the primary cluster |
| `SCALITY_ACCESS_KEY` / `SECRET_KEY` | all tests | Admin credentials for IAM operations on the primary cluster |
| `SCALITY_CONSOLE_ENDPOINT` | console tests | Management console endpoint (often the same as `SCALITY_ENDPOINT`) |
| `SCALITY_CONSOLE_USERNAME` / `PASSWORD` | console tests | Console admin login |
| `SCALITY_DEST_*` | CRR tests only | Same set of variables for the destination cluster; CRR tests are skipped if these are unset |

### Notes

- The file must use `export` so `set -a` / `source` in the workflow picks up the variables.
- `TF_ACC_TERRAFORM_PATH` must point to an OpenTofu (or Terraform) binary already installed on the runner. Setting it in `~/.bashrc` alone is **not sufficient** — GitHub Actions uses `bash --noprofile --norc`, so only variables from `~/.scality-test.env` are available.
- CRR tests require two independent Scality clusters. If `SCALITY_DEST_*` variables are not set, CRR tests are skipped rather than failed.

## Local Development Builds

For local testing without a release:

```bash
# Build with version injection
make build VERSION=v0.7.0-dev

# Or use dev_overrides in ~/.terraformrc / tofu config:
provider_installation {
  dev_overrides {
    "registry.terraform.io/scality/scality" = "/path/to/built/binary/directory"
  }
  direct {}
}
```

## Registry Publishing

Both registries auto-discover releases from GitHub — no manual upload needed. Requirements:

- GitHub repository must be **public**
- Repository name must follow `terraform-provider-{name}` convention
- Release artifacts must include signed SHA256SUMS
- GPG public key must be registered with the registry

The provider is published under `scality/scality` in both registries. Users consume it as:

```hcl
terraform {
  required_providers {
    scality = {
      source  = "scality/scality"
      version = "~> 0.7.0"
    }
  }
}
```
