# ADR 0054: Selection-dependent partitioning

- Status: Accepted
- Date: 2026-07-30

## Context

Two resolver calls may share a key but request different sub-selections.

## Decision

- Resolver calls are partitioned by a normalized, alias-independent selection digest when the provider's result depends on the selection.
- A field is never returned to a caller that did not request it, even when another caller in the same wave did.
- Incompatible selection-dependent providers are rejected with a diagnostic.

## Consequences

Selection semantics and field authorization are preserved across batched resolvers.
