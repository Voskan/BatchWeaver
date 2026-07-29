package batchweaver

import (
	"errors"
	"testing"
)

func newIntReq(t *testing.T, ids ...RequestID) BatchRequest[int] {
	t.Helper()
	items := make([]BatchItem[int], len(ids))
	for i, id := range ids {
		items[i] = NewBatchItem(id, int(id)*10)
	}
	return MustNewBatchRequest(items)
}

func TestOrderedOutcomes(t *testing.T) {
	t.Parallel()
	req := newIntReq(t, 1, 2, 3)
	resp, err := OrderedOutcomes(req, []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("OrderedOutcomes: %v", err)
	}
	if resp.Len() != 3 {
		t.Errorf("Len = %d", resp.Len())
	}
	if _, err := OrderedOutcomes(req, []string{"a"}); err == nil {
		t.Errorf("length mismatch not detected")
	}
}

func TestOrderedResultOutcomes(t *testing.T) {
	t.Parallel()
	req := newIntReq(t, 1, 2, 3)
	results := []ItemResult[string]{
		{Value: "a", Found: true},
		{Found: false},
		{Err: errors.New("boom")},
	}
	resp, err := OrderedResultOutcomes(req, results)
	if err != nil {
		t.Fatalf("OrderedResultOutcomes: %v", err)
	}
	out := resp.Outcomes()
	if !out[0].IsSuccess() || !out[1].IsNotFound() || !out[2].IsFailure() {
		t.Errorf("outcome states wrong: %+v", out)
	}
}

func TestKeyedOutcomes(t *testing.T) {
	t.Parallel()
	// Two items share the same key; both must get outcomes with their own IDs.
	req := MustNewBatchRequest([]BatchItem[string]{
		NewBatchItem(1, "x"),
		NewBatchItem(2, "y"),
		NewBatchItem(3, "x"),
	})
	values := map[string]int{"x": 100}
	resp, err := KeyedOutcomes(req, values, nil)
	if err != nil {
		t.Fatalf("KeyedOutcomes: %v", err)
	}
	out := resp.Outcomes()
	if !out[0].IsSuccess() || out[0].Value != 100 {
		t.Errorf("item 1 outcome: %+v", out[0])
	}
	if !out[1].IsNotFound() {
		t.Errorf("item 2 should be not-found: %+v", out[1])
	}
	if !out[2].IsSuccess() || out[2].RequestID != 3 {
		t.Errorf("duplicate-key item 3 outcome wrong: %+v", out[2])
	}
}

func TestKeyedOutcomesOnMissing(t *testing.T) {
	t.Parallel()
	req := newIntReq(t, 1)
	resp, err := KeyedOutcomes(req, map[int]string{}, func(id RequestID, key int) Outcome[string] {
		return Failure[string](id, errors.New("absent"))
	})
	if err != nil {
		t.Fatalf("KeyedOutcomes: %v", err)
	}
	if !resp.Outcomes()[0].IsFailure() {
		t.Errorf("onMissing not applied")
	}
}

func TestSparseOutcomesNilLookup(t *testing.T) {
	t.Parallel()
	req := newIntReq(t, 1)
	if _, err := SparseOutcomes[int, string](req, nil, nil); !errors.Is(err, ErrInvalidBatchResponse) {
		t.Errorf("nil lookup accepted: %v", err)
	}
}
