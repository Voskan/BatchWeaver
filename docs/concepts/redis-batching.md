# Redis batching

BatchWeaver maps explicit scalar Redis commands to explicit batch commands; it
never fuses arbitrary commands ([ADR 0048](../adr/0048-redis-command-capability-model.md)).

- `GET` maps to `MGET`.
- `HGET` maps to `HMGET` per hash key.
- otherwise-independent commands map to a pipeline (fewer round trips, still
  multiple commands — reported accurately).

## Cluster safety

Multi-key commands must not cross cluster hash slots. The adapter computes slots
with CRC-16/XMODEM and hash-tag handling (`{tag}`), and groups keys by slot so
each command stays within a slot
([ADR 0049](../adr/0049-redis-cluster-slot-partitioning.md)). The client remains
responsible for routing and MOVED/ASK handling.

## Client status

The client-agnostic slot and mapping logic is implemented and tested. The concrete
go-redis client binding is deferred in this build (offline dependency); see
[limitations](../limitations/prompt-08.md).
