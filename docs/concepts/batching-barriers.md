# Batching barriers

A batching barrier is a point across which pending batched reads must not be
reordered. Lowered runtime calls may defer backend work; a barrier flushes that
pending work before an observable side effect.

## Explicit barriers

Insert an explicit barrier before an observable side effect that must see the
results of pending reads:

```go
if err := bridge.Flush(ctx); err != nil {
    return err
}
```

`bridge.Flush` and its alias `bridge.Barrier` flush the active scope's pending
operations and are no-ops when no scope is active.

## Barrier kinds

Pending reads flush at observable barriers, including transaction
begin/commit/rollback, savepoint, session or authorization change, lock
boundaries, channel communication, unknown side effects, and scope end.
`batchweaver barrier inspect` lists these kinds and the runtime-lowered
operations subject to them.

## Automatic insertion

Automatic barrier insertion is gated by proof and the transformation plan and is
conservative: a barrier is inserted only when generated calls could otherwise
cross it and insertion preserves source behavior. See
[ADR 0039](../adr/0039-barrier-model.md).
