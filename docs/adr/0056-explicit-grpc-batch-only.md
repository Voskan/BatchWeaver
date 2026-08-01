# ADR 0056: Explicit gRPC batch protocols only

- Status: Accepted
- Date: 2026-07-30

## Context

BatchWeaver must not invent a remote batch method that a service does not expose.

## Decision

- gRPC batching requires an explicitly declared batch method binding (scalar method, batch method, request key, batch requests field, response mode, response key, optional per-item status).
- No server-side batch method is generated and no protobuf wire contract is changed.
- Bindings are validated strictly and rejected when incomplete.

## Consequences

Only services that expose a compatible batch RPC are batched.
