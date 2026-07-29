# Deduplication and Memoization

The runtime has two distinct reuse mechanisms. They are separate layers with
different safety rules.

## In-flight deduplication

When enabled, the first request for a given (operation, partition, key) creates
an in-flight item; later compatible requests join it as independent waiters. The
provider receives **one** logical item, and every waiter receives an equivalent
outcome.

- Identity is (operation ID, partition, canonical key). Deduplication never
  crosses partitions.
- Hash collisions never merge distinct keys: a hash accelerates lookup, but the
  key strategy's `Equal` is authoritative.
- Cancelling one waiter affects only that waiter; the provider work continues
  while any waiter remains active.
- Completed entries are removed before later calls are considered in flight.
- In-flight deduplication is **not** enabled for non-idempotent writes; binding
  validation rejects that combination.

## Scope memoization

When enabled, a completed read result is cached within the scope and reused by a
later call for the same (partition, key) in that scope, avoiding a provider call
entirely.

- Memoization is opt-in and permitted only for read operations that are not
  freshness-dependent; binding validation rejects unsafe kinds.
- Errors are never memoized. Not-found results are memoized only when the
  negative-result policy is enabled.
- Entries are bounded by count (and optionally bytes) with insertion-order
  eviction, and are released when the scope closes.
- Memoization is strictly scope-local; it never crosses scopes.

## Difference summary

| | In-flight deduplication | Scope memoization |
| --- | --- | --- |
| Reuses | Overlapping concurrent work | A completed result later in the scope |
| Lifetime | Until the item completes | Until the scope closes |
| Default | Per operation spec | Opt-in, read-only |
| Writes | Rejected | Rejected |
