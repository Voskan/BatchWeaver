// Package fairnessoverload demonstrates BatchWeaver's fairness scheduling and
// overload control. Three anonymized classes share the scheduler under weighted
// fair queueing with a reserved share; the overload detector then classifies a
// saturated queue and shows the admission decisions it would make.
package fairnessoverload

import (
	"time"

	"github.com/Voskan/BatchWeaver/internal/adaptive"
)

// Result is the demonstration result.
type Result struct {
	Shares         map[string]float64
	OverloadState  string
	LowPriority    string
	CriticalAction string
}

// Demo runs a deterministic fairness scenario and an overload classification.
func Demo() Result {
	clock := adaptive.NewFakeClock(time.Unix(0, 0))
	sched := adaptive.NewFairScheduler(adaptive.FairnessConfig{
		Algorithm:           adaptive.FairWeighted,
		StarvationThreshold: 100 * time.Millisecond,
		Classes: []adaptive.ClassPolicy{
			{Class: "class_a", Weight: 3, Priority: 1, ReservedShare: 0.2},
			{Class: "class_b", Weight: 1},
			{Class: "class_c", Weight: 1},
		},
	}, clock)
	for i := 0; i < 200; i++ {
		sched.Admit("class_a", 0)
		sched.Admit("class_b", 0)
		sched.Admit("class_c", 0)
	}
	for i := 0; i < 300; i++ {
		if c, ok := sched.Next(); ok {
			sched.Serve(c)
		}
	}
	rep := sched.Report()
	shares := map[string]float64{}
	for _, c := range rep.Classes {
		shares[c.Class] = c.ServiceShare
	}

	det := adaptive.NewOverloadDetector(adaptive.OverloadConfig{Policy: adaptive.AdmitShedLowPriority})
	sig := adaptive.OverloadSignals{QueueDepthRatio: 0.99}
	low := det.Admit(adaptive.AdmissionRequest{LowPriority: true, HasFallback: true}, sig)
	crit := det.Admit(adaptive.AdmissionRequest{Critical: true}, sig)
	return Result{
		Shares:         shares,
		OverloadState:  string(det.State(sig)),
		LowPriority:    string(low.Action),
		CriticalAction: string(crit.Action),
	}
}
