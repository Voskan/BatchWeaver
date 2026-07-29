# Partitioning (Runtime)

A partition is a batching boundary: requests in different partitions never share
a provider batch. This preserves tenant, authorization, transaction, session,
region, and consistency isolation.

## Partitioner

A binding may supply a `Partitioner[K]`, which runs in the caller goroutine
before enqueue and may read the caller context:

```go
Partitioner: batchruntime.PartitionerFunc[K](func(ctx context.Context, key K) (batchruntime.Partition, error) {
    return batchruntime.PartitionFromStrings(
        tenant.FromContext(ctx),
        auth.FingerprintFromContext(ctx),
    ), nil
})
```

If no partitioner is supplied, all requests use `SinglePartition()`.

## Encoding and equality

Partitions are built from length-delimited components (`PartitionFromStrings`,
`PartitionFromBytes`), so distinct groupings never collide — `["a","b"]` is never
equal to `["ab"]`. Equality is exact over the encoded bytes; a hash only
accelerates lookup. `Partition.String()` returns a redacted, stable-within-process
token derived from a hash, never the raw component values.

## Isolation guarantees

- Different partitions map to different queues and different provider batches.
- Authorization is represented as a fingerprint component, never a raw token.
- Partitioner failure fails the request before any queue state is created.
- The active-partition count is bounded per operation; a partition with no
  queued, in-flight, or memoized work is retired, and cardinality returns to
  zero after a workload completes.

## Security

Because partitions determine which callers share a batch, correct partitioning
is a security boundary: a tenant or authorization fingerprint that is not part of
the partition would allow cross-tenant batching. Include every dimension that
affects authorization or visibility in the partition.
