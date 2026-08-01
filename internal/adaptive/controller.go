package adaptive

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// ExplorationConfig bounds safe online exploration. Exploration is disabled by
// default and, when enabled, is limited to small steps within a canary scope
// with a cooldown, so the controller never runs unconstrained experiments.
type ExplorationConfig struct {
	Enabled       bool          `json:"enabled"`
	MaxStep       float64       `json:"max_step"` // max fractional change per decision, (0,1]
	CanaryPercent int           `json:"canary_percent"`
	Cooldown      time.Duration `json:"cooldown"`
}

// ControllerConfig configures an adaptive controller.
type ControllerConfig struct {
	Mode            TuningMode
	Objective       ObjectivePolicy
	CustomWeights   CostWeights
	Bounds          HardBounds
	Guardrails      SLOGuardrails
	Exploration     ExplorationConfig
	MinSamples      uint64
	ProfileHalfLife time.Duration
	Clock           Clock
}

// withDefaults fills unset fields with conservative defaults.
func (c ControllerConfig) withDefaults() ControllerConfig {
	if c.Mode == "" {
		c.Mode = TuningShadow
	}
	if c.Objective == "" {
		c.Objective = ObjectiveBalanced
	}
	if c.Bounds == (HardBounds{}) {
		c.Bounds = DefaultBounds()
	}
	if c.MinSamples == 0 {
		c.MinSamples = 200
	}
	if c.ProfileHalfLife == 0 {
		c.ProfileHalfLife = 6 * time.Hour
	}
	if c.Clock == nil {
		c.Clock = SystemClock()
	}
	if c.Exploration.MaxStep <= 0 || c.Exploration.MaxStep > 1 {
		c.Exploration.MaxStep = 0.2
	}
	return c
}

// Controller produces bounded, explainable tuning decisions from workload
// profiles. It is safe for concurrent use. It never mutates a runtime directly;
// callers apply an accepted decision's settings through the runtime's own
// bounded, atomic settings channel.
type Controller struct {
	cfg  ControllerConfig
	cost *CostModel
	log  *DecisionLog

	mu        sync.Mutex
	disabled  bool
	lastApply map[string]time.Time
}

// NewController returns a controller for the given configuration.
func NewController(cfg ControllerConfig) *Controller {
	cfg = cfg.withDefaults()
	return &Controller{
		cfg:       cfg,
		cost:      NewCostModel(cfg.Objective, cfg.CustomWeights),
		log:       NewDecisionLog(512),
		lastApply: map[string]time.Time{},
	}
}

// Log returns the controller's decision log.
func (c *Controller) Log() *DecisionLog { return c.log }

// CostModel returns the controller's cost model.
func (c *Controller) CostModel() *CostModel { return c.cost }

// EmergencyDisable freezes the controller. Once disabled it recommends no
// changes until Enable is called. This does not depend on any profile store.
func (c *Controller) EmergencyDisable() {
	c.mu.Lock()
	c.disabled = true
	c.mu.Unlock()
}

// Enable re-enables a previously disabled controller.
func (c *Controller) Enable() {
	c.mu.Lock()
	c.disabled = false
	c.mu.Unlock()
}

// Disabled reports whether the controller is emergency-disabled.
func (c *Controller) Disabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.disabled
}

// RecommendInput carries the per-operation inputs to a recommendation.
type RecommendInput struct {
	Profile           *OperationProfile
	Current           Settings
	ProfileAgeSeconds float64
	ProfileDigest     Digest
	// PhaseChanged marks a detected workload phase change, which lowers confidence
	// and biases the controller toward conservative settings.
	PhaseChanged bool
}

