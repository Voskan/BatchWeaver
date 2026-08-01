package adaptive

import (
	"math"
)

// estimateBackendCosts derives a fixed per-call cost and a marginal per-item
// cost from a latency histogram and a batch-size histogram. The fixed cost is
// approximated by the low-percentile latency (dominated by call overhead at
// small batches); the per-item cost is the residual mean latency spread over the
// mean batch size. Both are floored at zero. When evidence is absent the
// estimates are zero and confidence (computed separately) is low.
func estimateBackendCosts(latency, size *Histogram) (fixed, perItem float64) {
	if latency == nil || latency.Count() == 0 {
		return 0, 0
	}
	fixed = latency.Quantile(0.10)
	meanLatency := latency.Mean()
	meanSize := 1.0
	if size != nil && size.Count() > 0 && size.Mean() > 0 {
		meanSize = size.Mean()
	}
	residual := meanLatency - fixed
	if residual < 0 {
		residual = 0
	}
	perItem = residual / meanSize
	return fixed, perItem
}

// estimateBackendCostsFromData is the encoded-histogram variant used after merge.
func estimateBackendCostsFromData(latency, size HistogramData) (fixed, perItem float64) {
	lh, err := latency.Decode()
	if err != nil {
		return 0, 0
	}
	sh, _ := size.Decode()
	return estimateBackendCosts(lh, sh)
}

// ObjectivePolicy names an explicit optimization objective. There is no single
// universal objective; every policy maps to explicit, documented weights.
type ObjectivePolicy string

const (
	// ObjectiveLatency minimizes caller-visible added latency.
	ObjectiveLatency ObjectivePolicy = "latency"
	// ObjectiveThroughput minimizes backend calls per item.
	ObjectiveThroughput ObjectivePolicy = "throughput"
	// ObjectiveBalanced balances latency and backend cost.
	ObjectiveBalanced ObjectivePolicy = "balanced"
	// ObjectiveBackendCost minimizes backend call cost.
	ObjectiveBackendCost ObjectivePolicy = "backend-cost"
	// ObjectiveDeadlineProtection prioritizes meeting deadlines.
	ObjectiveDeadlineProtection ObjectivePolicy = "deadline-protection"
	// ObjectiveCustomWeighted uses caller-supplied weights.
	ObjectiveCustomWeighted ObjectivePolicy = "custom-weighted"
)

// CostWeights are the explicit, documented multipliers applied to each cost
// component. All weights are dimensionless and default to values chosen for
// readability, not tuned magic constants.
type CostWeights struct {
	BackendFixed   float64 `json:"backend_fixed"`
	BackendPerItem float64 `json:"backend_per_item"`
	QueueDelay     float64 `json:"queue_delay"`
	DeadlineRisk   float64 `json:"deadline_risk"`
	Serialization  float64 `json:"serialization"`
	Mapping        float64 `json:"mapping"`
	Chunking       float64 `json:"chunking"`
	Retry          float64 `json:"retry"`
	Error          float64 `json:"error"`
	Fairness       float64 `json:"fairness"`
	Overload       float64 `json:"overload"`
}

// WeightsFor returns the documented default weights for an objective policy.
func WeightsFor(policy ObjectivePolicy) CostWeights {
	base := CostWeights{
		BackendFixed: 1, BackendPerItem: 1, QueueDelay: 1, DeadlineRisk: 1,
		Serialization: 1, Mapping: 1, Chunking: 1, Retry: 1, Error: 1, Fairness: 1, Overload: 1,
	}
	switch policy {
	case ObjectiveLatency:
		base.QueueDelay = 4
		base.DeadlineRisk = 4
		base.BackendPerItem = 0.5
	case ObjectiveThroughput:
		base.BackendFixed = 4
		base.QueueDelay = 0.25
	case ObjectiveBackendCost:
		base.BackendFixed = 6
		base.BackendPerItem = 2
		base.QueueDelay = 0.2
	case ObjectiveDeadlineProtection:
		base.DeadlineRisk = 8
		base.QueueDelay = 3
	case ObjectiveBalanced, ObjectiveCustomWeighted:
		// balanced uses the neutral base; custom is overridden by the caller.
	}
	return base
}

// CostInputs are the modeled quantities for one candidate policy at one
// operating point. All time-like quantities are in nanoseconds. Rates are in
// [0,1].
type CostInputs struct {
	BackendFixedNanos   float64
	BackendPerItemNanos float64
	BatchSize           float64
	QueueDelayNanos     float64
	DeadlineRisk        float64 // probability of missing a deadline
	SerializationNanos  float64
	MappingNanos        float64
	ChunkCount          float64
	RetryRate           float64
	ErrorRate           float64
	FairnessPenalty     float64
	OverloadPenalty     float64
	// DeadlineMissPenaltyNanos scales the deadline-risk term into cost units.
	DeadlineMissPenaltyNanos float64
	// ErrorPenaltyNanos scales the error-rate term into cost units.
	ErrorPenaltyNanos float64
}

