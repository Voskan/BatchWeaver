package batchweaver

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// RequestID identifies a single logical request within one BatchRequest.
//
// The zero value is invalid for a dispatched request. IDs are opaque to
// providers, which must return the same IDs they received; they are scoped to a
// single BatchRequest and need not be globally unique. No IDs are generated in
// this release.
type RequestID uint64

// IsValid reports whether the request ID is non-zero.
func (id RequestID) IsValid() bool { return id != 0 }

// Priority bounds. Priority is relative: higher values indicate higher priority.
const (
	// MinPriority is the lowest allowed item priority.
	MinPriority = -1000
	// MaxPriority is the highest allowed item priority.
	MaxPriority = 1000
)

// ErrInvalidBatchItem is returned when a BatchItem fails validation.
var ErrInvalidBatchItem = errors.New("invalid batch item")

// BatchItem is one logical request within a batch. Key is the operation's input
// value; the remaining fields are batching metadata. A zero Deadline means the
// item has no item-specific deadline. K is unconstrained: keys need not be
// comparable.
type BatchItem[K any] struct {
	// ID identifies the request; it must be non-zero.
	ID RequestID
	// Key is the operation input for this request.
	Key K
	// Deadline is an optional per-item deadline; the zero time means none.
	Deadline time.Time
	// Priority is the relative priority within [MinPriority, MaxPriority].
	Priority int
	// Weight is the item's cost contribution; it must be positive.
	Weight int
}

// NewBatchItem returns a BatchItem with the given ID and key, a weight of one,
// and no deadline or priority. Use the With* methods to set optional fields.
func NewBatchItem[K any](id RequestID, key K) BatchItem[K] {
	return BatchItem[K]{ID: id, Key: key, Weight: 1}
}

// WithDeadline returns a copy of the item with the given deadline.
func (it BatchItem[K]) WithDeadline(d time.Time) BatchItem[K] {
	it.Deadline = d
	return it
}

// WithPriority returns a copy of the item with the given priority.
func (it BatchItem[K]) WithPriority(p int) BatchItem[K] {
	it.Priority = p
	return it
}

// WithWeight returns a copy of the item with the given weight.
func (it BatchItem[K]) WithWeight(w int) BatchItem[K] {
	it.Weight = w
	return it
}

// Validate reports whether the item is well-formed.
func (it BatchItem[K]) Validate() error {
	if !it.ID.IsValid() {
		return fmt.Errorf("%w: request id must be non-zero", ErrInvalidBatchItem)
	}
	if it.Weight < 1 {
		return fmt.Errorf("%w: weight must be positive", ErrInvalidBatchItem)
	}
	if it.Priority < MinPriority || it.Priority > MaxPriority {
		return fmt.Errorf("%w: priority %d out of range [%d, %d]", ErrInvalidBatchItem, it.Priority, MinPriority, MaxPriority)
	}
	return nil
}

// ScalarFunc is the signature of a scalar operation: one input, one result.
type ScalarFunc[K, V any] func(context.Context, K) (V, error)

// BatchFunc is the signature of a batch operation: many inputs, many outcomes.
// The returned error represents a global provider or transport failure;
// per-item failures belong in Outcome.Err.
type BatchFunc[K, V any] func(context.Context, BatchRequest[K]) (BatchResponse[V], error)

// ScalarMethod is a scalar operation expressed as a method expression, so the
// receiver R is an explicit first parameter. R may be a pointer or interface
// type.
type ScalarMethod[R, K, V any] func(R, context.Context, K) (V, error)

// BatchMethod is a batch operation expressed as a method expression.
type BatchMethod[R, K, V any] func(R, context.Context, BatchRequest[K]) (BatchResponse[V], error)
