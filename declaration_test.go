package batchweaver

import (
	"context"
	"errors"
	"testing"

	"github.com/Voskan/BatchWeaver/operation"
)

// The following types model the canonical declaration example and double as an
// API compilation test for method declarations, generic keys, and struct values.

type userID int

type user struct {
	ID   userID
	Name string
}

type repository struct{}

func (r *repository) getUser(_ context.Context, id userID) (user, error) {
	return user{ID: id, Name: "u"}, nil
}

func (r *repository) getUsersBatch(_ context.Context, req BatchRequest[userID]) (BatchResponse[user], error) {
	items := req.Items()
	values := make([]user, len(items))
	for i, it := range items {
		values[i] = user{ID: it.Key, Name: "u"}
	}
	return OrderedOutcomes(req, values)
}

func usersSpec(t *testing.T) operation.Spec {
	t.Helper()
	return operation.MustNewSpec(
		operation.MustParseID("users.get"),
		operation.ReadOnly(),
		operation.WithOrderedResults(),
		operation.WithRequestScope(),
	)
}

func TestMustDeclareMethodCanonicalShape(t *testing.T) {
	t.Parallel()
	decl := MustDeclareMethod(
		usersSpec(t),
		(*repository).getUser,
		(*repository).getUsersBatch,
	)
	if decl.Spec().ID() != "users.get" {
		t.Errorf("spec id = %q", decl.Spec().ID())
	}
	// The stored method expressions are callable.
	repo := &repository{}
	u, err := decl.Scalar()(repo, context.Background(), 7)
	if err != nil || u.ID != 7 {
		t.Errorf("scalar call: %+v %v", u, err)
	}
	req := MustNewBatchRequest([]BatchItem[userID]{NewBatchItem[userID](1, 7), NewBatchItem[userID](2, 8)})
	resp, err := decl.Batch()(repo, context.Background(), req)
	if err != nil || resp.Len() != 2 {
		t.Errorf("batch call: len=%d err=%v", resp.Len(), err)
	}
}

func TestDeclareFunction(t *testing.T) {
	t.Parallel()
	scalar := func(_ context.Context, k int) (string, error) { return "v", nil }
	batch := func(_ context.Context, req BatchRequest[int]) (BatchResponse[string], error) {
		vals := make([]string, req.Len())
		for i := range vals {
			vals[i] = "v"
		}
		return OrderedOutcomes(req, vals)
	}
	decl, err := DeclareFunction[int, string](usersSpec(t), scalar, batch)
	if err != nil {
		t.Fatalf("DeclareFunction: %v", err)
	}
	if decl.Validate() != nil {
		t.Errorf("valid declaration reports invalid")
	}
}

func TestDeclareFunctionNilImplementation(t *testing.T) {
	t.Parallel()
	_, err := DeclareFunction[int, string](usersSpec(t), nil, nil)
	if !errors.Is(err, ErrNilImplementation) {
		t.Errorf("nil implementation error = %v, want ErrNilImplementation", err)
	}
}

func TestDeclareFunctionInvalidSpec(t *testing.T) {
	t.Parallel()
	scalar := func(_ context.Context, k int) (string, error) { return "", nil }
	batch := func(_ context.Context, _ BatchRequest[int]) (BatchResponse[string], error) {
		return BatchResponse[string]{}, nil
	}
	// The zero Spec has an empty ID and invalid semantics.
	_, err := DeclareFunction[int, string](operation.Spec{}, scalar, batch)
	if err == nil {
		t.Fatalf("invalid spec accepted")
	}
	var verr *operation.ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("error is not *operation.ValidationError: %v", err)
	}
}

// nonComparableKey is intentionally not comparable, to compile-check that the
// core request/response types and SparseOutcomes accept K that is not comparable.
type nonComparableKey struct {
	tags []string
}

func TestNonComparableKeyCompiles(t *testing.T) {
	t.Parallel()
	req := MustNewBatchRequest([]BatchItem[nonComparableKey]{
		NewBatchItem(1, nonComparableKey{tags: []string{"a"}}),
	})
	resp, err := SparseOutcomes(req,
		func(k nonComparableKey) (string, bool) { return "v", true },
		nil,
	)
	if err != nil || resp.Len() != 1 {
		t.Errorf("sparse outcomes with non-comparable key: len=%d err=%v", resp.Len(), err)
	}
}
