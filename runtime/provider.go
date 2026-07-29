package runtime

import (
	"context"

	batchweaver "github.com/Voskan/BatchWeaver"
)

// Provider executes a batch of requests against a backend. The runtime never
// calls a provider with an empty request. The returned error is a global
// provider or transport failure; per-item failures belong in the response
// outcomes. Providers may be called concurrently for different batches, subject
// to the per-operation concurrency limit.
type Provider[K any, V any] interface {
	// Execute runs the batch request and returns per-item outcomes. A non-nil
	// error is a global failure applied to all unresolved items.
	Execute(context.Context, batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error)
}

// ProviderFunc adapts a function to Provider.
type ProviderFunc[K any, V any] func(context.Context, batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error)

// Execute calls the function.
func (f ProviderFunc[K, V]) Execute(ctx context.Context, req batchweaver.BatchRequest[K]) (batchweaver.BatchResponse[V], error) {
	return f(ctx, req)
}

// ScalarFallback executes a single request directly, bypassing batching. It is
// used by the direct-fallback overflow policy and scope-less execution when the
// binding permits it. It receives the original caller context.
type ScalarFallback[K any, V any] interface {
	// Execute runs a single request and returns its result.
	Execute(context.Context, K) (V, error)
}

// ScalarFallbackFunc adapts a function to ScalarFallback.
type ScalarFallbackFunc[K any, V any] func(context.Context, K) (V, error)

// Execute calls the function.
func (f ScalarFallbackFunc[K, V]) Execute(ctx context.Context, key K) (V, error) {
	return f(ctx, key)
}
