// Package batchweaver defines the public, typed contracts used to declare and
// implement semantically safe Go batch operations.
//
// The package provides generic request and response types, per-item outcomes,
// scalar and batch function signatures, and declarations that connect an
// [operation.Spec] to concrete implementations. Declarations have no global
// registration side effects and can be discovered statically by the
// BatchWeaver analyzer.
//
// A batch provider must return exactly one outcome for every request ID unless
// it returns a global error. Helpers such as [OrderedOutcomes], [KeyedOutcomes],
// and [SparseOutcomes] preserve request identity and validate result shape.
// See the package examples for a complete typed declaration.
//
// Request coalescing is implemented by the runtime package, normally imported
// with an alias:
//
//	import batchruntime "github.com/Voskan/BatchWeaver/runtime"
//
// Static analysis, proof-gated transformation, overlays, and materialization
// are exposed through the batchweaver command. Source is never changed merely
// by importing this package or declaring an operation.
package batchweaver
