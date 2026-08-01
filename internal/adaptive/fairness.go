package adaptive

import (
	"sort"
	"time"
)

// FairnessAlgorithm selects the scheduling discipline across classes.
type FairnessAlgorithm string

const (
	// FairWeighted is weighted fair queueing by class weight.
	FairWeighted FairnessAlgorithm = "weighted-fair"
	// FairDeficitRoundRobin is deficit round robin with per-class quanta.
	FairDeficitRoundRobin FairnessAlgorithm = "deficit-round-robin"
)

// Quota bounds a class's resource usage. Zero fields are unbounded on that
// dimension. Quotas are enforced at admission.
type Quota struct {
	MaxQueued       int   `json:"max_queued"`
	MaxActive       int   `json:"max_active"`
	MaxConcurrency  int   `json:"max_concurrency"`
	MaxPayloadBytes int64 `json:"max_payload_bytes"`
}

// ClassPolicy configures one anonymized fairness class. Priority classes and a
// reserved share prevent starvation of important traffic; the reserved share is
// a fraction in [0,1] of scheduling opportunities guaranteed to the class when
// it has backlog.
type ClassPolicy struct {
	Class         string  `json:"class"`
	Weight        int     `json:"weight"`
	Priority      int     `json:"priority"`
	ReservedShare float64 `json:"reserved_share"`
	Quota         Quota   `json:"quota"`
}

// FairnessConfig configures a fair scheduler.
type FairnessConfig struct {
	Algorithm           FairnessAlgorithm `json:"algorithm"`
	StarvationThreshold time.Duration     `json:"starvation_threshold"`
	Classes             []ClassPolicy     `json:"classes"`
}

// classState is the mutable per-class scheduling state.
type classState struct {
	policy     ClassPolicy
	deficit    int
	queued     int
	active     int
	served     uint64
	rejected   uint64
	lastServed time.Time
	firstWait  time.Time
	hasWaiting bool
}

// FairScheduler schedules items across anonymized classes using weighted fair
// queueing or deficit round robin, enforcing quotas and reserved capacity and
// detecting starvation. It is deterministic under a fake clock and safe for
// single-goroutine control-loop use; callers serialize access.
type FairScheduler struct {
	cfg     FairnessConfig
	clock   Clock
	classes map[string]*classState
	order   []string
	rr      int
}

// NewFairScheduler returns a scheduler for the configuration.
func NewFairScheduler(cfg FairnessConfig, clock Clock) *FairScheduler {
	if clock == nil {
		clock = SystemClock()
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = FairWeighted
	}
	s := &FairScheduler{cfg: cfg, clock: clock, classes: map[string]*classState{}}
	for _, p := range cfg.Classes {
		if p.Weight <= 0 {
			p.Weight = 1
		}
		s.classes[p.Class] = &classState{policy: p, deficit: p.Weight}
		s.order = append(s.order, p.Class)
	}
	sort.Strings(s.order)
	return s
}

// AdmitResult reports an admission decision for a fairness class.
type AdmitResult struct {
	Admitted bool
	Reason   string
	Diag     *Diagnostic
}

// classFor returns the class state, creating a default class on first sight so
// unconfigured traffic is still scheduled fairly (weight 1).
func (s *FairScheduler) classFor(class string) *classState {
	cs := s.classes[class]
	if cs == nil {
		cs = &classState{policy: ClassPolicy{Class: class, Weight: 1}, deficit: 1}
		s.classes[class] = cs
		s.order = append(s.order, class)
		sort.Strings(s.order)
	}
	return cs
}

// Admit enqueues an item for a class, enforcing the class quota. A rejected
// admission returns a BW8201 diagnostic.
func (s *FairScheduler) Admit(class string, payloadBytes int64) AdmitResult {
	cs := s.classFor(class)
	q := cs.policy.Quota
	if q.MaxQueued > 0 && cs.queued >= q.MaxQueued {
		cs.rejected++
		d := newDiag(CodeQuotaExceeded, "warning", "", "class "+class+" exceeded max queued")
		return AdmitResult{Reason: "max_queued", Diag: &d}
	}
	if q.MaxPayloadBytes > 0 && payloadBytes > q.MaxPayloadBytes {
		cs.rejected++
		d := newDiag(CodeQuotaExceeded, "warning", "", "class "+class+" exceeded max payload bytes")
		return AdmitResult{Reason: "max_payload", Diag: &d}
	}
	cs.queued++
	if !cs.hasWaiting {
		cs.firstWait = s.clock.Now()
		cs.hasWaiting = true
	}
	return AdmitResult{Admitted: true}
}

// Next selects the next class to serve. It honors reserved shares first (a class
// below its reserved share with backlog is preferred), then applies the
// configured discipline. It returns ("", false) when no class has backlog.
func (s *FairScheduler) Next() (string, bool) {
	// Reserved-capacity preference: serve the most under-served reserved class.
	if class, ok := s.reservedPick(); ok {
		return class, true
	}
	switch s.cfg.Algorithm {
	case FairDeficitRoundRobin:
		return s.deficitPick()
	default:
		return s.weightedPick()
	}
}

