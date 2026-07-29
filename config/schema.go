package config

import (
	"errors"

	"github.com/Voskan/BatchWeaver/operation"
)

// CompilerMode selects how the (future) compiler integrates with a build.
type CompilerMode uint8

const (
	// CompilerTransparent is the default: transparent source-level integration.
	CompilerTransparent CompilerMode = iota
	// CompilerDisabled turns compiler integration off.
	CompilerDisabled
)

// ErrInvalidCompilerMode is returned for an unknown compiler mode.
var ErrInvalidCompilerMode = errors.New("invalid compiler mode")

var compilerModeNames = []string{"transparent", "disabled"}

// String returns the canonical name of the compiler mode.
func (m CompilerMode) String() string { return enumName(uint8(m), compilerModeNames) }

// ParseCompilerMode resolves the canonical name of a compiler mode.
func ParseCompilerMode(s string) (CompilerMode, error) {
	v, ok := enumValue(s, compilerModeNames)
	if !ok {
		return 0, ErrInvalidCompilerMode
	}
	return CompilerMode(v), nil
}

// LoggingLevel selects runtime logging verbosity.
type LoggingLevel uint8

const (
	// LoggingSilent disables logging.
	LoggingSilent LoggingLevel = iota
	// LoggingErrors logs only errors.
	LoggingErrors
	// LoggingWarnings logs warnings and errors; the default.
	LoggingWarnings
	// LoggingInfo logs informational messages.
	LoggingInfo
	// LoggingDebug logs debug detail.
	LoggingDebug
)

// ErrInvalidLoggingLevel is returned for an unknown logging level.
var ErrInvalidLoggingLevel = errors.New("invalid logging level")

var loggingLevelNames = []string{"silent", "errors", "warnings", "info", "debug"}

// String returns the canonical name of the logging level.
func (l LoggingLevel) String() string { return enumName(uint8(l), loggingLevelNames) }

// ParseLoggingLevel resolves the canonical name of a logging level.
func ParseLoggingLevel(s string) (LoggingLevel, error) {
	v, ok := enumValue(s, loggingLevelNames)
	if !ok {
		return 0, ErrInvalidLoggingLevel
	}
	return LoggingLevel(v), nil
}

// Compiler holds compiler-related settings.
type Compiler struct {
	// Mode selects compiler integration behavior.
	Mode CompilerMode
}

// Runtime holds runtime default settings.
type Runtime struct {
	// DefaultScope is the default batching scope for operations.
	DefaultScope operation.Scope
}

// Security holds conservative security defaults.
type Security struct {
	// CrossScopeBatching permits batching across root scopes; default false.
	CrossScopeBatching bool
	// RawKeyObservability permits exposing raw keys in observability; default false.
	RawKeyObservability bool
}

// Observability holds observability defaults.
type Observability struct {
	// Metrics enables metrics.
	Metrics bool
	// Tracing enables tracing.
	Tracing bool
	// Logging selects logging verbosity.
	Logging LoggingLevel
}

// Config is the normalized, validated effective configuration. It is produced by
// the loader after decoding, include expansion, merging, defaulting, and
// normalization. Operations are exposed through Catalog.
type Config struct {
	// Version is the schema version.
	Version int
	// Compiler holds compiler settings.
	Compiler Compiler
	// Runtime holds runtime defaults.
	Runtime Runtime
	// Security holds security defaults.
	Security Security
	// Observability holds observability defaults.
	Observability Observability
	// Catalog holds the normalized operation specs.
	Catalog operation.Catalog
	// Extensions holds preserved, uninterpreted top-level extension data.
	Extensions []operation.Extension
}

// enumName returns names[v] or "unknown" when out of range.
func enumName(v uint8, names []string) string {
	if int(v) < len(names) {
		return names[v]
	}
	return "unknown"
}

// enumValue resolves s against names, returning its index and whether found.
func enumValue(s string, names []string) (uint8, bool) {
	for i, n := range names {
		if n == s {
			return uint8(i), true
		}
	}
	return 0, false
}
