# ADR 0031: Explicit, atomic materialization with backup and revert

- Status: Accepted
- Date: 2026-07-29

## Context

Writing transformed code into the source tree is irreversible without a backup
and dangerous if interrupted. It must be opt-in, atomic, and reversible.

## Decision

- Materialization verifies every source precondition (current digest equals the
  plan's original digest) before writing anything.
- It records a backup manifest, copies each original to a content-addressed
  backup, then writes each transformed file atomically (temp file + rename),
  updating the manifest after each committed file.
- Revert restores originals only when the current file still matches the recorded
  transformed digest; a post-materialization edit is reported as a conflict and
  never overwritten.
- `transform recover` inspects incomplete manifests idempotently.

## Consequences

Source mutation is reversible and crash-aware, and user edits made after
materialization are protected.
