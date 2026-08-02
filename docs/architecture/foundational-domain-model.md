# Foundational Domain Model

This document describes the contracts at the base of BatchWeaver. Constructing
them is side-effect free; the compiler and runtime consume them in later layers.

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
package-level `var` from `MustDeclareMethod`, which the analyzer discovers
statically. See [ADR 0008](../adr/0008-declarations-without-global-registration.md).

## Integration points

- The compiler analyzer discovers declarations and eligible call sites.
- The runtime consumes specs to coalesce, schedule, and distribute batched work.
- Adapters depend on versioned contracts rather than compiler implementation types.
