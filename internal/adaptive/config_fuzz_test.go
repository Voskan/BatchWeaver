package adaptive

import "testing"

const validConfig = `
runtime:
  adaptive:
    mode: shadow
    objective: latency
    profile:
      sampling_rate: 0.01
      retention: 24h
    bounds:
      max_wait:
        minimum: 0s
        maximum: 2ms
      max_batch_size:
        minimum: 1
        maximum: 512
      concurrency:
        minimum: 1
        maximum: 16
    guardrails:
      p95_queue_delay: 500us
      timeout_rate: 0.001
      error_rate_regression: 0
      rollback_window: 2m
      added_latency_budget: 250us
    exploration:
      enabled: false
      maximum_step: 0.2
      canary_percent: 5
      cooldown: 5m
fairness:
  algorithm: weighted-fair
  starvation_threshold: 100ms
overload:
  queue_high_watermark: 0.8
  queue_critical_watermark: 0.95
  policy: shed-low-priority
`

func TestParseValidConfig(t *testing.T) {
	c, err := ParseConfig([]byte(validConfig))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	cc, err := c.ControllerConfig(SystemClock())
	if err != nil {
		t.Fatalf("controller config: %v", err)
	}
	if cc.Mode != TuningShadow || cc.Objective != ObjectiveLatency {
		t.Errorf("unexpected controller config %+v", cc)
	}
	if cc.Bounds.MaxWaitNanos != int64(2_000_000) {
		t.Errorf("max_wait bound = %d, want 2ms", cc.Bounds.MaxWaitNanos)
	}
	if cc.Guardrails.RollbackWindow.String() != "2m0s" {
		t.Errorf("rollback window = %s", cc.Guardrails.RollbackWindow)
	}
	fc, err := c.FairnessConfig()
	if err != nil || fc.Algorithm != FairWeighted {
		t.Errorf("fairness config: %+v err=%v", fc, err)
	}
	if oc := c.OverloadConfig(); oc.Policy != AdmitShedLowPriority {
		t.Errorf("overload policy = %s", oc.Policy)
	}
}

func TestParseConfigRejectsUnknownField(t *testing.T) {
	bad := "runtime:\n  adaptive:\n    modee: shadow\n"
	if _, err := ParseConfig([]byte(bad)); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestParseConfigRejectsBadUnits(t *testing.T) {
	bad := "runtime:\n  adaptive:\n    bounds:\n      max_wait:\n        maximum: 2furlongs\n"
	if _, err := ParseConfig([]byte(bad)); err == nil {
		t.Fatal("expected error for bad duration units")
	}
}

func TestParseConfigRejectsBadEnum(t *testing.T) {
	bad := "runtime:\n  adaptive:\n    objective: teleport\n"
	if _, err := ParseConfig([]byte(bad)); err == nil {
		t.Fatal("expected error for unknown objective")
	}
}

func FuzzProfileUnmarshal(f *testing.F) {
	b := CollectSynthetic(WorkloadSpec{Pattern: PatternPoisson, Operation: "x", Count: 100, RatePerSec: 1000, Seed: 1, DistinctKeys: 50}, Settings{MaxWaitNanos: 1000, MaxBatchSize: 8, MaxConcurrency: 2}, BackendModel{FixedNanos: 1000})
	data, _ := Marshal(b)
	f.Add(data)
	f.Add([]byte(`{"magic":"batchweaver-profile"}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(_ *testing.T, data []byte) {
		// Must never panic; errors are acceptable.
		_, _ = Unmarshal(data)
	})
}

func FuzzHistogramDecode(f *testing.F) {
	h := NewHistogram(0.02)
	for i := 0; i < 100; i++ {
		h.Observe(float64(i))
	}
	f.Add(int64(len(h.Encode().BucketIndex)))
	f.Fuzz(func(t *testing.T, n int64) {
		d := HistogramData{Accuracy: 0.02, Count: uint64(n)}
		if n >= 0 && n < 5 {
			for i := int64(0); i < n; i++ {
				d.BucketIndex = append(d.BucketIndex, int(i))
				d.BucketCounts = append(d.BucketCounts, 1)
			}
			d.Count = uint64(n)
		}
		// Must never panic.
		_, _ = d.Decode()
	})
}
