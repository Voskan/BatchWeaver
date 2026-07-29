# ADR 0014: Provider context and deadline algorithm

- Status: Accepted
- Date: 2026-07-29

## Context

A batch aggregates callers with different contexts and deadlines. Passing one
caller's context to the provider would let a short caller cancel others and leak
caller values.

## Decision

The runtime builds a dedicated batch context from the engine/scope lifecycle. It
carries the latest active caller deadline only when every active waiter has a
deadline, and is cancelled when every active waiter cancels. It does not inherit
arbitrary caller values. Flush timing uses the earliest deadline minus a margin.

## Alternatives considered

- Shortest-deadline provider context: rejected; cancels longer callers.
- Merging caller values: rejected; ambiguous and unsafe.

## Consequences

Short callers never cancel longer ones; providers get predictable deadlines;
caller values do not leak into shared work.

## Security and concurrency

No cross-caller value leakage. Cancellation accounting is done serially by the
coordinator and tested under the race detector.

## Compatibility

New behavior; documented in the cancellation reference.
