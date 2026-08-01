package adaptive

import (
	"fmt"
	"math"
)

// PhaseChangeThresholds configure how large a shift in a workload signal must be
// to be treated as a phase change. Each threshold is a relative-change ratio;
// for example 0.5 means a signal that moved by more than 50% triggers on that
// dimension. Zero uses the documented default.
type PhaseChangeThresholds struct {
	ArrivalRate    float64
	Duplicate      float64
	PayloadBytes   float64
	DeadlineMix    float64
	BackendLatency float64
	ErrorRate      float64
	Partitions     float64
}

// defaultPhaseThresholds is the documented default (a doubling or halving on any
// dimension, or a large change in rates).
func defaultPhaseThresholds() PhaseChangeThresholds {
	return PhaseChangeThresholds{
		ArrivalRate: 1.0, Duplicate: 0.5, PayloadBytes: 1.0, DeadlineMix: 0.5,
		BackendLatency: 1.0, ErrorRate: 0.5, Partitions: 1.0,
	}
}

// PhaseChangeResult reports whether a phase change occurred and why.
type PhaseChangeResult struct {
	Changed bool     `json:"changed"`
	Reasons []string `json:"reasons,omitempty"`
}

// DetectPhaseChange compares a prior and current operation profile and reports a
// phase change when any monitored signal shifts beyond its threshold. It is a
// pure comparison used to lower confidence and bias the controller toward
// conservative settings; it never itself changes runtime behavior.
func DetectPhaseChange(prev, cur *OperationProfile, th PhaseChangeThresholds) PhaseChangeResult {
	if th == (PhaseChangeThresholds{}) {
		th = defaultPhaseThresholds()
	}
	var res PhaseChangeResult
	check := func(name string, was, now, threshold float64) {
		if rel := relChange(was, now); rel > threshold {
			res.Changed = true
			res.Reasons = append(res.Reasons, fmt.Sprintf("%s changed by %.0f%% (%.3g -> %.3g)", name, rel*100, was, now))
		}
	}
	if prev == nil || cur == nil {
		return res
	}
	check("arrival rate", arrivalRate(prev), arrivalRate(cur), th.ArrivalRate)
	check("duplicate rate", dupRate(prev), dupRate(cur), th.Duplicate)
	check("payload bytes", decodeMean(prev.Payloads.Bytes), decodeMean(cur.Payloads.Bytes), th.PayloadBytes)
	check("deadline mix", deadlineMix(prev), deadlineMix(cur), th.DeadlineMix)
	check("backend latency", decodeMean(prev.Backend.LatencyNanos), decodeMean(cur.Backend.LatencyNanos), th.BackendLatency)
	check("error rate", errRate(prev), errRate(cur), th.ErrorRate)
	check("partition classes", float64(prev.Partitions.DistinctClasses), float64(cur.Partitions.DistinctClasses), th.Partitions)
	return res
}

// relChange returns the symmetric relative change between two non-negative
// values, treating a change from or to zero as maximal.
func relChange(a, b float64) float64 {
	if a == b {
		return 0
	}
	denom := math.Max(math.Abs(a), 1e-9)
	if a == 0 {
		if b == 0 {
			return 0
		}
		return math.Inf(1)
	}
	return math.Abs(b-a) / denom
}

func arrivalRate(p *OperationProfile) float64 {
	if mean := decodeMean(p.Arrivals.InterArrival); mean > 0 {
		return 1e9 / mean
	}
	return 0
}

func dupRate(p *OperationProfile) float64 {
	return rate(p.Duplicates.Duplicates, p.Duplicates.Duplicates+p.Duplicates.Unique)
}

func errRate(p *OperationProfile) float64 {
	return rate(p.Errors.Total, p.Arrivals.LogicalCalls)
}

func deadlineMix(p *OperationProfile) float64 {
	return rate(p.Deadlines.WithDeadline, p.Arrivals.LogicalCalls)
}

// DecayFactor returns the exponential decay multiplier exp(-ln2 * age /
// halfLife) in [0,1]. A non-positive half-life or age returns 1 (no decay). It
// is used to age confidence and blend old and new evidence deterministically.
func DecayFactor(ageSeconds, halfLifeSeconds float64) float64 {
	if halfLifeSeconds <= 0 || ageSeconds <= 0 {
		return 1
	}
	return math.Exp(-math.Ln2 * ageSeconds / halfLifeSeconds)
}
