package config

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidDuration is returned when a duration string cannot be parsed.
var ErrInvalidDuration = errors.New("invalid duration")

// Duration is a configuration duration using Go duration syntax. It is
// deliberately strict: it requires a unit (rejecting unit-ambiguous bare
// numbers) and renders canonically. Both "us" and "µs" are accepted on input;
// output uses Go's canonical form.
type Duration time.Duration

// ParseDuration parses a Go-style duration string such as "500us" or "1m30s".
// It rejects the empty string and unit-less numbers.
func ParseDuration(s string) (Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("%w: empty", ErrInvalidDuration)
	}
	// Reject a bare number with no unit, which time.ParseDuration would only
	// accept for the special case "0"; require an explicit unit for clarity.
	if isAllDigits(s) {
		return 0, fmt.Errorf("%w: %q has no unit", ErrInvalidDuration, s)
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("%w: %q: %w", ErrInvalidDuration, s, err)
	}
	return Duration(d), nil
}

// isAllDigits reports whether s consists only of ASCII digits (optionally with a
// leading sign), which would be a unit-less duration.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if s[0] == '+' || s[0] == '-' {
		i = 1
	}
	if i == len(s) {
		return false
	}
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// Std returns the value as a time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }

// String renders the duration canonically. Zero renders as "0s".
func (d Duration) String() string {
	if d == 0 {
		return "0s"
	}
	return time.Duration(d).String()
}

// MarshalText implements encoding.TextMarshaler.
func (d Duration) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Duration) UnmarshalText(data []byte) error {
	v, err := ParseDuration(string(data))
	if err != nil {
		return err
	}
	*d = v
	return nil
}
