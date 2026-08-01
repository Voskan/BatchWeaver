package adaptive

import (
	"fmt"
	"time"

	"github.com/goccy/go-yaml"
)

// Config is the strict, versioned configuration for the adaptive scheduling,
// fairness, and overload layer. It is parsed from YAML with unknown-field
// rejection and validated for units and ranges. Durations are strings such as
// "500us", "2ms", or "24h".
type Config struct {
	Runtime  RuntimeSection  `yaml:"runtime"`
	Fairness FairnessSection `yaml:"fairness"`
	Overload OverloadSection `yaml:"overload"`
}

// RuntimeSection carries the adaptive controller configuration.
type RuntimeSection struct {
	Adaptive Section `yaml:"adaptive"`
}

// Section configures the controller.
type Section struct {
	Mode        string             `yaml:"mode"`
	Objective   string             `yaml:"objective"`
	Profile     ProfileSection     `yaml:"profile"`
	Bounds      BoundsSection      `yaml:"bounds"`
	Guardrails  GuardrailsSection  `yaml:"guardrails"`
	Exploration ExplorationSection `yaml:"exploration"`
}

// ProfileSection configures profile collection.
type ProfileSection struct {
	Mode         string  `yaml:"mode"`
	SamplingRate float64 `yaml:"sampling_rate"`
	Retention    string  `yaml:"retention"`
}

// RangeSection is a min/max range with duration or integer values, kept as
// strings so units are validated explicitly.
type RangeSection struct {
	Minimum string `yaml:"minimum"`
	Maximum string `yaml:"maximum"`
}

// BoundsSection configures hard bounds.
type BoundsSection struct {
	MaxWait      RangeSection `yaml:"max_wait"`
	MaxBatchSize RangeSection `yaml:"max_batch_size"`
	Concurrency  RangeSection `yaml:"concurrency"`
}

// GuardrailsSection configures SLO guardrails.
type GuardrailsSection struct {
	P95QueueDelay       string  `yaml:"p95_queue_delay"`
	TimeoutRate         float64 `yaml:"timeout_rate"`
	ErrorRateRegression float64 `yaml:"error_rate_regression"`
	RollbackWindow      string  `yaml:"rollback_window"`
	AddedLatencyBudget  string  `yaml:"added_latency_budget"`
}

// ExplorationSection configures exploration.
type ExplorationSection struct {
	Enabled       bool    `yaml:"enabled"`
	MaximumStep   float64 `yaml:"maximum_step"`
	CanaryPercent int     `yaml:"canary_percent"`
	Cooldown      string  `yaml:"cooldown"`
}

// FairnessSection configures fairness.
type FairnessSection struct {
	Algorithm           string                 `yaml:"algorithm"`
	StarvationThreshold string                 `yaml:"starvation_threshold"`
	Classes             []FairnessClassSection `yaml:"classes"`
}

// FairnessClassSection configures one fairness class.
type FairnessClassSection struct {
	Class         string  `yaml:"class"`
	Weight        int     `yaml:"weight"`
	Priority      int     `yaml:"priority"`
	ReservedShare float64 `yaml:"reserved_share"`
}

// OverloadSection configures overload control.
type OverloadSection struct {
	QueueHighWatermark     float64 `yaml:"queue_high_watermark"`
	QueueCriticalWatermark float64 `yaml:"queue_critical_watermark"`
	Policy                 string  `yaml:"policy"`
}

