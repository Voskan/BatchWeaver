# gRPC batch RPCs

BatchWeaver calls an explicitly declared batch RPC in place of many unary calls.
It never invents a server-side batch method.

A binding declares the scalar and batch methods, the request key, the batch
requests field, and how the batch response correlates to requests (keyed by a
response key, or strictly positional), plus an optional per-item status field.

Callers are partitioned so security- and routing-sensitive metadata
(authorization, credentials, tenant, region) is never merged across callers.
gRPC status codes, messages, and details are preserved per item and at the batch
level. Streaming batching requires an explicit multiplexed, correlated contract.
