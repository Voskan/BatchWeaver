// Package bridge is the stable, typed application binary interface (ABI) between
// BatchWeaver-generated code and the typed runtime. Generated bridge files
// (zz_batchweaver_*_gen.go) declare an Operation for each lowered operation and
// call its Call method in place of the original scalar call.
//
// The bridge contains no reflection on its call path. When no runtime bound
// operation is installed in the context, Call invokes the original scalar
// function directly, so a lowered call site behaves exactly like the original.
// When the application installs a typed bound operation for the scope (via
// WithOperation), Call routes the request through the runtime, which coalesces
// compatible concurrent or same-scope calls.
package bridge

import (
	"context"

	batchruntime "github.com/Voskan/BatchWeaver/runtime"
)

// ABIVersion identifies the generated-bridge ABI. Generated bridges record it so
// a mismatch between generated code and this package can be detected and the
// bridge regenerated. It is independent from the transformation, proof, and
// analysis schema versions.
const ABIVersion = "batchweaver.bridge/v1alpha1"

// Operation is the immutable binding metadata for one lowered operation. R is
// the receiver type, K the key type, and V the value type of the scalar
// operation. Construct it in generated code as a package-level value; it holds
// no mutable state and is safe for concurrent use.
type Operation[R, K, V any] struct {
	// OpID is the canonical operation identifier (for example "users.get").
	OpID string
	// Scalar is the original scalar function, used for direct execution when no
	// runtime bound operation is installed for this operation in the context.
	Scalar func(ctx context.Context, receiver R, key K) (V, error)
}

// boundKey is the context key under which a typed bound operation is installed.
// It is generic so lookup returns the exact typed handle without reflection.
type boundKey[K, V any] struct{ opID string }

// WithOperation installs a typed runtime bound operation for opID into ctx.
// Calls to the corresponding Operation.Call made with the returned context (and
// its descendants) route through the runtime; other calls fall back to the
// scalar function. The bound operation must have been created for the receiver
// used at the lowered call sites in scope; the caller owns that correspondence.
func WithOperation[K, V any](ctx context.Context, opID string, op *batchruntime.BoundOperation[K, V]) context.Context {
	return context.WithValue(ctx, boundKey[K, V]{opID: opID}, op)
}

// Call executes the operation. It routes through the installed runtime bound
// operation when one is present for this operation and key/value type, and
// otherwise invokes the scalar function directly. It never converts a panic to
// an error and preserves the scalar error identity on the fallback path.
func (o Operation[R, K, V]) Call(ctx context.Context, receiver R, key K) (V, error) {
	if bound, ok := ctx.Value(boundKey[K, V]{opID: o.OpID}).(*batchruntime.BoundOperation[K, V]); ok && bound != nil {
		return bound.Do(ctx, key)
	}
	return o.Scalar(ctx, receiver, key)
}

// Flush flushes all pending batched operations in the active scope, blocking
// until they complete. It is a no-op when no scope is active. Use it as an
// explicit batching barrier before an observable side effect that must not be
// reordered across pending reads.
func Flush(ctx context.Context) error {
	if s, ok := batchruntime.ScopeFromContext(ctx); ok {
		return s.Flush(ctx)
	}
	return nil
}

// Barrier is an alias for Flush expressing intent at a batching boundary.
func Barrier(ctx context.Context) error { return Flush(ctx) }
