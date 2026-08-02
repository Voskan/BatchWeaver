# Prerelease-to-v1 Migration Plan

The supported prerelease baselines are `v0.1.0-beta.1`, `v0.1.0-beta.2`, and
`v0.1.0-beta.3`. The beta.1-to-beta.2 upgrade changes no schema or ABI; it fixes
installed CLI version discovery. Beta.3 changes only the release-asset layout
so complete public downloads can be verified directly. An upgrade to v1 cannot
be executed until a future v1 candidate exists. This plan defines the migration
contract that must pass before stable publication.

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

## Test matrix required before approval

Every actually published beta or RC must have its own fixture that covers
configuration, generated bridges, caches, adapter manifests, editor settings,
profiles, materialization backup/revert, and scalar rollback. The beta fixture
can now be frozen; execution remains blocked until a v1 candidate exists.