// Recommend computes a decision for one operation. It always returns a decision
// (never nil): in shadow mode or when a gate fails, the decision's New equals
// Current and Applied is false, but the reasoning and evidence are still
// recorded so the outcome is explainable.
func (c *Controller) Recommend(in RecommendInput) *Decision {
	now := c.cfg.Clock.Now()
	d := &Decision{
		Operation:          opName(in.Profile, in.Current),
		Previous:           in.Current,
		New:                in.Current,
		Objective:          c.cfg.Objective,
		Mode:               c.cfg.Mode,
		Guardrails:         c.cfg.Guardrails,
		ProfileDigest:      in.ProfileDigest,
		TimestampUnixNanos: now.UnixNano(),
	}

	if c.Disabled() {
		d.Reasons = append(d.Reasons, "controller emergency-disabled; no change")
		d.finalize()
		c.log.Record(*d)
		return d
	}

	// Cold start: no profile or no evidence.
	if in.Profile == nil || in.Profile.Arrivals.LogicalCalls == 0 {
		d.Reasons = append(d.Reasons, "cold start: no workload evidence; keeping conservative defaults")
		d.Confidence = 0
		d.Diagnostics = append(d.Diagnostics, newDiag(CodeInsufficientEvidence, "info", d.Operation, "no arrivals observed"))
		d.finalize()
		c.log.Record(*d)
		return d
	}

	ev := extractEvidence(in.Profile)
	d.Evidence = ev

	// Confidence from evidence quality.
	conf := EstimateConfidence(ConfidenceInputs{
		Samples:       ev.Samples,
		MinSamples:    c.cfg.MinSamples,
		Overflowed:    histOverflow(in.Profile),
		MissingMetric: ev.P95BackendNanos == 0,
		AgeSeconds:    in.ProfileAgeSeconds,
		HalfLifeSecs:  c.cfg.ProfileHalfLife.Seconds(),
		Stationary:    !in.PhaseChanged,
	})
	d.Confidence = conf

	// Search for the minimum-cost settings.
	best, bestCost, currentCost, reasons := c.search(in.Profile, ev, in.Current)
	d.ExpectedCurrent = currentCost
	d.Expected = bestCost
	d.Reasons = append(d.Reasons, reasons...)

	// Exploration step-limiting keeps online changes small.
	if c.cfg.Mode == TuningActive && c.cfg.Exploration.Enabled {
		best = c.limitStep(in.Current, best)
		d.Canary = CanaryScope{Percent: c.cfg.Exploration.CanaryPercent, Dimension: "percentage"}
	}

	// Clamp to hard bounds (authoritative).
	clamped, clampDiags := c.cfg.Bounds.Clamp(best, d.Operation)
	d.Diagnostics = append(d.Diagnostics, clampDiags...)
	d.New = clamped

	// Phase change biases toward conservative settings and shadow-only.
	if in.PhaseChanged {
		d.Reasons = append(d.Reasons, "phase change detected: reduced confidence, conservative bias")
	}

	// Gate: sufficient evidence and confidence for active application.
	applied := c.cfg.Mode == TuningActive && d.Changed()
	if applied && (ev.Samples < c.cfg.MinSamples || conf < 0.4 || in.PhaseChanged) {
		applied = false
		d.Diagnostics = append(d.Diagnostics, newDiag(CodeInsufficientEvidence, "info", d.Operation,
			"confidence or sample count too low for active tuning; shadow only"))
		d.Reasons = append(d.Reasons, "insufficient confidence for active application; recommendation is advisory")
	}

	// Gate: modeled added latency must fit the SLO budget.
	if applied {
		if diag, ok := c.checkGuardrails(d); !ok {
			applied = false
			d.Diagnostics = append(d.Diagnostics, diag)
			d.Reasons = append(d.Reasons, "recommendation would breach an SLO guardrail; not applied")
		}
	}

	// Gate: exploration cooldown.
	if applied && c.cfg.Exploration.Enabled {
		if !c.cooldownElapsed(d.Operation, now) {
			applied = false
			d.Reasons = append(d.Reasons, "exploration cooldown not elapsed; not applied")
		}
	}

	d.Applied = applied
	d.RollbackArmed = applied && c.cfg.Guardrails.RollbackWindow > 0
	if applied {
		c.markApplied(d.Operation, now)
	}
	d.finalize()
	c.log.Record(*d)
	return d
}

