package diagnostics

import (
	"errors"
	"fmt"
)

// Severity classifies the importance of a Diagnostic.
//
// The zero value is SeverityUnknown so that an uninitialized Diagnostic is
// never silently treated as a real severity level.
type Severity uint8

const (
	// SeverityUnknown is the zero value and indicates an unset severity.
	SeverityUnknown Severity = iota
	// SeverityInfo reports informational findings that require no action.
	SeverityInfo
	// SeverityWarning reports findings that may indicate a problem.
	SeverityWarning
	// SeverityError reports findings that prevent correct or complete work.
	SeverityError
)

// ErrInvalidSeverity is returned when parsing or validating an unknown severity.
var ErrInvalidSeverity = errors.New("invalid severity")

// String returns the stable lowercase name of the severity: "unknown", "info",
// "warning", or "error". Out-of-range values render as "unknown".
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// Valid reports whether s is one of the defined, actionable severity levels.
// SeverityUnknown is not considered valid.
func (s Severity) Valid() bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityError:
		return true
	default:
		return false
	}
}

// sortWeight returns the ordering weight used when sorting diagnostics, with
// errors ordered before warnings before info. Lower sorts first.
func (s Severity) sortWeight() int {
	switch s {
	case SeverityError:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}

// ParseSeverity parses the canonical lowercase name of a severity. Parsing is
// case-sensitive and strict: it accepts only "unknown", "info", "warning", and
// "error", and rejects anything else with an error wrapping ErrInvalidSeverity.
func ParseSeverity(value string) (Severity, error) {
	switch value {
	case "unknown":
		return SeverityUnknown, nil
	case "info":
		return SeverityInfo, nil
	case "warning":
		return SeverityWarning, nil
	case "error":
		return SeverityError, nil
	default:
		return SeverityUnknown, fmt.Errorf("%w: %q", ErrInvalidSeverity, value)
	}
}

// MarshalText implements encoding.TextMarshaler using the canonical name.
func (s Severity) MarshalText() ([]byte, error) {
	if s > SeverityError {
		return nil, fmt.Errorf("%w: %d", ErrInvalidSeverity, uint8(s))
	}
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler with strict parsing.
func (s *Severity) UnmarshalText(data []byte) error {
	parsed, err := ParseSeverity(string(data))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}