// reservedPick returns a backlogged class that is below its reserved share.
func (s *FairScheduler) reservedPick() (string, bool) {
	total := s.totalServed()
	best := ""
	var bestDeficit float64
	for _, class := range s.order {
		cs := s.classes[class]
		if cs.queued == 0 || cs.policy.ReservedShare <= 0 {
			continue
		}
		share := 0.0
		if total > 0 {
			share = float64(cs.served) / float64(total)
		}
		if share < cs.policy.ReservedShare {
			deficit := cs.policy.ReservedShare - share
			if best == "" || deficit > bestDeficit {
				best, bestDeficit = class, deficit
			}
		}
	}
	return best, best != ""
}

// weightedPick selects the backlogged class with the highest priority, breaking
// ties by the largest weight-normalized starvation (lowest served/weight).
func (s *FairScheduler) weightedPick() (string, bool) {
	best := ""
	var bestScore float64
	bestPriority := -1 << 30
	for _, class := range s.order {
		cs := s.classes[class]
		if cs.queued == 0 {
			continue
		}
		score := float64(cs.served+1) / float64(cs.policy.Weight)
		switch {
		case cs.policy.Priority > bestPriority:
			best, bestScore, bestPriority = class, score, cs.policy.Priority
		case cs.policy.Priority == bestPriority && (best == "" || score < bestScore):
			best, bestScore = class, score
		}
	}
	return best, best != ""
}

// deficitPick implements deficit round robin: each class accrues a quantum equal
// to its weight and may be served while its deficit is positive.
func (s *FairScheduler) deficitPick() (string, bool) {
	n := len(s.order)
	for i := 0; i < n; i++ {
		idx := (s.rr + i) % n
		class := s.order[idx]
		cs := s.classes[class]
		if cs.queued == 0 {
			continue
		}
		if cs.deficit <= 0 {
			cs.deficit += cs.policy.Weight
			continue
		}
		s.rr = (idx + 1) % n
		return class, true
	}
	// Replenish and retry once if any class has backlog.
	for _, class := range s.order {
		if s.classes[class].queued > 0 {
			s.classes[class].deficit += s.classes[class].policy.Weight
		}
	}
	for i := 0; i < n; i++ {
		idx := (s.rr + i) % n
		cs := s.classes[s.order[idx]]
		if cs.queued > 0 && cs.deficit > 0 {
			s.rr = (idx + 1) % n
			return s.order[idx], true
		}
	}
	return "", false
}

// Serve records that one item of a class was dispatched, updating deficit,
// counters, and wait accounting.
func (s *FairScheduler) Serve(class string) {
	cs := s.classFor(class)
	if cs.queued > 0 {
		cs.queued--
	}
	cs.served++
	cs.active++
	cs.deficit--
	cs.lastServed = s.clock.Now()
	if cs.queued == 0 {
		cs.hasWaiting = false
	} else {
		cs.firstWait = s.clock.Now()
	}
}

// Complete records that one active item of a class finished.
func (s *FairScheduler) Complete(class string) {
	cs := s.classFor(class)
	if cs.active > 0 {
		cs.active--
	}
}

// DetectStarvation returns the classes whose head-of-line item has waited longer
// than the starvation threshold, and emits a BW8202 diagnostic for each.
func (s *FairScheduler) DetectStarvation() ([]string, []Diagnostic) {
	if s.cfg.StarvationThreshold <= 0 {
		return nil, nil
	}
	now := s.clock.Now()
	var starved []string
	var diags []Diagnostic
	for _, class := range s.order {
		cs := s.classes[class]
		if cs.hasWaiting && now.Sub(cs.firstWait) > s.cfg.StarvationThreshold {
			starved = append(starved, class)
			diags = append(diags, newDiag(CodeStarvation, "warning", "", "class "+class+" head-of-line wait exceeds threshold"))
		}
	}
	return starved, diags
}

// totalServed returns the total served across classes.
func (s *FairScheduler) totalServed() uint64 {
	var t uint64
	for _, cs := range s.classes {
		t += cs.served
	}
	return t
}

// FairnessClassReport is the per-class fairness report row (anonymized).
type FairnessClassReport struct {
	Class        string  `json:"class"`
	ServiceShare float64 `json:"service_share"`
	Served       uint64  `json:"served"`
	Queued       int     `json:"queued"`
	Active       int     `json:"active"`
	Rejected     uint64  `json:"rejected"`
	Reserved     float64 `json:"reserved_share"`
	Starved      bool    `json:"starved"`
}

// FairnessReport is the fairness report across classes. It never contains raw
// tenant identifiers.
type FairnessReport struct {
	Algorithm string                `json:"algorithm"`
	Classes   []FairnessClassReport `json:"classes"`
}

// Report returns the current fairness report.
func (s *FairScheduler) Report() FairnessReport {
	total := s.totalServed()
	starved := map[string]bool{}
	if names, _ := s.DetectStarvation(); names != nil {
		for _, n := range names {
			starved[n] = true
		}
	}
	rep := FairnessReport{Algorithm: string(s.cfg.Algorithm)}
	for _, class := range s.order {
		cs := s.classes[class]
		share := 0.0
		if total > 0 {
			share = float64(cs.served) / float64(total)
		}
		rep.Classes = append(rep.Classes, FairnessClassReport{
			Class:        class,
			ServiceShare: share,
			Served:       cs.served,
			Queued:       cs.queued,
			Active:       cs.active,
			Rejected:     cs.rejected,
			Reserved:     cs.policy.ReservedShare,
			Starved:      starved[class],
		})
	}
	return rep
}
