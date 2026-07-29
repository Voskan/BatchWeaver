# Typed Declarations Reference

Typed declarations connect an operation spec to concrete scalar and batch
implementations, with full type safety and no global registration. Declaring an
operation does not intercept or batch calls yet; it defines the contract that
later compiler and runtime prompts will use.

## Function types

```go
type ScalarFunc[K, V any] func(context.Context, K) (V, error)
type BatchFunc[K, V any]  func(context.Context, BatchRequest[K]) (BatchResponse[V], error)

type ScalarMethod[R, K, V any] func(R, context.Context, K) (V, error)
type BatchMethod[R, K, V any]  func(R, context.Context, BatchRequest[K]) (BatchResponse[V], error)
```

Keys need not be `comparable`.

## Constructors

```go
func DeclareFunction[K, V any](spec operation.Spec, scalar ScalarFunc[K, V], batch BatchFunc[K, V]) (FunctionDeclaration[K, V], error)
func MustDeclareFunction[K, V any](...) FunctionDeclaration[K, V]

func DeclareMethod[R, K, V any](spec operation.Spec, scalar ScalarMethod[R, K, V], batch BatchMethod[R, K, V]) (MethodDeclaration[R, K, V], error)
func MustDeclareMethod[R, K, V any](...) MethodDeclaration[R, K, V]
```

The `Must` variants panic only for programmer errors during package
initialization, with deterministic messages that include the operation ID.

## Canonical shape

```go
var GetUserOperation = batchweaver.MustDeclareMethod(
    operation.MustNewSpec(
        operation.MustParseID("users.get"),
        operation.ReadOnly(),
        operation.WithOrderedResults(),
        operation.WithRequestScope(),
    ),
    (*Repository).GetUser,
    (*Repository).GetUsersBatch,
)
```

A package-level `var` assigned from `MustDeclareMethod`/`MustDeclareFunction` is
the shape a future analyzer discovers statically. Do not wrap declarations in
aliases or helpers that obscure this shape.

## Building responses

Helpers convert common provider result shapes into a `BatchResponse`:

```go
func OrderedOutcomes[K, V any](req BatchRequest[K], values []V) (BatchResponse[V], error)
func OrderedResultOutcomes[K, V any](req BatchRequest[K], results []ItemResult[V]) (BatchResponse[V], error)
func KeyedOutcomes[K comparable, V any](req BatchRequest[K], values map[K]V, onMissing func(RequestID, K) Outcome[V]) (BatchResponse[V], error)
func SparseOutcomes[K, V any](req BatchRequest[K], lookup func(K) (V, bool), onMissing func(RequestID, K) Outcome[V]) (BatchResponse[V], error)
```

A complete, compile-tested example is in
[examples/declarations/basic](../../examples/declarations/basic).
