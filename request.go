package batchweaver

import (
	"errors"
	"fmt"
)

// ErrInvalidBatchRequest is returned when a BatchRequest fails validation.
var ErrInvalidBatchRequest = errors.New("invalid batch request")

// BatchRequest is an immutable-by-convention, ordered collection of batch items
// with unique request IDs. Construct it with NewBatchRequest, which defensively
// copies the input; accessors also return copies so callers cannot mutate the
// stored items.
type BatchRequest[K any] struct {
	items []BatchItem[K]
}

// NewBatchRequest validates items and returns a BatchRequest. It rejects an
// empty request, invalid items, and duplicate request IDs, and copies the input
// slice so later mutation of the caller's slice does not affect the request.
func NewBatchRequest[K any](items []BatchItem[K]) (BatchRequest[K], error) {
	if len(items) == 0 {
		return BatchRequest[K]{}, fmt.Errorf("%w: request has no items", ErrInvalidBatchRequest)
	}
	seen := make(map[RequestID]struct{}, len(items))
	cp := make([]BatchItem[K], len(items))
	for i, it := range items {
		if err := it.Validate(); err != nil {
			return BatchRequest[K]{}, fmt.Errorf("%w: item %d: %w", ErrInvalidBatchRequest, i, err)
		}
		if _, dup := seen[it.ID]; dup {
			return BatchRequest[K]{}, fmt.Errorf("%w: duplicate request id %d", ErrInvalidBatchRequest, it.ID)
		}
		seen[it.ID] = struct{}{}
		cp[i] = it
	}
	return BatchRequest[K]{items: cp}, nil
}

// MustNewBatchRequest is like NewBatchRequest but panics on error. It is
// intended for tests and internal construction with known-valid input.
func MustNewBatchRequest[K any](items []BatchItem[K]) BatchRequest[K] {
	r, err := NewBatchRequest(items)
	if err != nil {
		panic(fmt.Sprintf("batchweaver.MustNewBatchRequest: %v", err))
	}
	return r
}

// Len returns the number of items.
func (r BatchRequest[K]) Len() int { return len(r.items) }

// Items returns a copy of the items in their original order.
func (r BatchRequest[K]) Items() []BatchItem[K] {
	out := make([]BatchItem[K], len(r.items))
	copy(out, r.items)
	return out
}

// IDs returns the request IDs in order.
func (r BatchRequest[K]) IDs() []RequestID {
	out := make([]RequestID, len(r.items))
	for i := range r.items {
		out[i] = r.items[i].ID
	}
	return out
}

// Validate reports whether the request is well-formed. A request built by
// NewBatchRequest is always valid; this re-checks a value that may have been
// constructed as a zero value.
func (r BatchRequest[K]) Validate() error {
	if len(r.items) == 0 {
		return fmt.Errorf("%w: request has no items", ErrInvalidBatchRequest)
	}
	seen := make(map[RequestID]struct{}, len(r.items))
	for i, it := range r.items {
		if err := it.Validate(); err != nil {
			return fmt.Errorf("%w: item %d: %w", ErrInvalidBatchRequest, i, err)
		}
		if _, dup := seen[it.ID]; dup {
			return fmt.Errorf("%w: duplicate request id %d", ErrInvalidBatchRequest, it.ID)
		}
		seen[it.ID] = struct{}{}
	}
	return nil
}
