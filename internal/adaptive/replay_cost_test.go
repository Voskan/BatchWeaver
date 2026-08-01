package adaptive

import (
	"testing"
	"time"
)

func TestCostModelAmortization(t *testing.T) {
	m := NewCostModel(ObjectiveThroughput, CostWeights{})
	small := m.Evaluate(CostInputs{BackendFixedNanos: 1000, BatchSize: 1})
	large := m.Evaluate(CostInputs{BackendFixedNanos: 1000, BatchSize: 100})
	if large.BackendFixed >= small.BackendFixed {
		t.Errorf("larger batch should amortize fixed cost: %.1f vs %.1f", large.BackendFixed, small.BackendFixed)
	}
}

func TestObjectiveWeightsDiffer(t *testing.T) {
	lat := WeightsFor(ObjectiveLatency)
	thr := WeightsFor(ObjectiveThroughput)
	if lat.QueueDelay <= thr.QueueDelay {
		t.Error("latency objective should weight queue delay more than throughput")
	}
}

func TestConfidence(t *testing.T) {
	high := EstimateConfidence(ConfidenceInputs{Samples: 10000, MinSamples: 100, Stationary: true, HalfLifeSecs: 3600})
	low := EstimateConfidence(ConfidenceInputs{Samples: 5, MinSamples: 100, Stationary: false, MissingMetric: true, HalfLifeSecs: 3600, AgeSeconds: 7200})
	if high <= low {
		t.Errorf("high-evidence confidence %.3f should exceed low %.3f", high, low)
	}
	if high.Label() != "high" {
		t.Errorf("expected high label, got %q", high.Label())
	}
}

func TestSimulateDeterminismAndBatching(t *testing.T) {
	events := GenerateWorkload(WorkloadSpec{Pattern: PatternPoisson, Operation: "x", Count: 3000, RatePerSec: 8000, Seed: 3, DistinctKeys: 1500})
	sim := PolicySim{Name: "p", Settings: Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8}}
	backend := BackendModel{FixedNanos: 400000, PerItemNanos: 15000}
	cost := NewCostModel(ObjectiveBalanced, CostWeights{})
	r1 := Simulate(events, sim, backend, cost)
	r2 := Simulate(events, sim, backend, cost)
	if r1.BackendCalls != r2.BackendCalls || r1.P95QueueDelayNanos != r2.P95QueueDelayNanos ||
		r1.DeadlineMisses != r2.DeadlineMisses || r1.CostScore != r2.CostScore {
		t.Error("simulation must be deterministic for identical inputs")
	}
	if r1.BackendCalls >= r1.Events {
		t.Errorf("batching should reduce backend calls below event count: %d vs %d", r1.BackendCalls, r1.Events)
	}
}

func TestComparePoliciesLatencyVsThroughput(t *testing.T) {
	events := GenerateWorkload(WorkloadSpec{Pattern: PatternPoisson, Operation: "x", Count: 3000, RatePerSec: 8000, Seed: 5, DistinctKeys: 1500})
	backend := BackendModel{FixedNanos: 500000, PerItemNanos: 10000}
	cmp := ComparePolicies(events, backend, nil,
		PolicySim{Name: "latency", Settings: Settings{MaxWaitNanos: int64(50 * time.Microsecond), MaxBatchSize: 64, MaxConcurrency: 8}},
		PolicySim{Name: "throughput", Settings: Settings{MaxWaitNanos: int64(2 * time.Millisecond), MaxBatchSize: 512, MaxConcurrency: 8}},
	)
	lat, thr := cmp.Results[0], cmp.Results[1]
	if lat.BackendCalls <= thr.BackendCalls {
		t.Errorf("latency policy should make more backend calls than throughput: %d vs %d", lat.BackendCalls, thr.BackendCalls)
	}
	if lat.P95QueueDelayNanos >= thr.P95QueueDelayNanos {
		t.Errorf("latency policy should have lower p95 queue delay: %.0f vs %.0f", lat.P95QueueDelayNanos, thr.P95QueueDelayNanos)
	}
}

func TestSimulateEmptyInput(t *testing.T) {
	r := Simulate(nil, PolicySim{Name: "p"}, BackendModel{}, nil)
	if len(r.Diagnostics) == 0 || r.Diagnostics[0].Code != CodeReplayIncomplete {
		t.Errorf("expected replay-incomplete diagnostic, got %+v", r.Diagnostics)
	}
}

func TestSyntheticDeterminism(t *testing.T) {
	spec := WorkloadSpec{Pattern: PatternBursty, Operation: "x", Count: 500, RatePerSec: 4000, Seed: 42, DistinctKeys: 250}
	a := GenerateWorkload(spec)
	b := GenerateWorkload(spec)
	if len(a) != len(b) {
		t.Fatalf("length mismatch %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("event %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}
