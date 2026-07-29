package batchweaver

import (
	"errors"
	"testing"
)

func TestOutcomeStatesAndValidation(t *testing.T) {
	t.Parallel()
	if s := Success(1, "v"); !s.IsSuccess() || s.IsNotFound() || s.IsFailure() {
		t.Errorf("success state wrong: %+v", s)
	}
	if n := NotFound[string](1); !n.IsNotFound() || n.IsSuccess() || n.IsFailure() {
		t.Errorf("not-found state wrong: %+v", n)
	}
	if f := Failure[string](1, errors.New("boom")); !f.IsFailure() || f.IsSuccess() {
		t.Errorf("failure state wrong: %+v", f)
	}
	for _, o := range []Outcome[string]{Success(1, "v"), NotFound[string](1), Failure[string](1, errors.New("e"))} {
		if err := o.Validate(); err != nil {
			t.Errorf("valid outcome rejected: %v", err)
		}
	}
	// Ambiguous found+error.
	bad := Outcome[string]{RequestID: 1, Found: true, Err: errors.New("x")}
	if err := bad.Validate(); !errors.Is(err, ErrInvalidOutcome) {
		t.Errorf("ambiguous outcome accepted: %v", err)
	}
	// Zero request id.
	if err := (Outcome[string]{Found: true}).Validate(); !errors.Is(err, ErrInvalidOutcome) {
		t.Errorf("zero-id outcome accepted")
	}
}

func TestFailurePreservesWrappedError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("root cause")
	f := Failure[string](1, sentinel)
	if !errors.Is(f.Err, sentinel) {
		t.Errorf("wrapped error identity lost")
	}
}

func TestBatchResponseValidateAgainst(t *testing.T) {
	t.Parallel()
	resp := MustNewBatchResponse([]Outcome[string]{Success(1, "a"), NotFound[string](2)})
	if err := resp.ValidateAgainst([]RequestID{1, 2}); err != nil {
		t.Errorf("valid response rejected: %v", err)
	}
	if err := resp.ValidateAgainst([]RequestID{1, 2, 3}); err == nil {
		t.Errorf("missing id not detected")
	}
	if err := resp.ValidateAgainst([]RequestID{1}); err == nil {
		t.Errorf("unexpected id not detected")
	}
}

func TestBatchResponseDuplicateDetected(t *testing.T) {
	t.Parallel()
	_, err := NewBatchResponse([]Outcome[string]{Success(1, "a"), Success(1, "b")})
	if err != nil {
		// NewBatchResponse does not reject duplicates itself; Validate does.
		t.Fatalf("unexpected error: %v", err)
	}
	resp := MustNewBatchResponse([]Outcome[string]{Success(1, "a"), Success(1, "b")})
	if err := resp.Validate(); !errors.Is(err, ErrInvalidBatchResponse) {
		t.Errorf("duplicate request id not detected: %v", err)
	}
}

func TestBatchResponseDefensiveCopy(t *testing.T) {
	t.Parallel()
	resp := MustNewBatchResponse([]Outcome[int]{Success(1, 10)})
	got := resp.Outcomes()
	got[0].Value = 999
	if resp.Outcomes()[0].Value != 10 {
		t.Errorf("Outcomes() exposed internal storage")
	}
}
