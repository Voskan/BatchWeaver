package batchweaver

import (
	"errors"
	"testing"
)

func TestNewBatchRequest(t *testing.T) {
	t.Parallel()
	req, err := NewBatchRequest([]BatchItem[int]{
		NewBatchItem(1, 10),
		NewBatchItem(2, 20),
	})
	if err != nil {
		t.Fatalf("NewBatchRequest: %v", err)
	}
	if req.Len() != 2 {
		t.Errorf("Len = %d", req.Len())
	}
	if ids := req.IDs(); ids[0] != 1 || ids[1] != 2 {
		t.Errorf("IDs = %v", ids)
	}
}

func TestNewBatchRequestRejectsEmpty(t *testing.T) {
	t.Parallel()
	if _, err := NewBatchRequest[int](nil); !errors.Is(err, ErrInvalidBatchRequest) {
		t.Errorf("empty request accepted: %v", err)
	}
}

func TestNewBatchRequestRejectsDuplicateID(t *testing.T) {
	t.Parallel()
	_, err := NewBatchRequest([]BatchItem[int]{NewBatchItem(1, 10), NewBatchItem(1, 20)})
	if !errors.Is(err, ErrInvalidBatchRequest) {
		t.Errorf("duplicate id accepted: %v", err)
	}
}

func TestNewBatchRequestRejectsZeroID(t *testing.T) {
	t.Parallel()
	_, err := NewBatchRequest([]BatchItem[int]{{ID: 0, Key: 1, Weight: 1}})
	if !errors.Is(err, ErrInvalidBatchRequest) {
		t.Errorf("zero id accepted: %v", err)
	}
}

func TestBatchRequestDefensiveCopy(t *testing.T) {
	t.Parallel()
	items := []BatchItem[int]{NewBatchItem(1, 10)}
	req := MustNewBatchRequest(items)
	items[0].Key = 999 // mutate caller slice
	if req.Items()[0].Key != 10 {
		t.Errorf("request aliases caller slice")
	}
	// Mutating the returned slice must not affect the request.
	got := req.Items()
	got[0].Key = 777
	if req.Items()[0].Key != 10 {
		t.Errorf("Items() exposed internal storage")
	}
}

func TestBatchItemValidation(t *testing.T) {
	t.Parallel()
	if err := NewBatchItem(1, "k").Validate(); err != nil {
		t.Errorf("valid item rejected: %v", err)
	}
	if err := (BatchItem[string]{ID: 1, Weight: 0}).Validate(); err == nil {
		t.Errorf("zero weight accepted")
	}
	if err := NewBatchItem(1, "k").WithPriority(MaxPriority + 1).Validate(); err == nil {
		t.Errorf("out-of-range priority accepted")
	}
}
