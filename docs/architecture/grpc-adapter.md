# gRPC adapter

The gRPC adapter binds explicit unary scalar-to-batch RPCs and defines the
metadata, call-option, and status policies for coalescing. The concrete grpc-go
client/bufconn integration is deferred in this build (offline dependency); the
binding model and policies are implemented and tested.

## Explicit batch binding

Batching requires an explicitly declared batch method
([ADR 0056](../adr/0056-explicit-grpc-batch-only.md)): scalar method, batch
method, request key, batch requests field, response mode (keyed or positional),
response key, and an optional per-item status field. Bindings are validated
strictly (`GRPCBinding.Validate`). No server-side batch method is generated.

## Metadata and call options

Metadata keys are classified (`ClassifyMetadata`) as must-equal, partition,
merge, or forbidden ([ADR 0057](../adr/0057-grpc-metadata-partitioning.md)).
Authorization, credential, tenant, and routing keys partition so different
callers never share a batch; tracing keys may merge; unknown keys partition
conservatively. Status codes, messages, and details are preserved per item and at
the batch level.

## Streaming

Client-, server-, and bidirectional-streaming batching requires an explicit
multiplexed contract with per-message correlation IDs and a bounded lifecycle
state machine ([ADR 0058](../adr/0058-explicit-stream-contracts.md)).

## CLI

```bash
batchweaver grpc inspect --scalar=/s/GetUser --batch=/s/BatchGetUsers \
  --key=user_id --response-key=user_id
```
