# Transformation strategies

Eligibility is always attached to a named strategy. A strategy is a precise
future transformation shape with its own required-obligation set. The proof
engine decides eligibility; it does not perform any transformation.

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

`wave-candidate`, `direct-only`, and `adapter-deferred` are reserved for future
stages and recursive traversal classification.

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
