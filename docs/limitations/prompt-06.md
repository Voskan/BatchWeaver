# Transformation limitations (this stage)

The transformation stage delivers one production-quality strategy end to end.
This document records what it does not yet do, so results are not overinterpreted.

## Supported

- Static loop prefetch for a certified read-only `for _, v := range sliceOrArray`
  loop whose first body statement is `value, err := receiver.Scalar(ctx, key)`
  followed by `if err != nil { return ... }`, over an ordered, global-error batch
  provider `func(context.Context, []K) ([]V, error)`.
- Deterministic plans, unified diffs, source maps, overlay build/test/run, and
  atomic materialization with backup, revert, and recovery.

## Not supported yet

- Map range, channel range, and integer range loops.
- Writes and aggregations (only read-only operations are transformed).
- Deduplication and canonicalization of keys.
- Keyed, sparse, or per-item-error batch result modes (only ordered global-error).
- Conditional scalar calls and early-exit forms beyond the supported error guard
  (`break`, `continue`, labels, `defer`, `recover`, panic recovery).
- Interface and generic receivers beyond a uniquely resolved concrete receiver.
- Composition of overlapping or nested transformations in one file (at most one
  transformation per file in this stage).
- Multi-file import planning beyond what the supported shape needs; the key type
  must already be importable under its own package name.

## Deferred infrastructure

- Transformation cache LRU eviction and size bounds (plans are stored but not
  evicted automatically).
- SARIF output, CRLF normalization, and cross-platform path edge cases beyond the
  slash-relative deterministic identities already used.
- A full crash-recovery state machine with tests at every persistence transition;
  `transform recover` reports state but does not auto-repair.

## Non-guarantees

Every certificate and plan records that a backend may process read-only items the
scalar path would not have reached after an earlier error; this is safe only
because the operation is certified read-only. Provider performance is not
guaranteed. Correctness assumes the analyzed program is free of data races.

This stage does not generate SQL or backend batch providers, does not batch
writes, does not lower goroutine or errgroup fan-out, and does not claim universal
performance improvements.
