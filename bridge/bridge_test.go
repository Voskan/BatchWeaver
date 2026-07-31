package bridge_test

import (
	"context"
	"errors"
	"testing"

	batchweaver "github.com/Voskan/BatchWeaver"
	"github.com/Voskan/BatchWeaver/bridge"
	"github.com/Voskan/BatchWeaver/operation"
	batchruntime "github.com/Voskan/BatchWeaver/runtime"
)

// repo is a fake store that counts scalar and batch calls.
type repo struct {
	data        map[int]string
	scalarCalls int
	batchCalls  int
}

var errMissing = errors.New("missing")

func (r *repo) get(_ context.Context, id int) (string, error) {
	r.scalarCalls++
	v, ok := r.data[id]
	if !ok {
		return "", errMissing
	}
	return v, nil
}

func newOp(r *repo) bridge.Operation[*repo, int, string] {
	return bridge.Operation[*repo, int, string]{
		OpID:   "users.get",
		Scalar: func(ctx context.Context, rec *repo, k int) (string, error) { return rec.get(ctx, k) },
	}
}

func TestCallFallbackWithoutScope(t *testing.T) {
	t.Parallel()
	r := &repo{data: map[int]string{1: "a", 2: "b"}}
	op := newOp(r)
	v, err := op.Call(context.Background(), r, 1)
	if err != nil || v != "a" {
		t.Fatalf("got (%q,%v), want (a,nil)", v, err)
	}
	if r.scalarCalls != 1 {
		t.Errorf("scalar calls = %d, want 1", r.scalarCalls)
	}
	if _, err := op.Call(context.Background(), r, 9); !errors.Is(err, errMissing) {
		t.Errorf("missing key error = %v, want errMissing", err)
	}
}

func TestCallCoalescedThroughRuntime(t *testing.T) {
	t.Parallel()
	r := &repo{data: map[int]string{1: "a", 2: "b", 3: "c"}}
	op := newOp(r)

	engine, err := batchruntime.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	spec := operation.MustNewSpec(operation.MustParseID("users.get"), operation.ReadOnly(), operation.WithOrderedResults())
	provider := batchruntime.ProviderFunc[int, string](func(_ context.Context, req batchweaver.BatchRequest[int]) (batchweaver.BatchResponse[string], error) {
		r.batchCalls++
		items := req.Items()
		vals := make([]string, len(items))
		for i, it := range items {
			vals[i] = r.data[it.Key]
		}
		return batchweaver.OrderedOutcomes(req, vals)
	})
	bound, err := batchruntime.Bind[int, string](engine, decl{spec}, batchruntime.Binding[int, string]{
		Provider: provider,
		Keys:     batchruntime.ComparableKeys[int](),
		Fallback: batchruntime.ScalarFallbackFunc[int, string](func(ctx context.Context, k int) (string, error) { return r.get(ctx, k) }),
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := batchruntime.Run(engine, context.Background(), func(ctx context.Context) ([]string, error) {
		ctx = bridge.WithOperation(ctx, "users.get", bound)
		var out []string
		for _, id := range []int{1, 2, 3} {
			v, err := op.Call(ctx, r, id)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("coalesced result[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// decl adapts a spec to runtime.Declaration.
type decl struct{ spec operation.Spec }

func (d decl) Spec() operation.Spec { return d.spec }

func TestFlushWithoutScopeIsNoop(t *testing.T) {
	t.Parallel()
	if err := bridge.Flush(context.Background()); err != nil {
		t.Errorf("flush without scope = %v, want nil", err)
	}
}
