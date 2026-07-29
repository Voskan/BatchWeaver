package diagnostics

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

// String returns the stable lowercase name of the severity, for example
// "warning". Unknown or out-of-range values render as "unknown".
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
