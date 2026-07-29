# Result Semantics

The result contract defines how a native batch provider's outcomes map back to
the individual scalar callers. It is a data contract in this phase; the runtime
that enforces it comes later.

## Result mode

- **ordered** — outcomes correspond to request items by request ID or position.
- **keyed** — outcomes are associated with canonical keys.
- **sparse** — absent keys are omitted and resolved through missing-result
  semantics.

## Missing results

`not-found`, `error`, `zero-value`, or `contract-violation`. `zero-value`
requires explicit opt-in because it can hide missing data. Sparse results require
an explicit missing behavior other than `contract-violation`. When a config
operation omits `missing`, the default is `contract-violation` so unmodeled gaps
surface loudly.

## Errors

The error mode is `per-item`, `global`, or `mixed`. A `BatchFunc` returns a
global error separately from per-item `Outcome.Err`, so transport failures and
individual failures never blur together.

## Duplicates and unexpected results

Duplicate outcomes default to a contract violation (or `first`/`last`/`custom`).
A provider returning an unrequested key or request ID is a contract violation by
default; there is no ignore-by-default mode.

## Why request IDs

Outcomes are keyed by an opaque `RequestID`, not by the input key, so duplicate
input keys still map to distinct callers. See
[../architecture/foundational-domain-model.md](../architecture/foundational-domain-model.md).