// ParseConfig parses and validates adaptive configuration from YAML. Unknown
// fields are rejected so typos cannot silently disable a guardrail.
func ParseConfig(data []byte) (*Config, error) {
	var c Config
	if err := yaml.UnmarshalWithOptions(data, &c, yaml.Strict()); err != nil {
		return nil, fmt.Errorf("adaptive: parse config: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate checks units, ranges, and enum values.
func (c *Config) Validate() error {
	a := c.Runtime.Adaptive
	if a.Mode != "" && !KnownTuningMode(TuningMode(a.Mode)) {
		return fmt.Errorf("adaptive: unknown mode %q", a.Mode)
	}
	if a.Objective != "" && !KnownObjective(ObjectivePolicy(a.Objective)) {
		return fmt.Errorf("adaptive: unknown objective %q", a.Objective)
	}
	if a.Profile.Mode != "" && !knownCollectionMode(a.Profile.Mode) {
		return fmt.Errorf("adaptive: unknown profile mode %q", a.Profile.Mode)
	}
	if a.Profile.SamplingRate < 0 || a.Profile.SamplingRate > 1 {
		return fmt.Errorf("adaptive: sampling_rate must be in [0,1]")
	}
	if _, err := c.Bounds(); err != nil {
		return err
	}
	if _, err := c.Guardrails(); err != nil {
		return err
	}
	if c.Fairness.Algorithm != "" && !knownFairnessAlgo(c.Fairness.Algorithm) {
		return fmt.Errorf("adaptive: unknown fairness algorithm %q", c.Fairness.Algorithm)
	}
	if c.Overload.Policy != "" && !knownAdmissionPolicy(c.Overload.Policy) {
		return fmt.Errorf("adaptive: unknown overload policy %q", c.Overload.Policy)
	}
	if c.Overload.QueueHighWatermark < 0 || c.Overload.QueueHighWatermark > 1 ||
		c.Overload.QueueCriticalWatermark < 0 || c.Overload.QueueCriticalWatermark > 1 {
		return fmt.Errorf("adaptive: overload watermarks must be in [0,1]")
	}
	return nil
}

// Bounds converts the config section into HardBounds, validating units.
func (c *Config) Bounds() (HardBounds, error) {
	b := c.Runtime.Adaptive.Bounds
	h := DefaultBounds()
	if v, ok, err := parseDurRange(b.MaxWait.Minimum); err != nil {
		return h, err
	} else if ok {
		h.MinWaitNanos = v.Nanoseconds()
	}
	if v, ok, err := parseDurRange(b.MaxWait.Maximum); err != nil {
		return h, err
	} else if ok {
		h.MaxWaitNanos = v.Nanoseconds()
	}
	if v, ok, err := parseIntRange(b.MaxBatchSize.Minimum); err != nil {
		return h, err
	} else if ok {
		h.MinBatchSize = v
	}
	if v, ok, err := parseIntRange(b.MaxBatchSize.Maximum); err != nil {
		return h, err
	} else if ok {
		h.MaxBatchSize = v
	}
	if v, ok, err := parseIntRange(b.Concurrency.Minimum); err != nil {
		return h, err
	} else if ok {
		h.MinConcurrency = v
	}
	if v, ok, err := parseIntRange(b.Concurrency.Maximum); err != nil {
		return h, err
	} else if ok {
		h.MaxConcurrency = v
	}
	if err := h.Validate(); err != nil {
		return h, err
	}
	return h, nil
}

// Guardrails converts the config section into SLOGuardrails, validating units.
func (c *Config) Guardrails() (SLOGuardrails, error) {
	g := c.Runtime.Adaptive.Guardrails
	var out SLOGuardrails
	if v, ok, err := parseDurRange(g.P95QueueDelay); err != nil {
		return out, err
	} else if ok {
		out.P95QueueDelayNanos = v.Nanoseconds()
	}
	if v, ok, err := parseDurRange(g.AddedLatencyBudget); err != nil {
		return out, err
	} else if ok {
		out.AddedLatencyBudgetNanos = v.Nanoseconds()
	}
	if v, ok, err := parseDurRange(g.RollbackWindow); err != nil {
		return out, err
	} else if ok {
		out.RollbackWindow = v
	}
	out.TimeoutRate = g.TimeoutRate
	out.ErrorRateRegression = g.ErrorRateRegression
	return out, nil
}

// ControllerConfig builds a controller configuration from the parsed config and
// a clock.
func (c *Config) ControllerConfig(clock Clock) (ControllerConfig, error) {
	bounds, err := c.Bounds()
	if err != nil {
		return ControllerConfig{}, err
	}
	guard, err := c.Guardrails()
	if err != nil {
		return ControllerConfig{}, err
	}
	a := c.Runtime.Adaptive
	cd, err := parseDurOpt(a.Exploration.Cooldown)
	if err != nil {
		return ControllerConfig{}, err
	}
	cc := ControllerConfig{
		Mode:       TuningMode(orDefault(a.Mode, string(TuningShadow))),
		Objective:  ObjectivePolicy(orDefault(a.Objective, string(ObjectiveBalanced))),
		Bounds:     bounds,
		Guardrails: guard,
		Exploration: ExplorationConfig{
			Enabled:       a.Exploration.Enabled,
			MaxStep:       a.Exploration.MaximumStep,
			CanaryPercent: a.Exploration.CanaryPercent,
			Cooldown:      cd,
		},
		Clock: clock,
	}
	return cc, nil
}

// FairnessConfig builds the fairness configuration.
func (c *Config) FairnessConfig() (FairnessConfig, error) {
	st, err := parseDurOpt(c.Fairness.StarvationThreshold)
	if err != nil {
		return FairnessConfig{}, err
	}
	fc := FairnessConfig{
		Algorithm:           FairnessAlgorithm(orDefault(c.Fairness.Algorithm, string(FairWeighted))),
		StarvationThreshold: st,
	}
	for _, cl := range c.Fairness.Classes {
		fc.Classes = append(fc.Classes, ClassPolicy{
			Class: cl.Class, Weight: cl.Weight, Priority: cl.Priority, ReservedShare: cl.ReservedShare,
		})
	}
	return fc, nil
}

// OverloadConfig builds the overload configuration.
func (c *Config) OverloadConfig() OverloadConfig {
	return OverloadConfig{
		QueueHighWatermark:     c.Overload.QueueHighWatermark,
		QueueCriticalWatermark: c.Overload.QueueCriticalWatermark,
		Policy:                 AdmissionPolicy(c.Overload.Policy),
	}
}

// --- parsing helpers ---

func parseDurRange(s string) (time.Duration, bool, error) {
	if s == "" {
		return 0, false, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, false, fmt.Errorf("adaptive: invalid duration %q: %w", s, err)
	}
	if d < 0 {
		return 0, false, fmt.Errorf("adaptive: duration %q must not be negative", s)
	}
	return d, true, nil
}

func parseDurOpt(s string) (time.Duration, error) {
	d, _, err := parseDurRange(s)
	return d, err
}

func parseIntRange(s string) (int, bool, error) {
	if s == "" {
		return 0, false, nil
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return 0, false, fmt.Errorf("adaptive: invalid integer %q: %w", s, err)
	}
	if v < 0 {
		return 0, false, fmt.Errorf("adaptive: value %q must not be negative", s)
	}
	return v, true, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func knownCollectionMode(m string) bool {
	switch CollectionMode(m) {
	case CollectOff, CollectCounters, CollectHistograms, CollectSampledEvents, CollectFullLocalDebug:
		return true
	default:
		return false
	}
}

func knownFairnessAlgo(a string) bool {
	switch FairnessAlgorithm(a) {
	case FairWeighted, FairDeficitRoundRobin:
		return true
	default:
		return false
	}
}

func knownAdmissionPolicy(p string) bool {
	switch AdmissionPolicy(p) {
	case AdmitAccept, AdmitBlock, AdmitReject, AdmitFallbackDirect, AdmitShedLowPriority, AdmitFlushEarly:
		return true
	default:
		return false
	}
}
