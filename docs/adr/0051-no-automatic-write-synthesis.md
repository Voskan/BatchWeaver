# ADR 0051: No automatic write synthesis

- Status: Accepted
- Date: 2026-07-29

## Context

Automatic write batching is unsafe without explicit, proven contracts.

## Decision

- This stage synthesizes only read queries. INSERT/UPDATE/DELETE/MERGE and DDL are rejected.
- Writes may only be batched through a fully explicit, declared provider contract, not synthesized.
- Unknown transport outcomes for reads map to errors, never silently to missing data.

## Consequences

Automatic transformation stays within safe read semantics.
