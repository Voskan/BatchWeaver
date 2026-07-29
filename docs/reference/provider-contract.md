# Provider Contract

A provider executes a batch of requests against a backend:

```go
type Provider[K, V any] interface {
    Execute(context.Context, batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error)
}
```

## Request and response

- The runtime never calls a provider with an empty request.
- The provider must return exactly one outcome per request ID it received, using
  the request IDs from the `BatchRequest`. Order does not matter; outcomes are
  matched by request ID.
- A non-nil returned error is a **global** failure applied to all unresolved
  items. Per-item failures belong in the outcomes (`Outcome.Err`).

## Validation

Before any value reaches a caller, the runtime validates the response against the
exact set of request IDs:

- a missing required result is a batch contract violation (`ErrBatchContractViolation`);
- a duplicate result is a contract violation;
- an unexpected (unrequested) request ID is a contract violation.

A per-item `Outcome` of not-found (found = false, no error) is valid and maps to
`ErrMissingResult` for the scalar caller.

## Context

The provider receives a dedicated batch context, not any individual caller's
context. It carries engine/scope lifecycle cancellation and, when applicable, a
batch deadline. It does not inherit arbitrary caller context values. A provider
that needs to call another bound operation must create its own scope.

## Concurrency

A provider may be called concurrently for different batches, up to the
per-operation concurrency limit. Provider calls never run while the coordinator
holds no lock (the coordinator is lock-free) and never block the coordinator.

## Panics

By default the runtime recovers a provider panic, converts it into a
`ProviderPanicError` (matched by `errors.Is(err, ErrProviderPanic)`), fails the
batch's items, and keeps the engine healthy. A sanitized value is included; no
stack trace is exposed. The engine can be configured to re-panic instead.

## Recursion

If a provider calls the same operation and partition it is currently executing,
the runtime detects it (via an execution marker in the batch context) and returns
`ErrRecursiveOperation` (or runs the scalar fallback when configured) rather than
deadlocking.

## Ownership

The runtime hands the provider an owned, stable request slice; later queue
mutation does not affect it. Result slices are not retained longer than needed.
