# ADR 0048: Redis MGET/HMGET/pipeline capability model

- Status: Accepted
- Date: 2026-07-29

## Context

Redis batching must map explicit scalar commands to explicit batch commands, not fuse arbitrary commands.

## Decision

- GET maps to MGET, HGET to HMGET, and otherwise-independent commands to pipelining.
- Arbitrary Redis command fusion and Lua synthesis are out of scope.
- The concrete go-redis client binding is contract-defined but deferred offline; the client-agnostic slot and mapping logic is implemented and tested.

## Consequences

Redis batching is explicit and truthful about round trips.
