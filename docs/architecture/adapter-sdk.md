# Adapter SDK

The adapter SDK (`internal/adapter`) is BatchWeaver's backend integration layer.
It models adapters declaratively and separates two concerns
([ADR 0042](../adr/0042-adapter-sdk-separation.md)):

- **compile-time** — manifests and capabilities, SQL parsing and exact/composite-key
  synthesis, binding, and diagnostics. This side never opens a backend connection.
- **runtime** — typed batch providers, chunking, result mapping, and contract
  verification. This side never imports `go/ast` or `go/types`.

## Manifests and capabilities

Each adapter has a versioned `Manifest` (schema `batchweaver.adapter/v1alpha1`)
with a closed capability vocabulary and a deterministic content digest that
excludes host-specific data. There is no mutable global registry; the built-in
manifest set is immutable and callers receive copies. Unknown capabilities are
rejected. See [the adapter manifest reference](../reference/adapter-manifest.md).

## Built-in adapters

- `database/sql` — **ready**. Exact/composite-key PostgreSQL read synthesis with
  one bounded at-most-one join, ordered/sparse result mapping, transaction
  identity, content-addressed plans, overlay generation, chunking, and verification.
- `pgx` — **ready**. `adapters/pgxv5` executes parameterized array
  queries through a caller-owned connection, transaction, or pool and maps rows
  by request ordinal.
- `redis` — **ready**. `adapters/redisv9` provides cluster-slot-safe MGET,
  per-hash HMGET, and explicit pipeline providers over go-redis v9.

## Runtime provider

A runtime provider implements the typed batch contract
(`Execute(ctx, BatchRequest[K]) (BatchResponse[V], error)`), so adapter providers
plug directly into the runtime and generated bridge.
