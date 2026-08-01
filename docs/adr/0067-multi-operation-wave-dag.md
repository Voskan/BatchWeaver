# 67. Multi-operation wave DAG

Date: 2026-08-01

## Status

Accepted

## Context

Batching within one operation is not enough; independent operations can be
co-scheduled to reduce sequential backend stalls.

## Decision

A versioned operation dependency DAG models operations, computations, barriers,
adapter-compound requests, and recursive frontiers, with data, control, barrier,
partition, transaction, and error-order edges. Dispatch groups independent nodes
into waves (co-scheduled, not merged into one batch unless a fusion group is
declared). Waves and the critical path are computed deterministically; a cycle is
a hard error.

## Consequences

Compatible operations run in parallel with a coordinated wait. Fusion is only
used where an adapter explicitly declares a compound capability.
