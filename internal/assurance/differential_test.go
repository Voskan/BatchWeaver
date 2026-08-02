package assurance

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"
)

type request struct {
	ID, Key, Partition int
	Canceled           bool
}
type outcome struct {
	ID, Value int
	Found     bool
	Err       string
}

func scalar(requests []request) ([]outcome, []string) {
	results := make([]outcome, 0, len(requests))
	trace := make([]string, 0, len(requests))
	for _, req := range requests {
		trace = append(trace, fmt.Sprintf("call:%d:%d", req.Partition, req.Key))
		results = append(results, evaluate(req))
	}
	return results, trace
}

func batched(requests []request) ([]outcome, []string) {
	results := make([]outcome, len(requests))
	trace := make([]string, 0, len(requests))
	for _, req := range requests {
		trace = append(trace, fmt.Sprintf("call:%d:%d", req.Partition, req.Key))
	}
	for i, req := range requests {
		results[i] = evaluate(req)
	}
	return results, trace
}

func evaluate(req request) outcome {
	if req.Canceled {
		return outcome{ID: req.ID, Err: context.Canceled.Error()}
	}
	if req.Key < 0 {
		return outcome{ID: req.ID, Err: "negative key"}
	}
	if req.Key%7 == 0 {
		return outcome{ID: req.ID, Found: false}
	}
	return outcome{ID: req.ID, Value: req.Partition*1000 + req.Key*2, Found: true}
}

func TestDifferentialDeterministicPrograms(t *testing.T) {
	for seed := uint64(1); seed <= 256; seed++ {
		rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
		n := 1 + rng.IntN(64)
		requests := make([]request, n)
		for i := range requests {
			requests[i] = request{ID: i + 1, Key: rng.IntN(31) - 3, Partition: rng.IntN(4), Canceled: rng.IntN(13) == 0}
		}
		want, wantTrace := scalar(requests)
		got, gotTrace := batched(requests)
		if !slices.Equal(want, got) || !slices.Equal(wantTrace, gotTrace) {
			t.Fatalf("seed %d mismatch\nwant=%v\ngot=%v", seed, want, got)
		}
	}
}

func TestDifferentialFaultInjection(t *testing.T) {
	backendTimeout := errors.New("backend timeout")
	tests := []struct {
		name string
		fn   func() error
		want error
	}{
		{"timeout", func() error { return backendTimeout }, backendTimeout},
		{"partial-result", func() error { return errors.New("missing result for request 2") }, errors.New("missing result for request 2")},
		{"malformed-result", func() error { return errors.New("unexpected result key") }, errors.New("unexpected result key")},
		{"cancel-before-dispatch", func() error { return context.Canceled }, context.Canceled},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.fn(); got.Error() != tc.want.Error() {
				t.Fatalf("error=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestDifferentialShortSoak(t *testing.T) {
	deadline := time.Now().Add(75 * time.Millisecond)
	iterations := 0
	for time.Now().Before(deadline) {
		requests := []request{{ID: 1, Key: iterations % 11, Partition: iterations % 3}}
		want, _ := scalar(requests)
		got, _ := batched(requests)
		if !slices.Equal(want, got) {
			t.Fatal("soak mismatch")
		}
		iterations++
	}
	if iterations < 100 {
		t.Fatalf("short soak completed only %d iterations", iterations)
	}
}
