package operation

import (
	"errors"
	"fmt"
	"time"
)

// SchedulerMode selects how a future runtime forms batch waves. This package
// only stores and validates the choice; no scheduler is implemented here.
type SchedulerMode uint8

const (
	// SchedulerImmediateWave dispatches a wave as soon as work is available.
	SchedulerImmediateWave SchedulerMode = iota
	// SchedulerFixedWindow waits a fixed window before dispatching.
	SchedulerFixedWindow
	// SchedulerAdaptive adjusts batching to observed load.
	SchedulerAdaptive
	// SchedulerManual dispatches only when explicitly flushed.
	SchedulerManual
	// SchedulerThroughput optimizes for throughput.
	SchedulerThroughput
	// SchedulerLatency optimizes for latency.
	SchedulerLatency
	// SchedulerDeadlineAware dispatches with respect to item deadlines.
	SchedulerDeadlineAware
)

// ErrInvalidSchedulerMode is returned for an unknown scheduler mode.
var ErrInvalidSchedulerMode = errors.New("invalid scheduler mode")

var schedulerModeNames = []string{
	"immediate-wave", "fixed-window", "adaptive", "manual",
	"throughput", "latency", "deadline-aware",
}

// String returns the canonical name of the scheduler mode.
func (m SchedulerMode) String() string { return enumString(m, schedulerModeNames) }

// Valid reports whether m is a defined scheduler mode.
func (m SchedulerMode) Valid() bool { return int(m) < len(schedulerModeNames) }

// ParseSchedulerMode resolves the canonical name of a scheduler mode.
func ParseSchedulerMode(s string) (SchedulerMode, error) {
	return parseEnum[SchedulerMode](s, schedulerModeNames, ErrInvalidSchedulerMode)
}

// MarshalText implements encoding.TextMarshaler.
func (m SchedulerMode) MarshalText() ([]byte, error) {
	return marshalEnum(m, schedulerModeNames, ErrInvalidSchedulerMode)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *SchedulerMode) UnmarshalText(data []byte) error {
	v, err := ParseSchedulerMode(string(data))
	if err != nil {
		return err
	}
	*m = v
	return nil
}

// FairnessMode selects how a future scheduler orders competing partitions.
type FairnessMode uint8

const (
	// FairnessFIFO serves partitions in arrival order.
	FairnessFIFO FairnessMode = iota
	// FairnessWeighted serves partitions by weight.
	FairnessWeighted
)

var fairnessNames = []string{"fifo", "weighted"}

// String returns the canonical name of the fairness mode.
func (f FairnessMode) String() string { return enumString(f, fairnessNames) }

// Valid reports whether f is a defined fairness mode.
func (f FairnessMode) Valid() bool { return int(f) < len(fairnessNames) }

// OverflowBehavior selects what happens when a queue limit is reached.
type OverflowBehavior uint8

const (
	// OverflowBlock blocks until capacity is available.
	OverflowBlock OverflowBehavior = iota
	// OverflowReject rejects new work.
	OverflowReject
	// OverflowFallback diverts new work to the fallback path.
	OverflowFallback
)

var overflowNames = []string{"block", "reject", "fallback"}

// String returns the canonical name of the overflow behavior.
func (o OverflowBehavior) String() string { return enumString(o, overflowNames) }

// Valid reports whether o is a defined overflow behavior.
func (o OverflowBehavior) Valid() bool { return int(o) < len(overflowNames) }

// ErrInvalidSchedulerPolicy is returned when a SchedulerPolicy fails validation.
var ErrInvalidSchedulerPolicy = errors.New("invalid scheduler policy")

// SchedulerParams is the input used to construct a SchedulerPolicy. It exists so
// that callers (including configuration normalization) can build a policy from
// named fields without a long positional constructor.
type SchedulerParams struct {
	Mode             SchedulerMode
	MinBatchSize     int
	MaxBatchSize     int
	MaxBatchWeight   int
	MaxPayloadBytes  int64
	MaxWait          time.Duration
	DeadlineMargin   time.Duration
	MaxConcurrency   int
	QueueItems       int
	QueueBytes       int64
	ActivePartitions int
	WaitersPerKey    int
	Fairness         FairnessMode
	PrioritySupport  bool
	Overflow         OverflowBehavior
}

// SchedulerPolicy is a validated, immutable-by-convention scheduler
// configuration. No scheduling behavior is implemented in this release.
type SchedulerPolicy struct {
	params SchedulerParams
}

// DefaultSchedulerPolicy returns a conservative, valid immediate-wave policy.
func DefaultSchedulerPolicy() SchedulerPolicy {
	return SchedulerPolicy{params: SchedulerParams{
		Mode:             SchedulerImmediateWave,
		MinBatchSize:     1,
		MaxBatchSize:     256,
		MaxBatchWeight:   4096,
		MaxPayloadBytes:  4 << 20, // 4 MiB
		MaxWait:          0,
		DeadlineMargin:   0,
		MaxConcurrency:   1,
		QueueItems:       100000,
		QueueBytes:       64 << 20, // 64 MiB
		ActivePartitions: 1024,
		WaitersPerKey:    1024,
		Fairness:         FairnessFIFO,
		Overflow:         OverflowBlock,
	}}
}

