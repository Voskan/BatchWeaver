# ADR 0005: Canonical batch request and response

- Status: Accepted
- Date: 2026-07-29

## Context

A native batch provider must map many logical requests to many outcomes without
ambiguity, even when input keys are duplicated or non-comparable, and while
keeping global transport failures distinct from per-item failures.

## Decision

- Each logical request carries an opaque `RequestID` (a non-zero `uint64`) scoped
  to a single `BatchRequest`. Providers echo the same IDs back.
- `BatchRequest[K]` and `BatchResponse[V]` are immutable by convention: they copy
  input on construction and return copies from accessors, so callers cannot
  mutate stored data.
- `Outcome[V]` encodes exactly one of success, not-found, or failure; the
  ambiguous found-plus-error state is rejected by validation.
- A `BatchFunc` returns `(BatchResponse[V], error)`: the error is a global
  provider or transport failure, while per-item failures live in `Outcome.Err`.
- Keys are not required to be `comparable`; response helpers that need a map
  require `comparable` locally, and a callback-based helper supports arbitrary
  keys.
- `ValidateAgainst` checks a response against the exact set of request IDs,
  detecting missing, unexpected, and duplicate IDs.

## Consequences

- Duplicate keys are handled correctly because outcomes are keyed by request ID,
  not by the input key.
- Providers and future runtime code have a precise, reflection-free contract.

## Alternatives considered

- **Positional-only mapping.** Rejected; it is fragile when providers reorder or
  omit results.
- **A single error channel for both global and per-item errors.** Rejected; it
  loses the distinction the runtime needs.
