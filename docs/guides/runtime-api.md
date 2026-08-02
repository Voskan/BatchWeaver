# Runtime API Guide

The runtime package name shadows the standard library, so import it with an
alias:

```go
import batchruntime "github.com/Voskan/BatchWeaver/runtime"
```

## Create an engine

```go
engine, err := batchruntime.NewEngine(
    batchruntime.WithDefaultQueueLimits(batchruntime.QueueLimits{
        MaxItems: 100_000, MaxBytes: 64 << 20, MaxPartitions: 10_000,
    }),
)
if err != nil {
    return err
}
defer engine.Close(context.Background())
```

## Bind a typed operation

Bindings use a struct for clean generic inference:

```go
getUser, err := batchruntime.Bind(engine, users.GetUserOperation, batchruntime.Binding[users.UserID, users.User]{
    Keys:     batchruntime.ComparableKeys[users.UserID](),
    Provider: batchruntime.ProviderFunc[users.UserID, users.User](repo.GetUsersBatch),
    Partitioner: batchruntime.PartitionerFunc[users.UserID](func(ctx context.Context, id users.UserID) (batchruntime.Partition, error) {
        return batchruntime.PartitionFromStrings(tenant.FromContext(ctx)), nil
    }),
})
```

The declaration (`users.GetUserOperation`) supplies the operation ID, semantics,
and operation policies; the `Binding` supplies the runtime provider and
strategies. This differs from a variadic-option sketch because a config struct
keeps type inference clean while keys are unconstrained.

## Run a scope and coalesce calls

```go
users, err := batchruntime.Run(engine, ctx, func(ctx context.Context) ([]User, error) {
    out := make([]User, len(ids))
    g, ctx := errgroup.WithContext(ctx)
    for i, id := range ids {
        i, id := i, id
        g.Go(func() error {
            u, err := getUser.Do(ctx, id)
            if err != nil { return err }
            out[i] = u
            return nil
        })
    }
    return out, g.Wait()
})
```

Compatible `Do` calls within the scope are coalesced into bounded provider
batches according to the operation's scheduling policy.

## Flush and drain

```go
scope.Flush(ctx) // dispatch barrier
scope.Drain(ctx) // completion barrier
```

## Limitations

The runtime package is explicit: importing or binding it does not scan or mutate
source. The separate compiler can lower proven scalar call sites through the
typed bridge, and the adaptive layer can apply settings within binding hard
bounds when explicitly enabled. Providers and telemetry exporters remain
application-wired; there is no hidden global registration or remote telemetry.
