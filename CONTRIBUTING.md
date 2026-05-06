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
`~/.terraform.d/plugins/registry.terraform.io/scality/scality/<VERSION>/<OS>_<ARCH>/`.

For interactive development, configure dev_overrides in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "scality/scality" = "/path/to/your/built/binary/dir"
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
