package adaptive

import "time"

// Settings are the tunable scheduler parameters the controller may recommend or
// apply. Durations are stored as nanoseconds so the type serializes and compares
// deterministically. A zero MaxWaitNanos means no batching wait.
type Settings struct {
	MaxWaitNanos   int64         `json:"max_wait_nanos"`
	MaxBatchSize   int           `json:"max_batch_size"`
	MaxBatchWeight int64         `json:"max_batch_weight"`
	MaxConcurrency int           `json:"max_concurrency"`
	ChunkSize      int           `json:"chunk_size"`
	Mode           ExecutionMode `json:"mode"`
}

// MaxWait returns the batching wait as a Duration.
func (s Settings) MaxWait() time.Duration { return time.Duration(s.MaxWaitNanos) }

// HardBounds are authoritative limits the adaptive controller can never exceed.
// They are supplied by configuration and derived from operation contracts and
// adapter limits. A zero Max means "not bounded above by adaptive logic" only
// when explicitly documented; the controller treats a zero Max on size or
// concurrency as 1 to stay conservative.
type HardBounds struct {
	MinWaitNanos   int64 `json:"min_wait_nanos"`
	MaxWaitNanos   int64 `json:"max_wait_nanos"`
	MinBatchSize   int   `json:"min_batch_size"`
	MaxBatchSize   int   `json:"max_batch_size"`
	MinConcurrency int   `json:"min_concurrency"`
	MaxConcurrency int   `json:"max_concurrency"`
	MaxBatchWeight int64 `json:"max_batch_weight"`
	MaxChunkSize   int   `json:"max_chunk_size"`
}

// DefaultBounds returns conservative bounds used when configuration omits them.
func DefaultBounds() HardBounds {
	return HardBounds{
		MinWaitNanos:   0,
		MaxWaitNanos:   int64(2 * time.Millisecond),
		MinBatchSize:   1,
		MaxBatchSize:   512,
		MinConcurrency: 1,
		MaxConcurrency: 16,
		MaxChunkSize:   512,
	}
}

// Clamp returns s constrained to the bounds, plus a diagnostic for every
// dimension that had to be clamped. Hard bounds are authoritative: a
// recommendation that exceeds them is silently impossible, and the diagnostic
// records that it was reduced.
func (h HardBounds) Clamp(s Settings, operation string) (Settings, []Diagnostic) {
	var diags []Diagnostic
	clampInt := func(name string, v, lo, hi int) int {
		if hi > 0 && v > hi {
			diags = append(diags, newDiag(CodeRecommendationExceedsBound, "info", operation, name+" reduced to hard maximum"))
			return hi
		}
		if v < lo {
			return lo
		}
		return v
	}
	clampInt64 := func(name string, v, lo, hi int64) int64 {
		if hi > 0 && v > hi {
			diags = append(diags, newDiag(CodeRecommendationExceedsBound, "info", operation, name+" reduced to hard maximum"))
			return hi
		}
		if v < lo {
			return lo
		}
		return v
	}
	minBatch := h.MinBatchSize
	if minBatch < 1 {
		minBatch = 1
	}
	minConc := h.MinConcurrency
	if minConc < 1 {
		minConc = 1
	}
	s.MaxWaitNanos = clampInt64("max_wait", s.MaxWaitNanos, h.MinWaitNanos, h.MaxWaitNanos)
	s.MaxBatchSize = clampInt("max_batch_size", s.MaxBatchSize, minBatch, h.MaxBatchSize)
	s.MaxConcurrency = clampInt("concurrency", s.MaxConcurrency, minConc, h.MaxConcurrency)
	if h.MaxBatchWeight > 0 {
		s.MaxBatchWeight = clampInt64("max_batch_weight", s.MaxBatchWeight, 0, h.MaxBatchWeight)
	}
	if h.MaxChunkSize > 0 && s.ChunkSize > 0 {
		s.ChunkSize = clampInt("chunk_size", s.ChunkSize, 1, h.MaxChunkSize)
	}
	return s, diags
}

// Validate reports whether the bounds are internally consistent.
func (h HardBounds) Validate() error {
	if h.MaxWaitNanos < 0 || h.MinWaitNanos < 0 || h.MinWaitNanos > h.MaxWaitNanos && h.MaxWaitNanos > 0 {
		return errBounds("max_wait")
	}
	if h.MaxBatchSize > 0 && h.MinBatchSize > h.MaxBatchSize {
		return errBounds("max_batch_size")
	}
	if h.MaxConcurrency > 0 && h.MinConcurrency > h.MaxConcurrency {
		return errBounds("concurrency")
	}
	return nil
}

// SLOGuardrails constrain adaptive changes so a recommendation that risks the
// service objective is rejected or rolled back. Hard guardrails block a change;
// soft guardrails only lower confidence.
type SLOGuardrails struct {
	P95QueueDelayNanos  int64         `json:"p95_queue_delay_nanos"`
	TimeoutRate         float64       `json:"timeout_rate"`
	ErrorRateRegression float64       `json:"error_rate_regression"`
	RollbackWindow      time.Duration `json:"rollback_window"`
	// AddedLatencyBudgetNanos bounds the modeled added p95 latency of a change.
	AddedLatencyBudgetNanos int64 `json:"added_latency_budget_nanos"`
}

// errBounds returns a bounds validation error.
func errBounds(field string) error {
	return &BoundsError{Field: field}
}

// BoundsError reports an invalid hard-bounds configuration.
type BoundsError struct{ Field string }

func (e *BoundsError) Error() string {
	return "adaptive: invalid hard bounds for " + e.Field
}
