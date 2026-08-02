# Backend Adapter Limitations

This build includes the adapter SDK, exact-key PostgreSQL synthesis over the
standard library, concrete pgx v5 and go-redis v9 runtime providers, Redis
cluster-slot safety, and contract verification.

## Supported

- Versioned adapter manifests and a closed capability vocabulary with deterministic
  digests and no global mutable registry.
- A narrow, real exact-key SQL parser (no regex, never panics) and PostgreSQL
  batch synthesis, with exact rejections for every unsupported construct.
- A typed, reflection-free runtime SQL provider that maps rows by ordinal,
  preserving order, duplicates, and `sql.ErrNoRows`, verified with a fake driver.
- Redis CRC-16 cluster slot computation, hash-tag handling, and slot grouping.
- `adapters/pgxv5`: caller-owned pgx connection/transaction/pool execution,
  deterministic chunking, ordinal correlation, duplicates, and missing results.
- `adapters/redisv9`: cluster-slot-safe MGET, per-hash HMGET, and explicit typed
  pipelines with per-item error mapping.
- A scalar/batch contract-verification harness with a deterministic artifact.
- CLI: `adapter list | inspect | explain | verify | doctor`.

Integration coverage uses pgxmock for the pgx.Rows contract and miniredis through
the real go-redis client. A live PostgreSQL or Redis Cluster deployment remains
an environment-specific acceptance test, not a claim made by the hermetic suite.

## Not implemented (out of scope)

- Composite-key synthesis (infrastructure present; retained as `deferred`), joins,
  LIMIT/OFFSET, and any write synthesis.
- Automatic SQL discovery from source call sites (the parser/synthesis operate on
  a provided query via `adapter inspect`); wiring discovery into the transform
  pipeline is a later step.
- Generated row-decoder code generation (the runtime provider takes a typed
  decoder; emitting decoders as source is deferred).
- Containerized PostgreSQL and multi-node Redis Cluster acceptance in every
  supported CI environment.

## Diagnostics

Adapter codes use the `BW6xxx` range to avoid colliding with the proof stage's
`BW5xxx`; this deviates from the prompt's illustrative codes deliberately.

## Non-guarantees

No arbitrary SQL transformation, no automatic writes, no GraphQL/gRPC fusion, no
distributed batching. Unsupported queries are always rejected conservatively with
an exact diagnostic.
