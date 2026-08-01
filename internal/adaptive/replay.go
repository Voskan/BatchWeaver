package adaptive

import (
	"sort"
)

// BackendModel is the deterministic backend timing model used by simulation: a
// fixed per-call cost plus a marginal per-item cost, both in nanoseconds.
type BackendModel struct {
	FixedNanos   float64
	PerItemNanos float64
}

// PolicySim names a policy and its settings for simulation.
type PolicySim struct {
	Name     string
	Settings Settings
}

// SimResult is the modeled outcome of replaying events under one policy. Every
// figure is a model estimate, not a measurement, and is labeled as such by
// consumers.
type SimResult struct {
	Policy             string       `json:"policy"`
	Settings           Settings     `json:"settings"`
	Events             int          `json:"events"`
	Batches            int          `json:"batches"`
	BackendCalls       int          `json:"backend_calls"`
	MeanBatchSize      float64      `json:"mean_batch_size"`
	P50QueueDelayNanos float64      `json:"p50_queue_delay_nanos"`
	P95QueueDelayNanos float64      `json:"p95_queue_delay_nanos"`
	MaxQueueDelayNanos float64      `json:"max_queue_delay_nanos"`
	DeadlineMisses     int          `json:"deadline_misses"`
	CostScore          float64      `json:"cost_score"`
	Diagnostics        []Diagnostic `json:"diagnostics,omitempty"`
}

// simBatch is a formed batch awaiting a concurrency slot.
type simBatch struct {
	readyNanos int64
	items      []int // indices into events
}

// Simulate replays events under a policy against a backend model with a fake
// clock and no real backend calls. It is fully deterministic. It returns a
// BW8401 diagnostic when the input is empty (nothing to replay).
func Simulate(events []Event, sim PolicySim, backend BackendModel, cost *CostModel) SimResult {
	res := SimResult{Policy: sim.Name, Settings: sim.Settings, Events: len(events)}
	if len(events) == 0 {
		res.Diagnostics = append(res.Diagnostics, newDiag(CodeReplayIncomplete, "warning", "", "no events to replay"))
		return res
	}
	maxWait := sim.Settings.MaxWaitNanos
	maxSize := sim.Settings.MaxBatchSize
	if maxSize < 1 {
		maxSize = 1
	}
	conc := sim.Settings.MaxConcurrency
	if conc < 1 {
		conc = 1
	}

	// Group event indices by partition, preserving arrival order.
	byPart := map[string][]int{}
	var partOrder []string
	for i, e := range events {
		if _, ok := byPart[e.PartitionClass]; !ok {
			partOrder = append(partOrder, e.PartitionClass)
		}
		byPart[e.PartitionClass] = append(byPart[e.PartitionClass], i)
	}
	sort.Strings(partOrder)

	// Form batches per partition using the windowed rule.
	var batches []simBatch
	for _, part := range partOrder {
		idxs := byPart[part]
		var cur []int
		var windowStart int64
		flush := func() {
			if len(cur) == 0 {
				return
			}
			ready := windowStart + maxWait
			last := events[cur[len(cur)-1]].ArrivalNanos
			if last > ready {
				ready = last
			}
			batches = append(batches, simBatch{readyNanos: ready, items: append([]int(nil), cur...)})
			cur = cur[:0]
		}
		for _, i := range idxs {
			if len(cur) == 0 {
				windowStart = events[i].ArrivalNanos
			}
			// If this arrival is past the current window, close the window first.
			if len(cur) > 0 && events[i].ArrivalNanos > windowStart+maxWait {
				flush()
				windowStart = events[i].ArrivalNanos
			}
			cur = append(cur, i)
			if len(cur) >= maxSize {
				flush()
			}
		}
		flush()
	}
	sort.SliceStable(batches, func(i, j int) bool { return batches[i].readyNanos < batches[j].readyNanos })

	// Apply concurrency: a pool of slots each free at a time.
	slots := make([]int64, conc)
	queueDelays := NewHistogram(DefaultHistogramAccuracy)
	var totalSize int
	for _, b := range batches {
		// Earliest free slot.
		si := 0
		for k := 1; k < len(slots); k++ {
			if slots[k] < slots[si] {
				si = k
			}
		}
		start := b.readyNanos
		if slots[si] > start {
			start = slots[si]
		}
		completion := start + int64(backend.FixedNanos+backend.PerItemNanos*float64(len(b.items)))
		slots[si] = completion
		totalSize += len(b.items)
		for _, i := range b.items {
			e := events[i]
			delay := start - e.ArrivalNanos
			if delay < 0 {
				delay = 0
			}
			queueDelays.Observe(float64(delay))
			if e.DeadlineNanos > 0 && e.ArrivalNanos+e.DeadlineNanos < completion {
				res.DeadlineMisses++
			}
		}
	}

	res.Batches = len(batches)
	res.BackendCalls = len(batches)
	if len(batches) > 0 {
		res.MeanBatchSize = float64(totalSize) / float64(len(batches))
	}
	res.P50QueueDelayNanos = queueDelays.Quantile(0.50)
	res.P95QueueDelayNanos = queueDelays.Quantile(0.95)
	res.MaxQueueDelayNanos = queueDelays.Max()

	if cost != nil {
		deadlineRisk := 0.0
		if len(events) > 0 {
			deadlineRisk = float64(res.DeadlineMisses) / float64(len(events))
		}
		cb := cost.Evaluate(CostInputs{
			BackendFixedNanos:        backend.FixedNanos,
			BackendPerItemNanos:      backend.PerItemNanos,
			BatchSize:                res.MeanBatchSize,
			QueueDelayNanos:          res.P95QueueDelayNanos,
			DeadlineRisk:             deadlineRisk,
			DeadlineMissPenaltyNanos: backend.FixedNanos + backend.PerItemNanos,
			ErrorPenaltyNanos:        backend.FixedNanos + backend.PerItemNanos,
		})
		res.CostScore = cb.Total
	}
	return res
}

