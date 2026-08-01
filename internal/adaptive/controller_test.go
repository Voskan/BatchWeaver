package adaptive

import (
	"testing"
	"time"
)

func TestControllerColdStart(t *testing.T) {
	ctrl := NewController(ControllerConfig{Mode: TuningActive, Clock: NewFakeClock(time.Unix(0, 0))})
	d := ctrl.Recommend(RecommendInput{Profile: nil, Current: Settings{MaxBatchSize: 1, MaxConcurrency: 1}})
	if d.Applied {
		t.Error("cold start must not apply a change")
	}
	if len(d.Diagnostics) == 0 || d.Diagnostics[0].Code != CodeInsufficientEvidence {
		t.Errorf("expected insufficient-evidence diagnostic, got %+v", d.Diagnostics)
	}
}

func TestControllerDeterministicDecisionID(t *testing.T) {
	b := sampleBundle(t)
	op := &b.Operations[0]
	cur := Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8}
	mk := func() *Decision {
		ctrl := NewController(ControllerConfig{Mode: TuningShadow, Clock: NewFakeClock(time.Unix(0, 0))})
		return ctrl.Recommend(RecommendInput{Profile: op, Current: cur, ProfileDigest: op.Digest})
	}
	d1, d2 := mk(), mk()
	if d1.ID != d2.ID {
		t.Errorf("decision IDs differ for identical inputs: %s vs %s", d1.ID, d2.ID)
	}
}

func TestControllerShadowNeverApplies(t *testing.T) {
	b := sampleBundle(t)
	op := &b.Operations[0]
	ctrl := NewController(ControllerConfig{Mode: TuningShadow, MinSamples: 1, Clock: NewFakeClock(time.Unix(0, 0))})
	d := ctrl.Recommend(RecommendInput{Profile: op, Current: Settings{MaxWaitNanos: int64(time.Millisecond), MaxBatchSize: 256, MaxConcurrency: 8}, ProfileDigest: op.Digest})
	if d.Applied {
		t.Error("shadow mode must never apply a change")
	}
}

func TestControllerActiveRespectsHardBounds(t *testing.T) {
	b := sampleBundle(t)
	op := &b.Operations[0]
	bounds := HardBounds{MinWaitNanos: 0, MaxWaitNanos: int64(300 * time.Microsecond), MinBatchSize: 1, MaxBatchSize: 64, MinConcurrency: 1, MaxConcurrency: 8}
	ctrl := NewController(ControllerConfig{Mode: TuningActive, MinSamples: 1, Bounds: bounds, Clock: NewFakeClock(time.Unix(0, 0))})
	d := ctrl.Recommend(RecommendInput{Profile: op, Current: Settings{MaxWaitNanos: int64(time.Millisecond), MaxBatchSize: 256, MaxConcurrency: 8}, ProfileDigest: op.Digest})
	if d.New.MaxBatchSize > 64 {
		t.Errorf("new max_batch_size %d exceeds hard bound 64", d.New.MaxBatchSize)
	}
	if d.New.MaxWaitNanos > int64(300*time.Microsecond) {
		t.Errorf("new max_wait %d exceeds hard bound", d.New.MaxWaitNanos)
	}
}

func TestControllerPhaseChangeBlocksActive(t *testing.T) {
	b := sampleBundle(t)
	op := &b.Operations[0]
	ctrl := NewController(ControllerConfig{Mode: TuningActive, MinSamples: 1, Clock: NewFakeClock(time.Unix(0, 0))})
	d := ctrl.Recommend(RecommendInput{Profile: op, Current: Settings{MaxWaitNanos: int64(time.Millisecond), MaxBatchSize: 256, MaxConcurrency: 8}, ProfileDigest: op.Digest, PhaseChanged: true})
	if d.Applied {
		t.Error("phase change must block active application")
	}
}

func TestControllerEmergencyDisable(t *testing.T) {
	b := sampleBundle(t)
	op := &b.Operations[0]
	ctrl := NewController(ControllerConfig{Mode: TuningActive, MinSamples: 1, Clock: NewFakeClock(time.Unix(0, 0))})
	ctrl.EmergencyDisable()
	d := ctrl.Recommend(RecommendInput{Profile: op, Current: Settings{MaxBatchSize: 256, MaxConcurrency: 8}, ProfileDigest: op.Digest})
	if d.Applied || d.Changed() {
		t.Error("disabled controller must not change settings")
	}
}

func TestPhaseChangeDetection(t *testing.T) {
	slow := WorkloadSpec{Pattern: PatternPoisson, Operation: "x", Count: 1000, RatePerSec: 1000, Seed: 1, DistinctKeys: 500}
	fast := WorkloadSpec{Pattern: PatternPoisson, Operation: "x", Count: 1000, RatePerSec: 8000, Seed: 1, DistinctKeys: 500}
	settings := Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 128, MaxConcurrency: 8}
	backend := BackendModel{FixedNanos: 400000, PerItemNanos: 15000}
	a := CollectSynthetic(slow, settings, backend)
	c := CollectSynthetic(fast, settings, backend)
	res := DetectPhaseChange(&a.Operations[0], &c.Operations[0], PhaseChangeThresholds{})
	if !res.Changed {
		t.Errorf("expected phase change between 1000/s and 8000/s workloads: %+v", res)
	}
}
