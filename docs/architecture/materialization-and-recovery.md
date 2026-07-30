# Materialization and recovery

Materialization writes a plan's transformed files into the working tree. It is an
explicit, atomic, reversible operation, separate from the non-mutating overlay
path.

## Materialize

1. Verify every source precondition: the current file digest must equal the
   plan's recorded original digest. Any drift aborts with `BW3701`.
2. Create a backup directory under `.batchweaver/backups/<materialization-id>/`
   and write a backup manifest.
3. For each file: copy the original to a content-addressed backup, then write the
   transformed content atomically (temp file + fsync + rename), updating the
   manifest after each committed file.
4. Finalize the manifest state to `committed`.

## Revert

1. Load the backup manifest.
2. For each committed file, verify the current content still matches the recorded
   transformed digest. If it was edited after materialization, report a conflict
   and do not overwrite it.
3. Otherwise restore the original atomically and verify the restored digest.

Revert never overwrites user edits by default.

## Recover

`batchweaver transform recover` inspects incomplete manifests and reports their
state (`writing`, `committed`, `recovery-required`, …). Recovery is idempotent and
never mutates source files on its own.

## Backups are local

Backup manifests and objects live under the ignored `.batchweaver/` directory and
are never committed. See [ADR 0031](../adr/0031-atomic-materialization-and-revert.md).
