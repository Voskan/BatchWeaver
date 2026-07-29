package batchweaver

import "fmt"

// ItemResult is a per-item provider result used when adapting ordered results
// that may individually succeed, fail, or be missing.
type ItemResult[V any] struct {
	// Value is the successful value; meaningful only when Found is true.
	Value V
	// Err is a per-item error, if any.
	Err error
	// Found reports whether a value was produced.
	Found bool
}

// OrderedOutcomes maps a slice of values to outcomes by request order. It
// requires len(values) == req.Len() and treats every value as a success.
// Distinct request IDs are preserved even when item keys are equal.
func OrderedOutcomes[K, V any](req BatchRequest[K], values []V) (BatchResponse[V], error) {
	items := req.Items()
	if len(values) != len(items) {
		return BatchResponse[V]{}, fmt.Errorf("%w: got %d values for %d items", ErrInvalidBatchResponse, len(values), len(items))
	}
	outcomes := make([]Outcome[V], len(items))
	for i := range items {
		outcomes[i] = Success(items[i].ID, values[i])
	}
	return NewBatchResponse(outcomes)
}

// OrderedResultOutcomes maps ordered per-item results to outcomes by request
// order. It requires len(results) == req.Len().
func OrderedResultOutcomes[K, V any](req BatchRequest[K], results []ItemResult[V]) (BatchResponse[V], error) {
	items := req.Items()
	if len(results) != len(items) {
		return BatchResponse[V]{}, fmt.Errorf("%w: got %d results for %d items", ErrInvalidBatchResponse, len(results), len(items))
	}
	outcomes := make([]Outcome[V], len(items))
	for i := range items {
		outcomes[i] = itemResultToOutcome(items[i].ID, results[i])
	}
	return NewBatchResponse(outcomes)
}

// KeyedOutcomes maps a map of values keyed by K to outcomes, one per request
// item. When a key is absent, onMissing is consulted; if onMissing is nil, a
// not-found outcome is produced. Duplicate keys across items yield separate
// outcomes with their own request IDs.
func KeyedOutcomes[K comparable, V any](
	req BatchRequest[K],
	values map[K]V,
	onMissing func(id RequestID, key K) Outcome[V],
) (BatchResponse[V], error) {
	items := req.Items()
	outcomes := make([]Outcome[V], len(items))
	for i := range items {
		if v, ok := values[items[i].Key]; ok {
			outcomes[i] = Success(items[i].ID, v)
			continue
		}
		outcomes[i] = missingOutcome(items[i].ID, items[i].Key, onMissing)
	}
	return NewBatchResponse(outcomes)
}

// SparseOutcomes maps outcomes using a callback lookup, which avoids requiring
// comparable keys. When lookup reports a key absent, onMissing is consulted; if
// onMissing is nil, a not-found outcome is produced.
func SparseOutcomes[K, V any](
	req BatchRequest[K],
	lookup func(key K) (V, bool),
	onMissing func(id RequestID, key K) Outcome[V],
) (BatchResponse[V], error) {
	if lookup == nil {
		return BatchResponse[V]{}, fmt.Errorf("%w: lookup is nil", ErrInvalidBatchResponse)
	}
	items := req.Items()
	outcomes := make([]Outcome[V], len(items))
	for i := range items {
		if v, ok := lookup(items[i].Key); ok {
			outcomes[i] = Success(items[i].ID, v)
			continue
		}
		outcomes[i] = missingOutcome(items[i].ID, items[i].Key, onMissing)
	}
	return NewBatchResponse(outcomes)
}

// itemResultToOutcome converts an ItemResult into a canonical Outcome.
func itemResultToOutcome[V any](id RequestID, r ItemResult[V]) Outcome[V] {
	switch {
	case r.Err != nil:
		return Failure[V](id, r.Err)
	case r.Found:
		return Success(id, r.Value)
	default:
		return NotFound[V](id)
	}
}

// missingOutcome applies onMissing or returns a not-found outcome.
func missingOutcome[K, V any](id RequestID, key K, onMissing func(RequestID, K) Outcome[V]) Outcome[V] {
	if onMissing != nil {
		return onMissing(id, key)
	}
	return NotFound[V](id)
}
