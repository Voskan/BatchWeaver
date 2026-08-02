# ADR 0015: In-flight deduplication versus scope memoization

- Status: Accepted
- Date: 2026-07-29

## Context

Two different reuse mechanisms are useful, with different safety profiles, and
conflating them would be unsafe.

## Decision

In-flight deduplication collapses overlapping concurrent work for the same
(operation, partition, key) and lasts until the item completes. Scope
memoization reuses a completed read result later within a scope and lasts until
scope close. They are independent layers; both are rejected for writes; memoization
is read-only and does not cache errors by default.

## Alternatives considered

- A single unified cache: rejected; different lifetimes and safety rules.

## Consequences

Clear semantics and validation; memoization is bounded and scope-local.

## Security and concurrency

Neither crosses partitions; memoization never crosses scopes; writes are
rejected. Deduplication state is owned by the coordinator goroutine.

## Compatibility

New API; consistent with operation policies.
