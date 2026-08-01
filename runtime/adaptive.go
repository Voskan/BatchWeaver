package runtime

import "time"

// AdaptiveSettings are bounded scheduler overrides an adaptive controller may
// apply to a bound operation at runtime. The runtime treats the binding's
// configured limits as authoritative hard bounds and clamps every field into
// them, so applying adaptive settings can only tighten scheduling (shorter wait,
// smaller batches, fewer concurrent calls), never exceed a configured bound.
//
// A zero field means "leave the configured default in effect" for that
// dimension.
type AdaptiveSettings struct {
	// MaxWait overrides the batching wait. It is clamped to [0, configured
	// MaxWait]. A zero value leaves the configured wait unchanged.
	MaxWait time.Duration
	// MaxBatchSize overrides the maximum batch size. It is clamped to [1,
	// configured MaxBatchSize].
	MaxBatchSize int
	// MaxConcurrency overrides the maximum concurrent provider calls. It is
	// clamped to [1, configured MaxConcurrency].
	MaxConcurrency int
}

// dynamicSettings is the immutable, already-clamped snapshot the coordinator
// reads on its own goroutine.
type dynamicSettings struct {
	maxWait        time.Duration
	maxBatchSize   int
	maxConcurrency int
}

// ApplyAdaptiveSettings installs bounded adaptive settings for this operation.
// The change is applied atomically: batches already in flight keep the settings
// they were dispatched under, and the next scheduling decision observes the new
// snapshot. Fields are clamped to the binding's hard configuration limits.
func (o *BoundOperation[K, V]) ApplyAdaptiveSettings(s AdaptiveSettings) {
	cfg := o.coord.cfg
	ds := &dynamicSettings{
		maxWait:        cfg.maxWait,
		maxBatchSize:   cfg.maxBatchSize,
		maxConcurrency: cfg.maxConcurrency,
	}
	switch {
	case s.MaxWait > 0 && s.MaxWait < cfg.maxWait:
		ds.maxWait = s.MaxWait
	case s.MaxWait == 0:
		ds.maxWait = cfg.maxWait
	default:
		ds.maxWait = clampDur(s.MaxWait, 0, cfg.maxWait)
	}
	if s.MaxBatchSize > 0 {
		ds.maxBatchSize = clampInt(s.MaxBatchSize, 1, cfg.maxBatchSize)
	}
	if s.MaxConcurrency > 0 {
		ds.maxConcurrency = clampInt(s.MaxConcurrency, 1, cfg.maxConcurrency)
	}
	o.coord.dyn.Store(ds)
}

// ClearAdaptiveSettings removes any adaptive overrides, restoring the configured
// defaults. It is the runtime side of an emergency disable or freeze.
func (o *BoundOperation[K, V]) ClearAdaptiveSettings() {
	o.coord.dyn.Store(nil)
}

// EffectiveSettings returns the settings currently in effect for the operation,
// after clamping, for observability.
func (o *BoundOperation[K, V]) EffectiveSettings() AdaptiveSettings {
	return AdaptiveSettings{
		MaxWait:        o.coord.effectiveMaxWait(),
		MaxBatchSize:   o.coord.effectiveMaxBatchSize(),
		MaxConcurrency: o.coord.effectiveMaxConcurrency(),
	}
}

// effectiveMaxWait returns the batching wait in effect, honoring any adaptive
// override but never exceeding the configured wait.
func (c *coordinator[K, V]) effectiveMaxWait() time.Duration {
	if ds := c.dyn.Load(); ds != nil {
		return ds.maxWait
	}
	return c.cfg.maxWait
}

// effectiveMaxBatchSize returns the maximum batch size in effect.
func (c *coordinator[K, V]) effectiveMaxBatchSize() int {
	if ds := c.dyn.Load(); ds != nil {
		return ds.maxBatchSize
	}
	return c.cfg.maxBatchSize
}

// effectiveMaxConcurrency returns the maximum concurrency in effect.
func (c *coordinator[K, V]) effectiveMaxConcurrency() int {
	if ds := c.dyn.Load(); ds != nil {
		return ds.maxConcurrency
	}
	return c.cfg.maxConcurrency
}

// clampInt clamps v to [lo, hi]; hi<=0 means no upper bound.
func clampInt(v, lo, hi int) int {
	if v < lo {
		v = lo
	}
	if hi > 0 && v > hi {
		v = hi
	}
	return v
}

// clampDur clamps a duration to [lo, hi]; hi<=0 means no upper bound.
func clampDur(v, lo, hi time.Duration) time.Duration {
	if v < lo {
		v = lo
	}
	if hi > 0 && v > hi {
		v = hi
	}
	return v
}
