package adaptive

import (
	"testing"
	"time"
)

func BenchmarkHistogramObserve(b *testing.B) {
	h := NewHistogram(0.02)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Observe(float64(i % 100000))
	}
}

func BenchmarkCollectorRecordCall(b *testing.B) {
	c := NewCollector(CollectorOptions{Mode: CollectHistograms, Clock: NewFakeClock(time.Unix(0, 0))})
	o := CallObservation{Operation: "x", PartitionRaw: "p", TenantRaw: "t", QueueWait: time.Microsecond, PayloadBytes: 128, Mode: ModeRuntimeCoalesced}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.RecordCall(o)
	}
}

func BenchmarkControllerRecommend(b *testing.B) {
	bundle := benchBundle()
	op := &bundle.Operations[0]
	ctrl := NewController(ControllerConfig{Mode: TuningShadow, Clock: NewFakeClock(time.Unix(0, 0))})
	cur := Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ctrl.Recommend(RecommendInput{Profile: op, Current: cur, ProfileDigest: op.Digest})
	}
}

func BenchmarkSimulate(b *testing.B) {
	events := GenerateWorkload(WorkloadSpec{Pattern: PatternPoisson, Operation: "x", Count: 2000, RatePerSec: 8000, Seed: 1, DistinctKeys: 1000})
	sim := PolicySim{Name: "p", Settings: Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8}}
	backend := BackendModel{FixedNanos: 400000, PerItemNanos: 15000}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Simulate(events, sim, backend, nil)
	}
}

func BenchmarkFairSchedulerServe(b *testing.B) {
	s := NewFairScheduler(FairnessConfig{Algorithm: FairDeficitRoundRobin, Classes: []ClassPolicy{{Class: "a", Weight: 2}, {Class: "b", Weight: 1}}}, NewFakeClock(time.Unix(0, 0)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Admit("a", 0)
		s.Admit("b", 0)
		if c, ok := s.Next(); ok {
			s.Serve(c)
		}
	}
}

func benchBundle() *ProfileBundle {
	return CollectSynthetic(
		WorkloadSpec{Pattern: PatternPoisson, Operation: "users.get", Count: 2000, RatePerSec: 8000, Seed: 1, DistinctKeys: 1000, TenantClasses: 3},
		Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8},
		BackendModel{FixedNanos: 400000, PerItemNanos: 15000},
	)
}
