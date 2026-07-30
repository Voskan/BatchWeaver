# Materialize and revert

Materialization writes a plan's transformed files into the working tree. It is
explicit and reversible.

## Materialize

```bash
batchweaver transform materialize <plan-id>
```

BatchWeaver verifies every source precondition, backs up the originals, and writes
the transformed files atomically. It prints the materialization ID and the revert
command. Materialization fails if any source file changed since planning.

## Revert

```bash
batchweaver transform revert <materialization-id>
```

Restores the original files. If a file was edited after materialization, it is
reported as a conflict and left untouched — revert never overwrites your edits by
default.

## Recover

```bash
batchweaver transform recover
```

Inspects incomplete materializations and reports their state. Recovery is
idempotent.

## Safety

Backups live under the ignored `.batchweaver/backups/` directory. See
[materialization and recovery](../architecture/materialization-and-recovery.md).
