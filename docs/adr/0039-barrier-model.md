# ADR 0039: Batching barrier model

- Status: Accepted
- Date: 2026-07-29

## Context

Pending batched reads must not be reordered across observable side effects.

## Decision

- A closed barrier vocabulary (transaction commit, scope end, lock boundary, unknown side effect, and others) identifies points where pending operations must flush.
- An explicit bridge.Flush/Barrier API flushes the active scope; automatic barrier insertion is gated by proof and plan and is conservative.
- barrier inspect reports the barrier kinds that flush pending reads.

## Consequences

Reordering across observable effects is prevented, and barriers are explainable.
