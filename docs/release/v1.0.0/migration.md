# Prerelease-to-v1 Migration Plan

No public prerelease exists yet, so an upgrade from a downloadable version
cannot be executed. This plan defines the migration contract that must be tested
after beta publication and before stable v1.

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
profiles, materialization backup/revert, and scalar rollback. At present this
matrix is blocked because there is no published source version.
