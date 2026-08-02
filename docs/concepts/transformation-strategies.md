# Transformation strategies

Eligibility is always attached to a named strategy. A strategy is a precise
transformation shape with its own required-obligation set. The semantic proof
engine decides Go call-site eligibility; the SQL parser and synthesis contract
provide the corresponding fail-closed evidence for generated SQL bindings.

## Strategy vocabulary

- `static-loop-prefetch` — evaluate proven-safe keys, perform one batch call, and
  reconstruct scalar observations in original loop order.
- `static-ordered-prefetch` — like loop prefetch but uses an ordered per-item
  contract to preserve source-order first-error semantics.
- `static-sibling-fusion` — combine straight-line sibling calls while preserving
  lexical evaluation and observation order.
- `existing-fanout-coalescing` — coalesce calls that are already concurrent
  without introducing additional concurrency.
- `runtime-scope-coalescing` — route scalar calls through the runtime without
  hoisting or creating concurrency.
- `exact-key-sql-synthesis` — generate a typed Go query constant from a validated
  exact-key PostgreSQL read.
- `composite-key-sql-synthesis` — generate parallel-array SQL for contiguous,
  parameterized composite key components.
- `bounded-join-sql-synthesis` — generate one qualified INNER/LEFT join only
  when the synthesis plan records an explicit at-most-one contract.

`wave-candidate`, `direct-only`, and `adapter-deferred` are reserved for future
stages and recursive traversal classification.

SQL strategies generate a content-addressed file in the standard transformation
IR. The file is formatted, type-checked through an in-memory overlay, source
mapped with the `sql-synthesis` role, and kept non-mutating until an explicit
materialization command. Modified SQL or contract metadata invalidates the plan
digest before database I/O.

## Why runtime coalescing can be allowed when static hoisting is rejected

Static prefetch moves key evaluation and the batch call to a single collection
point at compile time, so it must prove that no observable effect, loop-carried
dependency, or early-exit reconstruction problem stands in the way. Runtime
coalescing leaves the source evaluation exactly where it is and only routes the
call through the runtime; it therefore does not require the order, key
independence, or early-exit obligations. A candidate blocked by an observable
barrier or a call-derived key can still be eligible for runtime coalescing, while
a candidate with an unresolved target or a write operation is ineligible for both
because those obligations are required by every strategy.

## Never introduce concurrency

No strategy in the core engine runs sequential scalar calls concurrently.
`existing-fanout-coalescing` applies only to calls that are already concurrent and
preserves the original concurrency envelope. The `BW-PROOF-CONC-001` obligation
records that the engine introduces no new concurrency.
