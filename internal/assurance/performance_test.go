//go:build !race

package assurance

import (
	"testing"
	"time"
)

func TestPerformanceBudgetDifferentialHarness(t *testing.T) {
	requests := make([]request, 1024)
	for i := range requests {
		requests[i] = request{ID: i + 1, Key: i % 97, Partition: i % 8}
	}
	allocs := testing.AllocsPerRun(20, func() { batched(requests) })
	if allocs > 1100 {
		t.Fatalf("allocations/run %.1f exceed budget 1100", allocs)
	}
	start := time.Now()
	const samples = 25
	for range samples {
		batched(requests)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("25 samples took %s, budget 2s", elapsed)
	}
}