// NewSchedulerPolicy validates params and returns a SchedulerPolicy.
func NewSchedulerPolicy(params SchedulerParams) (SchedulerPolicy, error) {
	p := SchedulerPolicy{params: params}
	if err := p.Validate(); err != nil {
		return SchedulerPolicy{}, err
	}
	return p, nil
}

// Mode returns the scheduler mode.
func (p SchedulerPolicy) Mode() SchedulerMode { return p.params.Mode }

// MinBatchSize returns the minimum batch size.
func (p SchedulerPolicy) MinBatchSize() int { return p.params.MinBatchSize }

// MaxBatchSize returns the maximum batch size.
func (p SchedulerPolicy) MaxBatchSize() int { return p.params.MaxBatchSize }

// MaxBatchWeight returns the maximum aggregate item weight per batch.
func (p SchedulerPolicy) MaxBatchWeight() int { return p.params.MaxBatchWeight }

// MaxPayloadBytes returns the maximum batch payload in bytes.
func (p SchedulerPolicy) MaxPayloadBytes() int64 { return p.params.MaxPayloadBytes }

// MaxWait returns the maximum wait before a wave is dispatched.
func (p SchedulerPolicy) MaxWait() time.Duration { return p.params.MaxWait }

// DeadlineMargin returns the safety margin subtracted from item deadlines.
func (p SchedulerPolicy) DeadlineMargin() time.Duration { return p.params.DeadlineMargin }

// MaxConcurrency returns the maximum concurrent provider calls.
func (p SchedulerPolicy) MaxConcurrency() int { return p.params.MaxConcurrency }

// QueueItems returns the queue item limit.
func (p SchedulerPolicy) QueueItems() int { return p.params.QueueItems }

// QueueBytes returns the queue byte limit.
func (p SchedulerPolicy) QueueBytes() int64 { return p.params.QueueBytes }

// ActivePartitions returns the active-partition limit.
func (p SchedulerPolicy) ActivePartitions() int { return p.params.ActivePartitions }

// WaitersPerKey returns the per-key waiter limit.
func (p SchedulerPolicy) WaitersPerKey() int { return p.params.WaitersPerKey }

// Fairness returns the fairness mode.
func (p SchedulerPolicy) Fairness() FairnessMode { return p.params.Fairness }

// PrioritySupport reports whether item priority is honored.
func (p SchedulerPolicy) PrioritySupport() bool { return p.params.PrioritySupport }

// Overflow returns the overflow behavior.
func (p SchedulerPolicy) Overflow() OverflowBehavior { return p.params.Overflow }

// Validate reports whether the scheduler policy is internally consistent and
// free of unsafe or unbounded settings.
func (p SchedulerPolicy) Validate() error {
	q := p.params
	if !q.Mode.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidSchedulerPolicy, ErrInvalidSchedulerMode)
	}
	if !q.Fairness.Valid() {
		return fmt.Errorf("%w: invalid fairness mode", ErrInvalidSchedulerPolicy)
	}
	if !q.Overflow.Valid() {
		return fmt.Errorf("%w: invalid overflow behavior", ErrInvalidSchedulerPolicy)
	}
	if q.MinBatchSize < 1 {
		return fmt.Errorf("%w: minimum batch size must be positive", ErrInvalidSchedulerPolicy)
	}
	if q.MaxBatchSize < 1 {
		return fmt.Errorf("%w: maximum batch size must be positive", ErrInvalidSchedulerPolicy)
	}
	if q.MinBatchSize > q.MaxBatchSize {
		return fmt.Errorf("%w: minimum batch size exceeds maximum", ErrInvalidSchedulerPolicy)
	}
	if q.MaxBatchWeight < 1 {
		return fmt.Errorf("%w: maximum batch weight must be positive", ErrInvalidSchedulerPolicy)
	}
	if q.MaxPayloadBytes < 1 {
		return fmt.Errorf("%w: maximum payload bytes must be positive", ErrInvalidSchedulerPolicy)
	}
	if q.MaxWait < 0 {
		return fmt.Errorf("%w: maximum wait must not be negative", ErrInvalidSchedulerPolicy)
	}
	if q.DeadlineMargin < 0 {
		return fmt.Errorf("%w: deadline margin must not be negative", ErrInvalidSchedulerPolicy)
	}
	if q.MaxConcurrency < 1 {
		return fmt.Errorf("%w: maximum concurrency must be positive", ErrInvalidSchedulerPolicy)
	}
	if q.QueueItems < 1 {
		return fmt.Errorf("%w: queue item limit must be positive", ErrInvalidSchedulerPolicy)
	}
	if q.QueueBytes < 1 {
		return fmt.Errorf("%w: queue byte limit must be positive", ErrInvalidSchedulerPolicy)
	}
	if q.ActivePartitions < 1 {
		return fmt.Errorf("%w: active-partition limit must be positive", ErrInvalidSchedulerPolicy)
	}
	if q.WaitersPerKey < 1 {
		return fmt.Errorf("%w: waiters-per-key limit must be positive", ErrInvalidSchedulerPolicy)
	}
	if q.Mode == SchedulerManual && q.MaxWait > 0 {
		return fmt.Errorf("%w: manual mode must not configure an automatic wait", ErrInvalidSchedulerPolicy)
	}
	if q.Mode == SchedulerAdaptive && q.MaxWait <= 0 {
		return fmt.Errorf("%w: adaptive mode requires a positive maximum wait", ErrInvalidSchedulerPolicy)
	}
	return nil
}
