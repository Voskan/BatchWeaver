# ADR 0047: pgx native batch versus synthesized query

- Status: Accepted
- Date: 2026-07-29

## Context

pgx.Batch pipelines multiple statements; it is not the same as reducing N reads to one SQL query.

## Decision

- Exact-key synthesis (one SQL statement) is distinct from pipelining (many statements in one round trip).
- The adapter model exposes distinct execution modes and never reports a pipeline as a single database query.
- The concrete pgx v5 client binding is contract-defined but deferred in this build because its dependency closure is unavailable offline.

## Consequences

Users get accurate execution-mode reporting; the pgx binding is a thin, well-scoped addition.
