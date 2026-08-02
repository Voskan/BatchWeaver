# Backend Adapter Limitations

This stage delivers the adapter SDK, exact-key PostgreSQL synthesis over the
standard library, the Redis cluster slot algorithm, and contract verification —
all dependency-free and hermetically tested.

## Supported

- Versioned adapter manifests and a closed capability vocabulary with deterministic
  digests and no global mutable registry.
- A narrow, real exact-key SQL parser (no regex, never panics) and PostgreSQL
  batch synthesis, with exact rejections for every unsupported construct.
- A typed, reflection-free runtime SQL provider that maps rows by ordinal,
  preserving order, duplicates, and `sql.ErrNoRows`, verified with a fake driver.
- Redis CRC-16 cluster slot computation, hash-tag handling, and slot grouping.
- A scalar/batch contract-verification harness with a deterministic artifact.
- CLI: `adapter list | inspect | explain | verify | doctor`.

## Deferred (blocked offline; contracts ready)

- The concrete **pgx v5** and **go-redis v9** client bindings are not compiled in
  because their dependency closures are unavailable with the module proxy
  disabled. Their manifests are marked `deferred`, their capabilities defined, and
  the client-agnostic logic (SQL synthesis, mapping, Redis slots) is implemented
  so the bindings are thin additions once the dependencies are available.

## Not implemented (out of scope)

- Composite-key synthesis (infrastructure present; retained as `deferred`), joins,
  LIMIT/OFFSET, and any write synthesis.
- Automatic SQL discovery from source call sites (the parser/synthesis operate on
  a provided query via `adapter inspect`); wiring discovery into the transform
  pipeline is a later step.
- Generated row-decoder code generation (the runtime provider takes a typed
  decoder; emitting decoders as source is deferred).
- Real database/Redis integration tests behind an opt-in build tag.

## Diagnostics

Adapter codes use the `BW6xxx` range to avoid colliding with the proof stage's
`BW5xxx`; this deviates from the prompt's illustrative codes deliberately.

## Non-guarantees

No arbitrary SQL transformation, no automatic writes, no GraphQL/gRPC fusion, no
distributed batching. Unsupported queries are always rejected conservatively with
an exact diagnostic.
