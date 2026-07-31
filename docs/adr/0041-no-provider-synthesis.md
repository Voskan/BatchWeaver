# ADR 0041: No automatic provider synthesis in runtime lowering

- Status: Accepted
- Date: 2026-07-29

## Context

Runtime lowering must not invent backend batch APIs.

## Decision

- Every lowered operation still requires an explicitly declared, compatible batch provider or adapter binding.
- The transformer generates the typed bridge and call rewrite only; it never synthesizes SQL, query, or backend batch code.

## Consequences

Correctness depends on declared contracts, and backend synthesis remains out of scope until a later stage.
