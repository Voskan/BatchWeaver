# ADR 0012: Key strategy and collision-safe deduplication

- Status: Accepted
- Date: 2026-07-29

## Context

Generic keys may not be comparable and may contain mutable data. Deduplication
must never merge different keys, even under hash collisions.

## Decision

A `KeyStrategy[K]` provides Clone, Hash, Equal, and EstimateBytes. Built-in
strategies cover comparable, string, and []byte keys, using hash/maphash (no
reflection in the hot path). Deduplication and memoization bucket by hash but
always confirm identity with Equal.

## Alternatives considered

- Requiring `comparable` keys: rejected; too restrictive.
- Using fmt.Sprintf as a hash: rejected; slow and ambiguous.

## Consequences

Non-comparable keys are supported; byte keys are defensively cloned; a collision
test with a constant hash proves separate items.

## Security and concurrency

Keys are cloned into owned storage; raw keys never appear in diagnostics.
Strategies are safe for concurrent use.

## Compatibility

New API.
