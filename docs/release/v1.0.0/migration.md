# Prerelease-to-v1 Migration Plan

The supported prerelease baselines are `v0.1.0-beta.1`, `v0.1.0-beta.2`, and
`v0.1.0-beta.3`. The beta.1-to-beta.2 upgrade changes no schema or ABI; it fixes
installed CLI version discovery. Beta.3 changes only the release-asset layout
so complete public downloads can be verified directly.

This plan is executable. `internal/release/migration_test.go` exercises the
contract from every published prerelease to the `v1.0.0` candidate built by this
working tree, using a per-prerelease configuration fixture under
`internal/release/testdata/migration/`. The suite verifies that schema 1
configuration still loads, validates, and yields a semantic digest and a
non-empty operation catalog; that each migration inventory is well-formed and
rejects cache or artifact reuse; that a prerelease workload profile is rejected
when the bridge ABI or configuration digest changes, accepted when nothing
changed, and reported stale rather than authoritative when aged; and that a
lowered call site still executes the original scalar function when no runtime
bound operation is installed, which is the documented rollback target.

## Required migration sequence

1. record the installed CLI, extension, config schema, bridge ABI, and artifact
   schema versions;
2. disable active adaptive changes and materialized transformations;
3. back up configuration and BatchWeaver-owned manifests;
4. install the candidate version without overwriting the previous binary;
5. migrate configuration through a dry run and review the diff;
6. discard incompatible analysis, proof, transform, profile, daemon, and editor
   caches rather than treating them as valid;
7. regenerate typed bridges and transformation plans;
8. run scan, proof, overlay build, overlay tests, race tests, and editor doctor;
9. materialize only after review and preserve the backup manifest;
10. verify rollback to scalar/direct mode.

## Compatibility rules

- Configuration schema 1 is accepted only when validation passes.
- `v1alpha1` proof and transformation artifacts are never promoted to stable
  evidence without regeneration.
- A bridge ABI mismatch is a hard error.
- Profiles with incompatible schema, ABI, config digest, operation digest, or
  toolchain identity are rejected.
- Daemon/LSP version mismatch requires restart or upgrade, not silent fallback.

## Downgrade

The supported rollback target is direct scalar execution using the prior source
and materialization backup. Reusing newer proof, generated code, profile, or
cache artifacts with an older tool is unsupported. The operational procedure is
in [rollback](../rollback.md) and
[upgrade/downgrade/uninstall](../upgrade-downgrade-uninstall.md).

## Test matrix

Every published prerelease has its own fixture and executes on every run of
`go test ./internal/release`:

| Surface | Coverage |
| --- | --- |
| Configuration schema 1 | loads, validates, digests, non-empty catalog |
| Migration inventory | well-formed; rejects cache and artifact reuse |
| Bridge ABI | mismatch is a hard incompatibility |
| Workload profiles | rejected on ABI/config change; stale when aged |
| Scalar rollback | lowered call site runs the original scalar function |
| Plan/fixture agreement | every named prerelease has a fixture |

Generated bridges, transformation plans, adapter manifests, and editor settings
are regenerated rather than migrated, which the inventory contract enforces by
requiring `discard_caches` and `regenerate_artifacts`. Materialization
backup/revert is covered by the transformation suite in `internal/transform`.
