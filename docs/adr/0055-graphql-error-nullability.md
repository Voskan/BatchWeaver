# ADR 0055: GraphQL error and nullability preservation

- Status: Accepted
- Date: 2026-07-30

## Context

GraphQL returns partial data with typed errors and null propagation.

## Decision

- Batched execution preserves field paths (alias-aware), error messages/paths/extensions, and per-field vs operation errors; per-item errors are never collapsed into one operation error.
- Null propagation for non-null fields follows GraphQL execution semantics and is covered by contract verification.
- List ordering and duplicate-parent field instances remain distinct.

## Consequences

Response shape, errors, and nullability match non-batched execution.
