# Contributing

Thanks for your interest in improving the Scality Terraform provider.

## Reporting issues

Open a GitHub issue with a clear description, the provider version, and the
shortest config that reproduces the problem. Include `TF_LOG=DEBUG` output if
the issue is a wire-level error.

For security issues, see [SECURITY.md](SECURITY.md) — please do not open a
public issue.

## Development setup

You'll need:

- Go 1.25+
- OpenTofu (preferred) or Terraform
- Access to a Scality cluster for acceptance tests

Build and install the provider locally:

```bash
make install VERSION=0.4.1-dev
```

This drops the binary at
`~/.terraform.d/plugins/registry.terraform.io/cmrh/scality/<VERSION>/<OS>_<ARCH>/`.

For interactive development, configure dev_overrides in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "cmrh/scality" = "/path/to/your/built/binary/dir"
  }
  direct {}
}
```

With dev_overrides active, `tofu init` is **not needed** (and won't work) —
go straight to `tofu plan`. No `.terraform.lock.hcl` is needed either.

## Tests

Unit tests run without infrastructure:

```bash
make test
```

Acceptance tests provision real Scality resources and require credentials in
your environment:

```bash
export SCALITY_ENDPOINT=...
export SCALITY_ACCESS_KEY=...
export SCALITY_SECRET_KEY=...
make testacc
```

Always add or extend acceptance tests when adding a resource or behavior.

## Style

- `make fmt` and `make lint` must pass.
- Comments: one short line max. Skip if the name is self-explanatory.
- For documentation, prefer "always do X to ensure Y" framing over "if you hit
  problem Z, fix it with X".

## Submitting changes

1. Fork the repo and create a topic branch.
2. Commit with descriptive messages.
3. Open a PR against `main` with a summary of the change and a test plan.
4. Make sure CI is green.

## Architectural conventions

The provider follows the AWS-provider modular pattern: bucket configuration
aspects (`scality_bucket_policy`, `scality_bucket_lifecycle`,
`scality_bucket_encryption`, etc.) are separate resources from
`scality_bucket`. Don't propose folding them back into a kitchen-sink resource
— for one-stop UX, write a wrapper module instead.

### Resource-lifecycle contract

Terraform assumes `Read(Apply(config)) == config` and that a second apply of the
same config is a no-op. The framework does not enforce this; when Create,
Update, and Read map desired-state ↔ API independently, they drift and the only
symptom is "plan not empty after apply". The entire v1.1.2 bug-fix release was
this one class of defect. Every resource must honor:

1. **One mapping, shared by Create and Update.** The desired-state → API-call
   translation lives in a single helper both call. Divergent Create/Update
   logic is the primary smell.
2. **Read preserves, API overrides only on a genuine change.** For an Optional
   attribute, Read keeps the configured value unless the API reports a real
   difference — not a cosmetic one (e.g. casing) and not lagging read-replica
   state (e.g. quota, custom attributes).
3. **Define what `null` means per Optional attribute** — "unmanaged" vs. "set to
   off" — and make Read honor it. A tri-state bool without that decision is the
   next versioning bug.
4. **No `RequiresReplace` on a field the API can mutate in place**, and never on
   a credential-minting resource (replace silently rotates root keys).
5. **Validate co-dependent / mutually-exclusive attributes in `ValidateConfig`**
   (plan-time), never mid-apply.

### Resource audit checklist

Run against every new or changed resource:

- [ ] Create and Update produce the same API calls for the same desired state.
- [ ] Read round-trips every managed attribute (apply, then plan is empty).
- [ ] Re-applying the same config is a no-op (idempotent).
- [ ] `terraform import` reconstructs state that matches an applied resource
      (covered by `ImportStateVerify`).
- [ ] Every Optional attribute's `null` semantics are decided and Read honors them.
- [ ] No one-way / destructive state transitions (Suspended-not-Unversioned,
      replace-on-mutable-field) are introduced.
- [ ] Co-dependent attributes fail at plan via `ValidateConfig`, not mid-apply.
