# Static loop prefetch

Static loop prefetch is BatchWeaver's first transformation strategy. It rewrites a
certified read-only slice/array loop so that keys are collected once, a single
batch call replaces the per-iteration scalar calls, and the original loop body is
replayed in source order.

## Supported shape

```go
for _, item := range sliceOrArray {
    value, err := receiver.Scalar(ctx, key(item))
    if err != nil {
        return ..., err
    }
    use(item, value)
}
```

Requirements, all verified from the certificate and the concrete resolved types:

- the range is over a slice, array, or pointer to array;
- the scalar call is the first body statement, an `:=` assignment binding a value
  and an error;
- it is immediately followed by `if err != nil { return ... }`;
- the operation is certified read-only and eligible for `static-loop-prefetch`;
- the batch provider has the ordered, global-error signature
  `func(context.Context, []K) ([]V, error)`.

Any deviation is a conservative skip with a reason.

## What the transformation preserves

- source iteration order and key evaluation order;
- receiver and context expressions (used verbatim);
- the first scalar error and early return (the batch's global error is returned at
  the same point the scalar loop would have failed);
- Go 1.26 loop-variable semantics and the original loop body after the scalar call.

## What changes

Only the call shape: one batch call replaces N scalar calls. A backend may process
items the scalar path would not have reached after an earlier error; this is
allowed only because the operation is certified read-only and this non-guarantee is
recorded on the certificate.

## Out of scope for this stage

Map range, channel range, integer range, writes, deduplication, canonicalization,
conditional scalar calls, and early-exit forms beyond the supported error guard.
See [limitations](../limitations/prompt-06.md) and
[ADR 0032](../adr/0032-static-loop-prefetch-first-strategy.md).