// PolicyComparison is a deterministic side-by-side of policies over one event
// stream, used by `tune replay` and counterfactual reporting.
type PolicyComparison struct {
	Events  int         `json:"events"`
	Results []SimResult `json:"results"`
}

// ComparePolicies simulates each policy over the same events and backend model
// and returns the results in the given order.
func ComparePolicies(events []Event, backend BackendModel, cost *CostModel, sims ...PolicySim) PolicyComparison {
	cmp := PolicyComparison{Events: len(events)}
	for _, s := range sims {
		cmp.Results = append(cmp.Results, Simulate(events, s, backend, cost))
	}
	return cmp
}

// Counterfactual returns the modeled improvement of a candidate policy over a
// baseline policy for the same workload: positive values mean the candidate is
// cheaper or faster. It is clearly a model estimate.
type Counterfactual struct {
	Baseline           string  `json:"baseline"`
	Candidate          string  `json:"candidate"`
	BackendCallDelta   int     `json:"backend_call_delta"`
	P95QueueDelayDelta float64 `json:"p95_queue_delay_delta_nanos"`
	DeadlineMissDelta  int     `json:"deadline_miss_delta"`
	CostScoreDelta     float64 `json:"cost_score_delta"`
}

// ComputeCounterfactual computes the counterfactual of candidate versus baseline.
func ComputeCounterfactual(baseline, candidate SimResult) Counterfactual {
	return Counterfactual{
		Baseline:           baseline.Policy,
		Candidate:          candidate.Policy,
		BackendCallDelta:   baseline.BackendCalls - candidate.BackendCalls,
		P95QueueDelayDelta: baseline.P95QueueDelayNanos - candidate.P95QueueDelayNanos,
		DeadlineMissDelta:  baseline.DeadlineMisses - candidate.DeadlineMisses,
		CostScoreDelta:     baseline.CostScore - candidate.CostScore,
	}
}
