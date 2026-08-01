package adaptive

import (
	"fmt"
	"sort"
	"sync"
)

// TuningMode is the operating mode of the adaptive controller.
type TuningMode string

const (
	// TuningOff disables the controller entirely.
	TuningOff TuningMode = "off"
	// TuningShadow computes recommendations but never changes runtime settings.
	TuningShadow TuningMode = "shadow"
	// TuningActive applies bounded recommendations, subject to every safety gate.
	TuningActive TuningMode = "active"
)

// KnownTuningMode reports whether m is a recognized tuning mode.
func KnownTuningMode(m TuningMode) bool {
	switch m {
	case TuningOff, TuningShadow, TuningActive:
		return true
	default:
		return false
	}
}

// CanaryScope describes the limited population a change is applied to first.
type CanaryScope struct {
	// Percent is the fraction of scopes the change applies to, in [0,100].
	Percent int `json:"percent"`
	// Dimension names what the canary is scoped by (percentage, operation,
	// partition-class, tenant-class, process). Raw identifiers never appear.
	Dimension string `json:"dimension"`
}

// Evidence summarizes the measured signals behind a decision. It never contains
// raw keys or tenant identifiers.
type Evidence struct {
	ArrivalRatePerSec  float64  `json:"arrival_rate_per_sec"`
	DuplicateRate      float64  `json:"duplicate_rate"`
	P95QueueDelayNanos float64  `json:"p95_queue_delay_nanos"`
	P95BackendNanos    float64  `json:"p95_backend_nanos"`
	DeadlineMissRate   float64  `json:"deadline_miss_rate"`
	BackendFixedNanos  float64  `json:"backend_fixed_nanos"`
	Samples            uint64   `json:"samples"`
	Notes              []string `json:"notes,omitempty"`
}

// Decision is an immutable record of one adaptive recommendation or change. Its
// ID is content-addressed over everything except wall-clock timestamps, so the
// same evidence and settings always yield the same decision ID.
type Decision struct {
	ID                string          `json:"id"`
	Operation         string          `json:"operation"`
	Previous          Settings        `json:"previous"`
	New               Settings        `json:"new"`
	Objective         ObjectivePolicy `json:"objective"`
	Mode              TuningMode      `json:"tuning_mode"`
	Confidence        Confidence      `json:"confidence"`
	ConfidenceLabel   string          `json:"confidence_label"`
	Evidence          Evidence        `json:"evidence"`
	Reasons           []string        `json:"reasons"`
	Expected          CostBreakdown   `json:"expected_cost"`
	ExpectedCurrent   CostBreakdown   `json:"expected_cost_current"`
	Guardrails        SLOGuardrails   `json:"guardrails"`
	Canary            CanaryScope     `json:"canary"`
	RollbackArmed     bool            `json:"rollback_armed"`
	Applied           bool            `json:"applied"`
	ProfileDigest     Digest          `json:"profile_digest"`
	ControllerVersion string          `json:"controller_version"`
	CostModelVersion  string          `json:"cost_model_version"`
	Diagnostics       []Diagnostic    `json:"diagnostics,omitempty"`
	// SourceCallSites lists original call-site references (from source maps) the
	// change affects, so a recommendation identifies where in the source it
	// applies. It is optional.
	SourceCallSites []string `json:"source_call_sites,omitempty"`
	// TimestampUnixNanos is wall-clock metadata excluded from the decision ID.
	TimestampUnixNanos int64 `json:"timestamp_unix_nanos"`
}

// computeID content-addresses the decision over its semantic fields.
func (d *Decision) computeID() string {
	parts := []string{
		"decision", d.Operation, string(d.Objective), string(d.Mode),
		string(d.ProfileDigest), d.ControllerVersion, d.CostModelVersion,
		fmt.Sprintf("prev=%+v", d.Previous),
		fmt.Sprintf("new=%+v", d.New),
		fmt.Sprintf("canary=%d/%s", d.Canary.Percent, d.Canary.Dimension),
	}
	sort.Strings(d.Reasons)
	parts = append(parts, d.Reasons...)
	return shortID("bwdec", parts...)
}

// finalize stamps the decision ID and confidence label.
func (d *Decision) finalize() {
	d.ControllerVersion = ControllerVersion
	d.CostModelVersion = CostModelVersion
	d.ConfidenceLabel = d.Confidence.Label()
	d.ID = d.computeID()
}

// Changed reports whether the decision proposes any settings change.
func (d *Decision) Changed() bool {
	return d.New != d.Previous
}

// DecisionLog is a bounded, concurrency-safe ring of recent decisions for
// explainability. It holds no secrets.
type DecisionLog struct {
	mu    sync.Mutex
	max   int
	items []Decision
	byID  map[string]Decision
}

// NewDecisionLog returns a decision log retaining up to max entries.
func NewDecisionLog(max int) *DecisionLog {
	if max <= 0 {
		max = 256
	}
	return &DecisionLog{max: max, byID: make(map[string]Decision)}
}

// Record appends a decision, evicting the oldest when the bound is reached.
func (l *DecisionLog) Record(d Decision) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = append(l.items, d)
	l.byID[d.ID] = d
	if len(l.items) > l.max {
		evicted := l.items[0]
		l.items = l.items[1:]
		delete(l.byID, evicted.ID)
	}
}

// Get returns a decision by ID.
func (l *DecisionLog) Get(id string) (Decision, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	d, ok := l.byID[id]
	return d, ok
}

// Recent returns up to n most recent decisions, newest last.
func (l *DecisionLog) Recent(n int) []Decision {
	l.mu.Lock()
	defer l.mu.Unlock()
	if n <= 0 || n > len(l.items) {
		n = len(l.items)
	}
	out := make([]Decision, n)
	copy(out, l.items[len(l.items)-n:])
	return out
}

// ForOperation returns the most recent decisions for one operation, newest last.
func (l *DecisionLog) ForOperation(operation string, n int) []Decision {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []Decision
	for _, d := range l.items {
		if d.Operation == operation {
			out = append(out, d)
		}
	}
	if n > 0 && len(out) > n {
		out = out[len(out)-n:]
	}
	return out
}
