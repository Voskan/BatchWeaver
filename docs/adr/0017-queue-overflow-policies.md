# ADR 0017: Queue overflow policies

- Status: Accepted
- Date: 2026-07-29

## Context

Bounded queues must have deterministic, correct behavior when full, without
unbounded memory growth.

## Decision

Queues are bounded per operation by items, waiters, bytes, weight, and
partitions. Three overflow policies are implemented: reject (typed QueueFullError
/ PartitionLimitError), scalar fallback (run outside the coordinator), and block
(parked and admitted when capacity frees, respecting caller cancellation). Zero
limit fields mean "no limit" for that dimension; engine and spec defaults supply
bounded values.

## Alternatives considered

- Unbounded queues: rejected; memory-exhaustion risk.
- Silent dropping: rejected; hides failures.

## Consequences

Backpressure is explicit and typed; empty partitions are retired.

## Security and concurrency

Bounded queues and partition limits prevent trivial memory and cardinality
attacks. Overflow decisions are made serially by the coordinator.

## Compatibility

Maps Prompt 02 overflow behavior to runtime policies.