// search grid-searches max_wait candidates and returns the best settings, its
// modeled cost, the current settings' modeled cost, and the human reasons.
func (c *Controller) search(p *OperationProfile, ev Evidence, current Settings) (Settings, CostBreakdown, CostBreakdown, []string) {
	lambdaPerNano := 0.0
	if mean := decodeMean(p.Arrivals.InterArrival); mean > 0 {
		lambdaPerNano = 1 / mean
	}
	slack, _ := p.Deadlines.SlackNanos.Decode()
	sizeCap := c.cfg.Bounds.MaxBatchSize
	if sizeCap < 1 {
		sizeCap = 1
	}

	evalCost := func(s Settings) CostBreakdown {
		expBatch := expectedBatchSize(float64(s.MaxWaitNanos), lambdaPerNano, float64(s.MaxBatchSize))
		queueDelay := float64(s.MaxWaitNanos) / 2
		deadlineRisk := 0.0
		if slack != nil && slack.Count() > 0 {
			threshold := float64(s.MaxWaitNanos) + ev.P95BackendNanos
			deadlineRisk = slack.CDF(threshold)
		}
		return c.cost.Evaluate(CostInputs{
			BackendFixedNanos:        ev.BackendFixedNanos,
			BackendPerItemNanos:      p.Backend.PerItemCostNanos,
			BatchSize:                expBatch,
			QueueDelayNanos:          queueDelay,
			DeadlineRisk:             deadlineRisk,
			SerializationNanos:       decodeMean(p.Backend.SerializationNanos),
			MappingNanos:             decodeMean(p.Backend.MappingNanos),
			ChunkCount:               decodeMean(p.Chunks.Count),
			ErrorRate:                rate(p.Errors.Total, p.Arrivals.LogicalCalls),
			DeadlineMissPenaltyNanos: math.Max(ev.P95BackendNanos, 1),
			ErrorPenaltyNanos:        math.Max(ev.P95BackendNanos, 1),
		})
	}

	currentCost := evalCost(current)

	best := current
	best.MaxBatchSize = sizeCap
	bestCost := evalCost(best)
	steps := 24
	lo, hi := c.cfg.Bounds.MinWaitNanos, c.cfg.Bounds.MaxWaitNanos
	if hi <= lo {
		hi = lo
	}
	for i := 0; i <= steps; i++ {
		w := lo
		if hi > lo {
			w = lo + int64(float64(hi-lo)*float64(i)/float64(steps))
		}
		cand := Settings{
			MaxWaitNanos:   w,
			MaxBatchSize:   sizeCap,
			MaxConcurrency: current.MaxConcurrency,
			MaxBatchWeight: current.MaxBatchWeight,
			ChunkSize:      current.ChunkSize,
			Mode:           chooseMode(current.Mode, w),
		}
		cost := evalCost(cand)
		if cost.Total < bestCost.Total {
			best, bestCost = cand, cost
		}
	}

	var reasons []string
	switch {
	case best.MaxWaitNanos < current.MaxWaitNanos:
		reasons = append(reasons, fmt.Sprintf("lower max_wait from %s to %s: marginal batch gain after the new wait is low",
			time.Duration(current.MaxWaitNanos), time.Duration(best.MaxWaitNanos)))
	case best.MaxWaitNanos > current.MaxWaitNanos:
		reasons = append(reasons, fmt.Sprintf("raise max_wait from %s to %s: additional batching amortizes fixed backend cost",
			time.Duration(current.MaxWaitNanos), time.Duration(best.MaxWaitNanos)))
	default:
		reasons = append(reasons, "max_wait already near cost-optimal")
	}
	if bestCost.Total < currentCost.Total {
		reasons = append(reasons, fmt.Sprintf("modeled total cost improves by %.1f%%", 100*(currentCost.Total-bestCost.Total)/math.Max(currentCost.Total, 1)))
	}
	return best, bestCost, currentCost, reasons
}

// limitStep constrains a candidate so no dimension moves more than MaxStep from
// the current value, keeping online exploration incremental.
func (c *Controller) limitStep(current, cand Settings) Settings {
	step := c.cfg.Exploration.MaxStep
	cand.MaxWaitNanos = stepInt64(current.MaxWaitNanos, cand.MaxWaitNanos, step)
	cand.MaxBatchSize = stepInt(current.MaxBatchSize, cand.MaxBatchSize, step)
	cand.MaxConcurrency = stepInt(current.MaxConcurrency, cand.MaxConcurrency, step)
	return cand
}

