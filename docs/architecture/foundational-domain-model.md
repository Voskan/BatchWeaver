# Foundational Domain Model

This document describes the contracts introduced in the foundational phase. They
are data and type safety only: nothing here intercepts calls, schedules work, or
performs batching.

## Package dependency direction

```text
diagnostics            (no BatchWeaver imports)
   ↑
operation              (imports diagnostics)
   ↑
batchweaver (root)     (imports operation)

operation + diagnostics
   ↑
config                 (imports operation, diagnostics, internal/config*)
   ↑
internal/cli           (imports config, operation, diagnostics)
```

The direction is one-way and acyclic, enforced by `arch_test.go`.

## Operation spec

`operation.Spec` is the central immutable-by-convention value describing an
operation: its `ID`, `Semantics`, optional scalar/batch `Symbol`s, and the
result, partition, scheduler, deduplication, retry, and fallback contracts.
`NewSpec` applies defaults and validates, returning a `*ValidationError` that
carries a full diagnostic collection. `Spec.Canonical()` gives a deterministic
representation used for hashing, excluding source positions.

## Semantic policies

Semantics bundle kind, effect, idempotency, ordering, atomicity, and whether
retry, deduplication, and cross-scope batching are permitted. Named constructors
(`ReadOnly`, `IdempotentWrite`, …) produce valid defaults, and validation
rejects impossible combinations (for example retry on a non-idempotent write, or
process-scope batching without isolation dimensions).

## Canonical request and response

The root package provides `BatchRequest[K]`, `BatchItem[K]`, `Outcome[V]`, and
`BatchResponse[V]`. These are immutable by convention, defensively copy their
inputs, key outcomes by opaque `RequestID`, and keep global errors distinct from
per-item errors. See [ADR 0005](../adr/0005-canonical-batch-request-response.md).

## Typed declarations

`FunctionDeclaration` and `MethodDeclaration` connect a spec to concrete
implementations. They perform no global registration; the canonical shape is a
package-level `var` from `MustDeclareMethod`, which a future analyzer can
discover statically. See [ADR 0008](../adr/0008-declarations-without-global-registration.md).

## Future integration points

- A compiler analyzer will discover declarations and eligible call sites.
- A runtime will consume specs to coalesce, schedule, and distribute batched work.
- Adapters will depend on stable extension interfaces rather than internal types.
