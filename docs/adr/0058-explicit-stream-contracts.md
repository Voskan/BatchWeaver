# ADR 0058: Explicit multiplexed stream contracts

- Status: Accepted
- Date: 2026-07-30

## Context

Streaming batching is only safe with an explicit correlated envelope.

## Decision

- Client-, server-, and bidirectional-streaming support requires an explicit multiplexed contract where every message carries a logical request ID and per-item status.
- Streams use bounded pending maps and an explicit lifecycle state machine; no unbounded goroutines or global streams.
- Automatic stream replay is disabled unless the protocol declares replay safety and item idempotency.
- Concrete grpc-go/bufconn coverage verifies unary explicit batch RPCs. Streaming remains contract-only until an explicit multiplexed service is supplied and tested.

## Consequences

Streaming remains bounded, correlated, and conservative.
