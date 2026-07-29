package operation

import (
	"errors"
	"fmt"
)

// DeduplicationMode selects how duplicate logical requests are collapsed.
type DeduplicationMode uint8

const (
	// DeduplicationDisabled performs no deduplication.
	DeduplicationDisabled DeduplicationMode = iota
	// DeduplicationExact deduplicates requests with identical keys.
	DeduplicationExact
	// DeduplicationCanonical deduplicates requests whose canonical form matches,
	// using a canonicalizer symbol provided by the operation.
	DeduplicationCanonical
)

// ErrInvalidDeduplicationMode is returned for an unknown deduplication mode.
var ErrInvalidDeduplicationMode = errors.New("invalid deduplication mode")

var deduplicationModeNames = []string{"disabled", "exact", "canonical"}

// String returns the canonical name of the deduplication mode.
func (m DeduplicationMode) String() string { return enumString(m, deduplicationModeNames) }

// Valid reports whether m is a defined deduplication mode.
func (m DeduplicationMode) Valid() bool { return int(m) < len(deduplicationModeNames) }

// ParseDeduplicationMode resolves the canonical name of a deduplication mode.
func ParseDeduplicationMode(s string) (DeduplicationMode, error) {
	return parseEnum[DeduplicationMode](s, deduplicationModeNames, ErrInvalidDeduplicationMode)
}

// MarshalText implements encoding.TextMarshaler.
func (m DeduplicationMode) MarshalText() ([]byte, error) {
	return marshalEnum(m, deduplicationModeNames, ErrInvalidDeduplicationMode)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *DeduplicationMode) UnmarshalText(data []byte) error {
	v, err := ParseDeduplicationMode(string(data))
	if err != nil {
		return err
	}
	*m = v
	return nil
}

// ErrInvalidDeduplicationPolicy is returned when a DeduplicationPolicy fails
// validation.
var ErrInvalidDeduplicationPolicy = errors.New("invalid deduplication policy")

// DeduplicationParams is the input used to construct a DeduplicationPolicy.
type DeduplicationParams struct {
	Mode                DeduplicationMode
	InFlight            bool
	ScopeMemoization    bool
	NegativeMemoization bool
	ErrorMemoization    bool
	MaxItems            int
	MaxBytes            int64
	Canonicalizer       Symbol
}

// DeduplicationPolicy is a validated, immutable-by-convention deduplication
// configuration. No deduplication is executed in this release.
type DeduplicationPolicy struct {
	params DeduplicationParams
}

// DefaultDeduplicationPolicy returns a disabled deduplication policy.
func DefaultDeduplicationPolicy() DeduplicationPolicy {
	return DeduplicationPolicy{params: DeduplicationParams{Mode: DeduplicationDisabled}}
}

// NewDeduplicationPolicy validates params and returns a DeduplicationPolicy.
func NewDeduplicationPolicy(params DeduplicationParams) (DeduplicationPolicy, error) {
	p := DeduplicationPolicy{params: params}
	if err := p.Validate(); err != nil {
		return DeduplicationPolicy{}, err
	}
	return p, nil
}

// Mode returns the deduplication mode.
func (p DeduplicationPolicy) Mode() DeduplicationMode { return p.params.Mode }

// InFlight reports whether in-flight deduplication is enabled.
func (p DeduplicationPolicy) InFlight() bool { return p.params.InFlight }

// ScopeMemoization reports whether scope memoization is enabled.
func (p DeduplicationPolicy) ScopeMemoization() bool { return p.params.ScopeMemoization }

// NegativeMemoization reports whether negative results are memoized.
func (p DeduplicationPolicy) NegativeMemoization() bool { return p.params.NegativeMemoization }

// ErrorMemoization reports whether deterministic errors are memoized.
func (p DeduplicationPolicy) ErrorMemoization() bool { return p.params.ErrorMemoization }

// MaxItems returns the maximum number of memoized items.
func (p DeduplicationPolicy) MaxItems() int { return p.params.MaxItems }

// MaxBytes returns the maximum memoized bytes.
func (p DeduplicationPolicy) MaxBytes() int64 { return p.params.MaxBytes }

// Canonicalizer returns the canonicalizer symbol (valid only in canonical mode).
func (p DeduplicationPolicy) Canonicalizer() Symbol { return p.params.Canonicalizer }

// Enabled reports whether any deduplication is configured.
func (p DeduplicationPolicy) Enabled() bool { return p.params.Mode != DeduplicationDisabled }

// Validate reports whether the deduplication policy is internally consistent.
// Bounded memoization is required whenever deduplication is enabled; unbounded
// memoization is rejected.
func (p DeduplicationPolicy) Validate() error {
	q := p.params
	if !q.Mode.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidDeduplicationPolicy, ErrInvalidDeduplicationMode)
	}
	if q.Mode == DeduplicationDisabled {
		return nil
	}
	if q.Mode == DeduplicationCanonical && q.Canonicalizer.IsZero() {
		return fmt.Errorf("%w: canonical mode requires a canonicalizer symbol", ErrInvalidDeduplicationPolicy)
	}
	if !q.Canonicalizer.IsZero() {
		if err := q.Canonicalizer.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidDeduplicationPolicy, err)
		}
	}
	if q.ScopeMemoization {
		if q.MaxItems < 1 {
			return fmt.Errorf("%w: memoization requires a positive item limit", ErrInvalidDeduplicationPolicy)
		}
		if q.MaxBytes < 1 {
			return fmt.Errorf("%w: memoization requires a positive byte limit", ErrInvalidDeduplicationPolicy)
		}
	}
	return nil
}
