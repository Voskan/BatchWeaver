# Operation Contracts

An operation contract describes how a scalar operation relates to its batch
equivalent and under what conditions batching is safe. In the foundational phase
these are **data contracts** only; no execution behavior is implemented.

## Anatomy

An `operation.Spec` combines:

- an **ID** (`users.get`) — a stable, dot-separated identifier;
- optional **symbols** — Go references to the scalar and batch implementations,
  used by configuration-based operations;
- **semantics** — kind, effect, idempotency, ordering, atomicity, and the coarse
  permissions for retry, deduplication, and cross-scope batching;
- a **result contract** — how outcomes map back to callers;
- a **partition contract** — how callers are isolated into separate batches;
- **scheduler, deduplication, retry, and fallback policies** — declarative
  configuration for the future runtime.

## Validation

Validation collects every problem as a diagnostic rather than stopping at the
first. It enforces cross-field invariants, for example: non-idempotent writes
cannot enable retry; transaction-bound operations must partition by transaction;
deduplication is only allowed when the semantics permit it.

## Current versus future

- **Now:** you can declare, validate, canonicalize, and catalog specs, and load
  them from configuration.
- **Later:** the runtime will use these contracts to coalesce and schedule work,
  and the compiler will discover and transform eligible call sites.