// checkGuardrails verifies the modeled added latency fits the budget. It returns
// a diagnostic and false when the change is rejected.
func (c *Controller) checkGuardrails(d *Decision) (Diagnostic, bool) {
	budget := c.cfg.Guardrails.AddedLatencyBudgetNanos
	if budget <= 0 {
		return Diagnostic{}, true
	}
	added := float64(d.New.MaxWaitNanos-d.Previous.MaxWaitNanos) / 2
	if added > float64(budget) {
		return newDiag(CodeGuardrailBreached, "warning", d.Operation,
			fmt.Sprintf("modeled added p95 latency %.0fns exceeds budget %dns", added, budget)), false
	}
	return Diagnostic{}, true
}

// cooldownElapsed reports whether the exploration cooldown has passed.
func (c *Controller) cooldownElapsed(operation string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	last, ok := c.lastApply[operation]
	if !ok {
		return true
	}
	return now.Sub(last) >= c.cfg.Exploration.Cooldown
}

// markApplied records the time of an applied change for cooldown accounting.
func (c *Controller) markApplied(operation string, now time.Time) {
	c.mu.Lock()
	c.lastApply[operation] = now
	c.mu.Unlock()
}

// extractEvidence derives the measured signals from an operation profile.
func extractEvidence(p *OperationProfile) Evidence {
	window := p.Sampling // sampling carries scale; arrival rate uses inter-arrival
	_ = window
	arrivalRate := 0.0
	if mean := decodeMean(p.Arrivals.InterArrival); mean > 0 {
		arrivalRate = 1e9 / mean
	}
	return Evidence{
		ArrivalRatePerSec:  arrivalRate,
		DuplicateRate:      rate(p.Duplicates.Duplicates, p.Duplicates.Duplicates+p.Duplicates.Unique),
		P95QueueDelayNanos: decodeQuantile(p.Queue.WaitNanos, 0.95),
		P95BackendNanos:    decodeQuantile(p.Backend.LatencyNanos, 0.95),
		DeadlineMissRate:   rate(p.Deadlines.Misses, p.Deadlines.WithDeadline),
		BackendFixedNanos:  p.Backend.FixedCostNanos,
		Samples:            p.Arrivals.LogicalCalls,
	}
}

// expectedBatchSize models the mean batch size given a wait window and arrival
// rate: one item plus those expected to arrive during the window, capped.
func expectedBatchSize(waitNanos, lambdaPerNano, sizeCap float64) float64 {
	if sizeCap < 1 {
		sizeCap = 1
	}
	est := 1 + lambdaPerNano*waitNanos
	if est > sizeCap {
		est = sizeCap
	}
	if est < 1 {
		est = 1
	}
	return est
}

// chooseMode selects a coarse execution mode for a candidate wait.
func chooseMode(current ExecutionMode, waitNanos int64) ExecutionMode {
	if current == ModeStaticPrefetch || current == ModeNativeBatch || current == ModePipeline {
		return current // declared modes are not overridden by the controller
	}
	if waitNanos == 0 {
		return ModeBatchOfOne
	}
	return ModeRuntimeCoalesced
}

// helpers for encoded histograms.

func decodeMean(h HistogramData) float64 {
	d, err := h.Decode()
	if err != nil || d.Count() == 0 {
		return 0
	}
	return d.Mean()
}

func decodeQuantile(h HistogramData, q float64) float64 {
	d, err := h.Decode()
	if err != nil || d.Count() == 0 {
		return 0
	}
	return d.Quantile(q)
}

func histOverflow(p *OperationProfile) bool {
	return p.Queue.WaitNanos.Overflow || p.Backend.LatencyNanos.Overflow || p.Arrivals.InterArrival.Overflow
}

func rate(num, den uint64) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func opName(p *OperationProfile, current Settings) string {
	if p != nil {
		return p.Operation
	}
	_ = current
	return ""
}

func stepInt64(current, target int64, step float64) int64 {
	maxDelta := int64(math.Abs(float64(current)) * step)
	if maxDelta < 1 {
		maxDelta = 1
	}
	delta := target - current
	if delta > maxDelta {
		return current + maxDelta
	}
	if delta < -maxDelta {
		return current - maxDelta
	}
	return target
}

func stepInt(current, target int, step float64) int {
	return int(stepInt64(int64(current), int64(target), step))
}
