package operation

import (
	"errors"
	"fmt"
)

// FallbackMode selects how work is handled when batching is unavailable. No
// fallback execution is implemented in this release.
type FallbackMode uint8

const (
	// FallbackScalar runs the scalar implementation for each item.
	FallbackScalar FallbackMode = iota
	// FallbackBatchOfOne submits single-item batches.
	FallbackBatchOfOne
	// FallbackParallelScalar runs scalar calls with bounded concurrency.
	FallbackParallelScalar
	// FallbackReject rejects work rather than falling back.
	FallbackReject
)

// ErrInvalidFallbackMode is returned for an unknown fallback mode.
var ErrInvalidFallbackMode = errors.New("invalid fallback mode")

var fallbackModeNames = []string{"scalar", "batch-of-one", "parallel-scalar", "reject"}

// String returns the canonical name of the fallback mode.
func (m FallbackMode) String() string { return enumString(m, fallbackModeNames) }

// Valid reports whether m is a defined fallback mode.
func (m FallbackMode) Valid() bool { return int(m) < len(fallbackModeNames) }

// ParseFallbackMode resolves the canonical name of a fallback mode.
func ParseFallbackMode(s string) (FallbackMode, error) {
	return parseEnum[FallbackMode](s, fallbackModeNames, ErrInvalidFallbackMode)
}

// MarshalText implements encoding.TextMarshaler.
func (m FallbackMode) MarshalText() ([]byte, error) {
	return marshalEnum(m, fallbackModeNames, ErrInvalidFallbackMode)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *FallbackMode) UnmarshalText(data []byte) error {
	v, err := ParseFallbackMode(string(data))
	if err != nil {
		return err
	}
	*m = v
	return nil
}

// ErrInvalidFallbackPolicy is returned when a FallbackPolicy fails validation.
var ErrInvalidFallbackPolicy = errors.New("invalid fallback policy")

// FallbackParams is the input used to construct a FallbackPolicy.
type FallbackParams struct {
	Mode                   FallbackMode
	MaxScalarConcurrency   int
	OnQueueOverflow        bool
	OnProviderUnavailable  bool
	OnUnsupportedPartition bool
}

// FallbackPolicy is a validated, immutable-by-convention fallback contract.
type FallbackPolicy struct {
	params FallbackParams
}

// DefaultFallbackPolicy returns a conservative scalar fallback policy.
func DefaultFallbackPolicy() FallbackPolicy {
	return FallbackPolicy{params: FallbackParams{Mode: FallbackScalar}}
}

// NewFallbackPolicy validates params and returns a FallbackPolicy.
func NewFallbackPolicy(params FallbackParams) (FallbackPolicy, error) {
	p := FallbackPolicy{params: params}
	if err := p.Validate(); err != nil {
		return FallbackPolicy{}, err
	}
	return p, nil
}

// Mode returns the fallback mode.
func (p FallbackPolicy) Mode() FallbackMode { return p.params.Mode }

// MaxScalarConcurrency returns the concurrency limit for parallel scalar fallback.
func (p FallbackPolicy) MaxScalarConcurrency() int { return p.params.MaxScalarConcurrency }

// OnQueueOverflow reports whether queue overflow may trigger fallback.
func (p FallbackPolicy) OnQueueOverflow() bool { return p.params.OnQueueOverflow }

// OnProviderUnavailable reports whether provider unavailability may trigger fallback.
func (p FallbackPolicy) OnProviderUnavailable() bool { return p.params.OnProviderUnavailable }

// OnUnsupportedPartition reports whether an unsupported partition may trigger fallback.
func (p FallbackPolicy) OnUnsupportedPartition() bool { return p.params.OnUnsupportedPartition }

// Validate reports whether the fallback policy is internally consistent.
func (p FallbackPolicy) Validate() error {
	q := p.params
	if !q.Mode.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidFallbackPolicy, ErrInvalidFallbackMode)
	}
	switch q.Mode {
	case FallbackParallelScalar:
		if q.MaxScalarConcurrency < 1 {
			return fmt.Errorf("%w: parallel scalar fallback requires positive concurrency", ErrInvalidFallbackPolicy)
		}
	case FallbackReject:
		if q.MaxScalarConcurrency != 0 {
			return fmt.Errorf("%w: reject fallback must not set scalar concurrency", ErrInvalidFallbackPolicy)
		}
	default:
		if q.MaxScalarConcurrency < 0 {
			return fmt.Errorf("%w: scalar concurrency must not be negative", ErrInvalidFallbackPolicy)
		}
	}
	return nil
}
