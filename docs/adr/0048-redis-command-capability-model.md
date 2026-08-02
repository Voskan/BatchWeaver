# ADR 0048: Redis MGET/HMGET/pipeline capability model

- Status: Accepted
- Date: 2026-07-29

## Context

Redis batching must map explicit scalar commands to explicit batch commands, not fuse arbitrary commands.

## Decision

- GET maps to MGET, HGET to HMGET, and otherwise-independent commands to pipelining.
- Arbitrary Redis command fusion and Lua synthesis are out of scope.
- The concrete go-redis v9 provider implements slot-safe MGET, per-hash HMGET, and explicit pipelining over public client APIs.

## Consequences

Redis batching is explicit and truthful about round trips.
