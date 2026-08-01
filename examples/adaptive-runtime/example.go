// Package adaptiveruntime demonstrates BatchWeaver's bounded, explainable
// adaptive controller. It collects a privacy-safe profile from a deterministic
// synthetic workload, then asks the controller (in shadow mode) for a
// recommendation. No runtime settings are changed and no backend is contacted.
package adaptiveruntime

import (
	"time"

	"github.com/Voskan/BatchWeaver/internal/adaptive"
)

// Recommendation is the demonstration result: the current and recommended
// scheduler settings for one operation, with the controller's confidence.
type Recommendation struct {
	Operation     string
	CurrentWait   time.Duration
	RecommendWait time.Duration
	Confidence    string
	Applied       bool
}

// Demo profiles a synthetic workload and returns the controller's shadow-mode
// recommendation for the operation. It is fully deterministic.
func Demo() Recommendation {
	spec := adaptive.WorkloadSpec{
		Pattern: adaptive.PatternPoisson, Operation: "users.get", Count: 4000,
		RatePerSec: 8000, Seed: 1, DistinctKeys: 2000, TenantClasses: 3,
		DeadlineFraction: 0.5, DeadlineNanos: int64(2 * time.Millisecond), PayloadBytes: 256,
	}
	current := adaptive.Settings{MaxWaitNanos: int64(500 * time.Microsecond), MaxBatchSize: 256, MaxConcurrency: 8}
	backend := adaptive.BackendModel{FixedNanos: 500_000, PerItemNanos: 20_000}
	bundle := adaptive.CollectSynthetic(spec, current, backend)

	ctrl := adaptive.NewController(adaptive.ControllerConfig{
		Mode: adaptive.TuningShadow, Objective: adaptive.ObjectiveBalanced,
		Bounds: adaptive.DefaultBounds(), Clock: adaptive.NewFakeClock(time.Unix(0, 0)),
	})
	op := &bundle.Operations[0]
	d := ctrl.Recommend(adaptive.RecommendInput{Profile: op, Current: current, ProfileDigest: op.Digest})
	return Recommendation{
		Operation:     d.Operation,
		CurrentWait:   d.Previous.MaxWait(),
		RecommendWait: d.New.MaxWait(),
		Confidence:    d.ConfidenceLabel,
		Applied:       d.Applied,
	}
}
