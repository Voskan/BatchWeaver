package adaptive

import (
	"hash/fnv"
	"math"
)

// SamplingStrategy selects how the collector decides which events to record in
// modes that store individual samples.
type SamplingStrategy string

const (
	// SampleAll records every event (rate 1.0).
	SampleAll SamplingStrategy = "all"
	// SampleFixedProbability records events at a fixed probability using a
	// deterministic hash of the event key, so the decision is reproducible.
	SampleFixedProbability SamplingStrategy = "fixed-probability"
	// SampleTailBiased always records error and timeout events and applies the
	// fixed probability to the rest, so rare failures are never lost.
	SampleTailBiased SamplingStrategy = "tail-biased"
)

// Sampler makes deterministic, reproducible sampling decisions. It holds no
// mutable state, so it is safe for concurrent use.
type Sampler struct {
	strategy SamplingStrategy
	rate     float64
	// threshold is the inclusive upper bound on the normalized hash for a sample
	// to be kept.
	threshold uint64
}

// NewSampler returns a Sampler. A rate outside [0,1] is clamped. For SampleAll
// the rate is forced to 1.
func NewSampler(strategy SamplingStrategy, rate float64) *Sampler {
	switch strategy {
	case SampleAll:
		rate = 1
	case SampleFixedProbability, SampleTailBiased:
		if rate < 0 {
			rate = 0
		}
		if rate > 1 {
			rate = 1
		}
	default:
		strategy = SampleAll
		rate = 1
	}
	return &Sampler{
		strategy:  strategy,
		rate:      rate,
		threshold: uint64(rate * float64(math.MaxUint64)),
	}
}

// Sample reports whether an event identified by key should be recorded. isTail
// marks error/timeout events that tail-biased sampling must always keep.
func (s *Sampler) Sample(key uint64, isTail bool) bool {
	if s.strategy == SampleAll || s.rate >= 1 {
		return true
	}
	if s.rate <= 0 {
		return isTail && s.strategy == SampleTailBiased
	}
	if s.strategy == SampleTailBiased && isTail {
		return true
	}
	return mixHash(key) <= s.threshold
}

// ScaleFactor returns the multiplier that converts a sampled count back to an
// estimated population count.
func (s *Sampler) ScaleFactor() float64 {
	if s.rate <= 0 {
		return 0
	}
	return 1 / s.rate
}

// Metadata returns the sampling metadata for the collected profile.
func (s *Sampler) Metadata(dropped uint64) SamplingMetadata {
	return SamplingMetadata{
		Strategy:    string(s.strategy),
		Rate:        s.rate,
		TailBiased:  s.strategy == SampleTailBiased,
		ScaleFactor: s.ScaleFactor(),
		Dropped:     dropped,
	}
}

// HashKey returns a stable hash for a string sampling key.
func HashKey(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// mixHash spreads a key across the 64-bit space so sequential keys sample
// independently (splitmix64 finalizer).
func mixHash(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}
