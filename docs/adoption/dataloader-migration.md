# DataLoader Migration

BatchWeaver can coexist with a DataLoader while teams evaluate explicit
operation contracts. Do not stack BatchWeaver batching around an already batched
call: identify the existing scope, key, cache, error, and dispatch boundaries,
then choose one owner for coalescing.

Map DataLoader keys and results to a typed BatchWeaver operation/provider. Verify
duplicate keys, ordering, not-found values, per-item errors, cancellation,
authorization partitions, and resolver-wave scope. Compare with the same dataset
and methodology, not a synthetic best case. Retain the original loader behind a
feature flag until shadow verification and rollback pass.
