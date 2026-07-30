package staticprefetch

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Voskan/BatchWeaver/examples/static-prefetch/repo"
)

// seed is the shared data set for equivalence scenarios.
var seed = map[int]string{1: "a", 2: "b", 3: "c", 4: "d"}

// TestScalarPrefetchEquivalence verifies that the transformed form is
// behaviorally equivalent to the scalar form across representative scenarios,
// while eliminating the N+1 call pattern structurally.
func TestScalarPrefetchEquivalence(t *testing.T) {
	cases := []struct {
		name string
		ids  []int
	}{
		{"empty", nil},
		{"one", []int{1}},
		{"many", []int{1, 2, 3, 4}},
		{"duplicates", []int{1, 1, 2, 2}},
		{"missing-first", []int{9, 1, 2}},
		{"missing-late", []int{1, 2, 9}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := repo.New(seed)
			pf := repo.New(seed)
			ctx := context.Background()
			orders := ordersFor(tc.ids)

			sVal, sErr := Scalar(ctx, sc, orders)
			pVal, pErr := Prefetched(ctx, pf, orders)

			if !reflect.DeepEqual(sVal, pVal) {
				t.Errorf("values differ:\n scalar=%v\n prefetch=%v", sVal, pVal)
			}
			if (sErr == nil) != (pErr == nil) {
				t.Errorf("error presence differs: scalar=%v prefetch=%v", sErr, pErr)
			}
			if sErr != nil && !errors.Is(pErr, repo.ErrNotFound) {
				t.Errorf("prefetch error identity differs: %v", pErr)
			}
			if sErr != nil && !errors.Is(sErr, pErr) && !errors.Is(pErr, sErr) {
				t.Errorf("errors.Is mismatch: scalar=%v prefetch=%v", sErr, pErr)
			}

			// Structural guarantee: at most one batch call regardless of input size.
			if pf.BatchCalls > 1 {
				t.Errorf("prefetch made %d batch calls, want <= 1", pf.BatchCalls)
			}
			if pf.ScalarCalls != 0 {
				t.Errorf("prefetch made %d scalar calls, want 0", pf.ScalarCalls)
			}
			if len(tc.ids) > 1 && tc.name == "many" && sc.ScalarCalls <= pf.BatchCalls {
				t.Errorf("expected scalar to make more backend calls than prefetch")
			}
		})
	}
}

// ordersFor builds orders referencing the given user ids.
func ordersFor(ids []int) []Order {
	orders := make([]Order, 0, len(ids))
	for i, id := range ids {
		orders = append(orders, Order{ID: i, UserID: id})
	}
	return orders
}

// BenchmarkScalar and BenchmarkPrefetched contrast backend call shape.
func BenchmarkScalar(b *testing.B) {
	s := repo.New(seed)
	orders := ordersFor([]int{1, 2, 3, 4})
	for i := 0; i < b.N; i++ {
		_, _ = Scalar(context.Background(), s, orders)
	}
}

func BenchmarkPrefetched(b *testing.B) {
	s := repo.New(seed)
	orders := ordersFor([]int{1, 2, 3, 4})
	for i := 0; i < b.N; i++ {
		_, _ = Prefetched(context.Background(), s, orders)
	}
}
