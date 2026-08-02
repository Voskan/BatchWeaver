# Adapter SDK

The adapter SDK (`internal/adapter`) is BatchWeaver's backend integration layer.
It models adapters declaratively and separates two concerns
([ADR 0042](../adr/0042-adapter-sdk-separation.md)):

- **compile-time** — manifests and capabilities, SQL parsing and exact-key
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

- `database/sql` — **ready**. Exact-key PostgreSQL read synthesis over the
  standard library, ordered/sparse result mapping, transaction partitioning,
  chunking, and semantic verification.
- `pgx` — **deferred**. Contract defined; the concrete pgx v5 client binding is
  not compiled into this build because its dependency closure is unavailable
  offline. See [limitations](../limitations/backend-adapters.md).
- `redis` — **deferred client**. Cluster hash-slot grouping and MGET/HMGET/
  pipeline mapping logic are implemented and tested; the concrete go-redis client
  binding is deferred offline.

## Runtime provider

A runtime provider implements the typed batch contract
(`Execute(ctx, BatchRequest[K]) (BatchResponse[V], error)`), so adapter providers
plug directly into the runtime and generated bridge.
