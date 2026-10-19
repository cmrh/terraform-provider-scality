## Summary

What does this PR change and why?

## Type of change

- [ ] Bug fix
- [ ] New resource / data source
- [ ] Enhancement to existing resource
- [ ] Documentation
- [ ] Refactoring / dependency update

## Test plan

- [ ] `make fmt` and `make lint` pass
- [ ] `make test` passes
- [ ] `make testacc` passes against a real cluster (note which env)
- [ ] Documentation updated under `docs/`
- [ ] CHANGELOG entry added under `[Unreleased]`

### For a new or changed resource (see CONTRIBUTING.md § The LifecycleTest gate)

- [ ] Runs through `acctest.LifecycleTest` (round-trip, idempotency, import-verify)
- [ ] Validated with unknown inputs (a `dynamic` block / variable-driven config)
- [ ] Every attribute exercised as null, empty, and set
- [ ] Out-of-band deletion of the object (and its parent) handled
- [ ] Any new assumption about server behavior verified with a probe, not assumed
- [ ] For a bug fix: the regression test was confirmed to fail on the old code

## Related issues

Closes #
