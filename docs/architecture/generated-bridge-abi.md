# Generated bridge ABI

The public `bridge` package is the stable, typed application binary interface
between BatchWeaver-generated code and the Prompt 03 runtime. Its version is
`batchweaver.bridge/v1alpha1`.

## Shape

```go
type Operation[R, K, V any] struct {
    OpID   string
    Scalar func(ctx context.Context, receiver R, key K) (V, error)
}

func (o Operation[R, K, V]) Call(ctx context.Context, receiver R, key K) (V, error)
func WithOperation[K, V any](ctx context.Context, opID string, op *runtime.BoundOperation[K, V]) context.Context
func Flush(ctx context.Context) error
func Barrier(ctx context.Context) error
```

## Semantics

- `Call` resolves the typed bound operation installed for `OpID` from context. If
  present, it routes the request through the runtime (`BoundOperation.Do`), which
  coalesces compatible same-scope or concurrent calls. If absent, it calls
  `Scalar` directly, preserving the original behavior and error identity.
- There is no reflection on the call path. The only context lookup is a single
  O(1) typed fetch of the bound operation.
- `Flush`/`Barrier` flush the active scope's pending operations; they are no-ops
  when no scope is active.

## Application responsibility

The application declares the batch provider and installs typed bound operations
into the scope context with `WithOperation` for the receiver in scope. A lowered
call site with no installed bound operation behaves exactly like the original
scalar call. The bound operation must correspond to the receiver used at the
lowered call sites; the caller owns that correspondence
(see [ADR 0035](../adr/0035-context-scoped-engine.md)).

## Versioning

The ABI version is recorded on every runtime transformation and in generated
files. A change invalidates generated bridges and plans through the plan digest
(see [ADR 0036](../adr/0036-runtime-abi-versioning.md)).