// CostBreakdown is the per-component and total modeled cost, in nanoseconds of
// equivalent effective latency. It is an estimate, not a measurement.
type CostBreakdown struct {
	BackendFixed   float64 `json:"backend_fixed"`
	BackendPerItem float64 `json:"backend_per_item"`
	QueueDelay     float64 `json:"queue_delay"`
	DeadlineRisk   float64 `json:"deadline_risk"`
	Serialization  float64 `json:"serialization"`
	Mapping        float64 `json:"mapping"`
	Chunking       float64 `json:"chunking"`
	Retry          float64 `json:"retry"`
	Error          float64 `json:"error"`
	Fairness       float64 `json:"fairness"`
	Overload       float64 `json:"overload"`
	Total          float64 `json:"total"`
}

// CostModel evaluates a versioned, weighted cost objective. It is immutable and
// safe for concurrent use.
type CostModel struct {
	version string
	policy  ObjectivePolicy
	weights CostWeights
}

// NewCostModel returns a cost model for an objective policy. For
// ObjectiveCustomWeighted the caller supplies weights; other policies ignore the
// weights argument and use their documented defaults.
func NewCostModel(policy ObjectivePolicy, custom CostWeights) *CostModel {
	w := WeightsFor(policy)
	if policy == ObjectiveCustomWeighted {
		w = custom
	}
	return &CostModel{version: CostModelVersion, policy: policy, weights: w}
}

// Version returns the cost model version.
func (m *CostModel) Version() string { return m.version }

// Policy returns the objective policy.
func (m *CostModel) Policy() ObjectivePolicy { return m.policy }

// Weights returns the effective weights.
func (m *CostModel) Weights() CostWeights { return m.weights }

// Evaluate computes the cost breakdown for the given inputs. The per-item
// backend cost is amortized across the batch (fixed cost is shared), which is
// the core reason batching reduces total cost up to a point.
func (m *CostModel) Evaluate(in CostInputs) CostBreakdown {
	size := in.BatchSize
	if size < 1 {
		size = 1
	}
	w := m.weights
	b := CostBreakdown{
		BackendFixed:   w.BackendFixed * in.BackendFixedNanos / size,
		BackendPerItem: w.BackendPerItem * in.BackendPerItemNanos,
		QueueDelay:     w.QueueDelay * in.QueueDelayNanos,
		DeadlineRisk:   w.DeadlineRisk * in.DeadlineRisk * in.DeadlineMissPenaltyNanos,
		Serialization:  w.Serialization * in.SerializationNanos / size,
		Mapping:        w.Mapping * in.MappingNanos,
		Chunking:       w.Chunking * in.ChunkCount,
		Retry:          w.Retry * in.RetryRate * (in.BackendFixedNanos + in.BackendPerItemNanos),
		Error:          w.Error * in.ErrorRate * in.ErrorPenaltyNanos,
		Fairness:       w.Fairness * in.FairnessPenalty,
		Overload:       w.Overload * in.OverloadPenalty,
	}
	b.Total = b.BackendFixed + b.BackendPerItem + b.QueueDelay + b.DeadlineRisk +
		b.Serialization + b.Mapping + b.Chunking + b.Retry + b.Error + b.Fairness + b.Overload
	return b
}

// Confidence is a bounded [0,1] measure of how much a recommendation can be
// trusted, derived from evidence quality.
type Confidence float64

// Label returns a human-facing confidence label.
func (c Confidence) Label() string {
	switch {
	case c >= 0.75:
		return "high"
	case c >= 0.4:
		return "medium"
	default:
		return "low"
	}
}

// ConfidenceInputs are the evidence-quality signals for a confidence estimate.
type ConfidenceInputs struct {
	Samples       uint64
	MinSamples    uint64
	Overflowed    bool
	MissingMetric bool
	AgeSeconds    float64
	HalfLifeSecs  float64
	Stationary    bool
}

// EstimateConfidence combines evidence-quality signals into a bounded score. It
// grows with sample count, shrinks with profile age, and is penalized by missing
// metrics, histogram overflow, and non-stationarity.
func EstimateConfidence(in ConfidenceInputs) Confidence {
	if in.MinSamples == 0 {
		in.MinSamples = 100
	}
	sampleScore := math.Min(1, float64(in.Samples)/float64(in.MinSamples))
	ageScore := 1.0
	if in.HalfLifeSecs > 0 && in.AgeSeconds > 0 {
		ageScore = math.Exp(-math.Ln2 * in.AgeSeconds / in.HalfLifeSecs)
	}
	score := sampleScore * ageScore
	if in.Overflowed {
		score *= 0.8
	}
	if in.MissingMetric {
		score *= 0.5
	}
	if !in.Stationary {
		score *= 0.6
	}
	return Confidence(clamp01(score))
}

// clamp01 clamps to [0,1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// KnownObjective reports whether p is a recognized objective policy.
func KnownObjective(p ObjectivePolicy) bool {
	switch p {
	case ObjectiveLatency, ObjectiveThroughput, ObjectiveBalanced,
		ObjectiveBackendCost, ObjectiveDeadlineProtection, ObjectiveCustomWeighted:
		return true
	default:
		return false
	}
}
