package batchweaver

import (
	"errors"
	"fmt"
)

// ErrInvalidOutcome is returned when an Outcome fails validation.
var ErrInvalidOutcome = errors.New("invalid outcome")

// ErrInvalidBatchResponse is returned when a BatchResponse fails validation.
var ErrInvalidBatchResponse = errors.New("invalid batch response")

// Outcome is the result of a single logical request within a batch. Exactly one
// of three states is valid: success (Found, no error), not-found (no error, not
// found), or failure (error, not found). The ambiguous found-plus-error state is
// rejected by validation.
type Outcome[V any] struct {
	// RequestID identifies which request this outcome answers; non-zero.
	RequestID RequestID
	// Value is the successful result; meaningful only when Found is true.
	Value V
	// Err is the per-item error; non-nil only for a failure outcome.
	Err error
	// Found reports whether a value was produced.
	Found bool
}

// Success returns a successful outcome for id carrying value.
func Success[V any](id RequestID, value V) Outcome[V] {
	return Outcome[V]{RequestID: id, Value: value, Found: true}
}

// NotFound returns a not-found outcome for id.
func NotFound[V any](id RequestID) Outcome[V] {
	return Outcome[V]{RequestID: id}
}

// Failure returns a failure outcome for id carrying err. A nil err is treated as
// an invalid outcome by Validate.
func Failure[V any](id RequestID, err error) Outcome[V] {
	return Outcome[V]{RequestID: id, Err: err}
}

// Validate reports whether the outcome is in a well-formed state.
func (o Outcome[V]) Validate() error {
	if !o.RequestID.IsValid() {
		return fmt.Errorf("%w: request id must be non-zero", ErrInvalidOutcome)
	}
	switch {
	case o.Err != nil && o.Found:
		return fmt.Errorf("%w: outcome cannot be both found and failed", ErrInvalidOutcome)
	case o.Err == nil && o.Found:
		return nil // success
	case o.Err == nil && !o.Found:
		return nil // not found
	default:
		return nil // failure (Err != nil, !Found)
	}
}

// IsSuccess reports whether the outcome is a successful value.
func (o Outcome[V]) IsSuccess() bool { return o.Found && o.Err == nil }

// IsNotFound reports whether the outcome represents a missing value.
func (o Outcome[V]) IsNotFound() bool { return !o.Found && o.Err == nil }

// IsFailure reports whether the outcome carries an error.
func (o Outcome[V]) IsFailure() bool { return o.Err != nil }

// BatchResponse is an immutable-by-convention, ordered collection of outcomes.
// Construct it with NewBatchResponse, which defensively copies the input;
// accessors return copies. Order is preserved but the type does not force any
// particular mapping policy.
type BatchResponse[V any] struct {
	outcomes []Outcome[V]
}

// NewBatchResponse validates outcomes and returns a BatchResponse, copying the
// input slice. It rejects invalid outcomes but permits an empty response so that
// callers can represent a global failure with no outcomes.
func NewBatchResponse[V any](outcomes []Outcome[V]) (BatchResponse[V], error) {
	cp := make([]Outcome[V], len(outcomes))
	for i, o := range outcomes {
		if err := o.Validate(); err != nil {
			return BatchResponse[V]{}, fmt.Errorf("%w: outcome %d: %w", ErrInvalidBatchResponse, i, err)
		}
		cp[i] = o
	}
	return BatchResponse[V]{outcomes: cp}, nil
}

// MustNewBatchResponse is like NewBatchResponse but panics on error.
func MustNewBatchResponse[V any](outcomes []Outcome[V]) BatchResponse[V] {
	r, err := NewBatchResponse(outcomes)
	if err != nil {
		panic(fmt.Sprintf("batchweaver.MustNewBatchResponse: %v", err))
	}
	return r
}

// Len returns the number of outcomes.
func (r BatchResponse[V]) Len() int { return len(r.outcomes) }

// Outcomes returns a copy of the outcomes in order.
func (r BatchResponse[V]) Outcomes() []Outcome[V] {
	out := make([]Outcome[V], len(r.outcomes))
	copy(out, r.outcomes)
	return out
}

// Validate reports whether all outcomes are well-formed and no request ID is
// duplicated.
func (r BatchResponse[V]) Validate() error {
	seen := make(map[RequestID]struct{}, len(r.outcomes))
	for i, o := range r.outcomes {
		if err := o.Validate(); err != nil {
			return fmt.Errorf("%w: outcome %d: %w", ErrInvalidBatchResponse, i, err)
		}
		if _, dup := seen[o.RequestID]; dup {
			return fmt.Errorf("%w: duplicate request id %d", ErrInvalidBatchResponse, o.RequestID)
		}
		seen[o.RequestID] = struct{}{}
	}
	return nil
}

// ValidateAgainst checks the response against the exact set of request IDs it
// should answer. It reports missing IDs, unexpected IDs, duplicate IDs, and
// invalid outcomes. The requestIDs argument is treated as the authoritative set.
func (r BatchResponse[V]) ValidateAgainst(requestIDs []RequestID) error {
	if err := r.Validate(); err != nil {
		return err
	}
	want := make(map[RequestID]struct{}, len(requestIDs))
	for _, id := range requestIDs {
		want[id] = struct{}{}
	}
	got := make(map[RequestID]struct{}, len(r.outcomes))
	for _, o := range r.outcomes {
		if _, ok := want[o.RequestID]; !ok {
			return fmt.Errorf("%w: unexpected request id %d", ErrInvalidBatchResponse, o.RequestID)
		}
		got[o.RequestID] = struct{}{}
	}
	for id := range want {
		if _, ok := got[id]; !ok {
			return fmt.Errorf("%w: missing request id %d", ErrInvalidBatchResponse, id)
		}
	}
	return nil
}
