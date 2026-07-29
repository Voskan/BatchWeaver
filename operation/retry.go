package operation

import (
	"errors"
	"fmt"
	"time"
)

// JitterMode selects how retry backoff is randomized. No retry execution is
// implemented in this release.
type JitterMode uint8

const (
	// JitterNone applies no randomization.
	JitterNone JitterMode = iota
	// JitterFull randomizes across the full backoff interval.
	JitterFull
	// JitterEqual randomizes across half the backoff interval.
	JitterEqual
)

var jitterNames = []string{"none", "full", "equal"}

// String returns the canonical name of the jitter mode.
func (j JitterMode) String() string { return enumString(j, jitterNames) }

// Valid reports whether j is a defined jitter mode.
func (j JitterMode) Valid() bool { return int(j) < len(jitterNames) }

// RetryClassification names a class of failure that may be retried.
type RetryClassification uint8

const (
	// RetryTransport covers transport-level failures.
	RetryTransport RetryClassification = iota
	// RetryThrottled covers throttling responses.
	RetryThrottled
	// RetryTimeout covers timeouts.
	RetryTimeout
	// RetryUnavailable covers temporary unavailability.
	RetryUnavailable
	// RetryConflict covers retryable conflicts.
	RetryConflict
)

// ErrInvalidRetryClassification is returned for an unknown classification.
var ErrInvalidRetryClassification = errors.New("invalid retry classification")

var retryClassificationNames = []string{"transport", "throttled", "timeout", "unavailable", "conflict"}

// String returns the canonical name of the retry classification.
func (c RetryClassification) String() string { return enumString(c, retryClassificationNames) }

// Valid reports whether c is a defined classification.
func (c RetryClassification) Valid() bool { return int(c) < len(retryClassificationNames) }

// ParseRetryClassification resolves the canonical name of a classification.
func ParseRetryClassification(s string) (RetryClassification, error) {
	return parseEnum[RetryClassification](s, retryClassificationNames, ErrInvalidRetryClassification)
}

// MarshalText implements encoding.TextMarshaler.
func (c RetryClassification) MarshalText() ([]byte, error) {
	return marshalEnum(c, retryClassificationNames, ErrInvalidRetryClassification)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (c *RetryClassification) UnmarshalText(data []byte) error {
	v, err := ParseRetryClassification(string(data))
	if err != nil {
		return err
	}
	*c = v
	return nil
}

// UnknownOutcomeBehavior selects what happens on an unknown outcome.
type UnknownOutcomeBehavior uint8

const (
	// UnknownOutcomeNoRetry does not retry on an unknown outcome; the default.
	UnknownOutcomeNoRetry UnknownOutcomeBehavior = iota
	// UnknownOutcomeRetry retries on an unknown outcome (only safe when
	// idempotent).
	UnknownOutcomeRetry
)

var unknownOutcomeNames = []string{"no-retry", "retry"}

// String returns the canonical name of the unknown-outcome behavior.
func (u UnknownOutcomeBehavior) String() string { return enumString(u, unknownOutcomeNames) }

// Valid reports whether u is a defined unknown-outcome behavior.
func (u UnknownOutcomeBehavior) Valid() bool { return int(u) < len(unknownOutcomeNames) }

// ErrInvalidRetryPolicy is returned when a RetryPolicy fails validation.
var ErrInvalidRetryPolicy = errors.New("invalid retry policy")

// RetryParams is the input used to construct a RetryPolicy.
type RetryParams struct {
	Enabled           bool
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	Jitter            JitterMode
	Retryable         []RetryClassification
	RespectRetryAfter bool
	PartialItemRetry  bool
	UnknownOutcome    UnknownOutcomeBehavior
	IdempotencyKey    Symbol
}

// RetryPolicy is a validated, immutable-by-convention retry contract. No retry
// execution is implemented in this release.
type RetryPolicy struct {
	params RetryParams
}

// DefaultRetryPolicy returns a disabled retry policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{params: RetryParams{Enabled: false}}
}

// NewRetryPolicy validates params and returns a RetryPolicy.
func NewRetryPolicy(params RetryParams) (RetryPolicy, error) {
	p := RetryPolicy{params: append0(params)}
	if err := p.Validate(); err != nil {
		return RetryPolicy{}, err
	}
	return p, nil
}

// append0 defensively copies the classification slice so the policy does not
// alias caller-owned storage.
func append0(params RetryParams) RetryParams {
	params.Retryable = append([]RetryClassification(nil), params.Retryable...)
	return params
}

// Enabled reports whether retry is enabled.
func (p RetryPolicy) Enabled() bool { return p.params.Enabled }

// MaxAttempts returns the maximum number of attempts.
func (p RetryPolicy) MaxAttempts() int { return p.params.MaxAttempts }

// InitialBackoff returns the initial backoff duration.
func (p RetryPolicy) InitialBackoff() time.Duration { return p.params.InitialBackoff }

// MaxBackoff returns the maximum backoff duration.
func (p RetryPolicy) MaxBackoff() time.Duration { return p.params.MaxBackoff }

// Jitter returns the jitter mode.
func (p RetryPolicy) Jitter() JitterMode { return p.params.Jitter }

// Retryable returns a copy of the retryable classifications.
func (p RetryPolicy) Retryable() []RetryClassification {
	return append([]RetryClassification(nil), p.params.Retryable...)
}

// RespectRetryAfter reports whether Retry-After hints are honored.
func (p RetryPolicy) RespectRetryAfter() bool { return p.params.RespectRetryAfter }

// PartialItemRetry reports whether individual failed items may be retried.
func (p RetryPolicy) PartialItemRetry() bool { return p.params.PartialItemRetry }

// UnknownOutcome returns the unknown-outcome behavior.
func (p RetryPolicy) UnknownOutcome() UnknownOutcomeBehavior { return p.params.UnknownOutcome }

// IdempotencyKey returns the idempotency-key symbol, if any.
func (p RetryPolicy) IdempotencyKey() Symbol { return p.params.IdempotencyKey }

// Validate reports whether the retry policy is internally consistent.
func (p RetryPolicy) Validate() error {
	q := p.params
	if !q.Jitter.Valid() {
		return fmt.Errorf("%w: invalid jitter mode", ErrInvalidRetryPolicy)
	}
	if !q.UnknownOutcome.Valid() {
		return fmt.Errorf("%w: invalid unknown-outcome behavior", ErrInvalidRetryPolicy)
	}
	if q.InitialBackoff < 0 || q.MaxBackoff < 0 {
		return fmt.Errorf("%w: backoff durations must not be negative", ErrInvalidRetryPolicy)
	}
	for _, c := range q.Retryable {
		if !c.Valid() {
			return fmt.Errorf("%w: %w", ErrInvalidRetryPolicy, ErrInvalidRetryClassification)
		}
	}
	if !q.IdempotencyKey.IsZero() {
		if err := q.IdempotencyKey.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidRetryPolicy, err)
		}
	}
	if !q.Enabled {
		return nil
	}
	if q.MaxAttempts < 2 {
		return fmt.Errorf("%w: enabled retry requires at least two attempts", ErrInvalidRetryPolicy)
	}
	if q.MaxBackoff < q.InitialBackoff {
		return fmt.Errorf("%w: maximum backoff is lower than initial backoff", ErrInvalidRetryPolicy)
	}
	if len(q.Retryable) == 0 {
		return fmt.Errorf("%w: enabled retry requires at least one retryable classification", ErrInvalidRetryPolicy)
	}
	return nil
}
