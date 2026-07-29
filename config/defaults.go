package config

import "github.com/Voskan/BatchWeaver/operation"

// This file centralizes every configuration default so defaults are applied in
// exactly one place (normalization) rather than scattered across the decoder,
// validator, and CLI. Security-sensitive defaults are conservative.

// defaultConfig returns the top-level defaults applied when sections are omitted.
// Cross-scope batching and raw-key observability are disabled by default;
// metrics and tracing are off by default to avoid imposing overhead; logging
// defaults to warnings.
func defaultConfig() Config {
	return Config{
		Version:       CurrentSchemaVersion,
		Compiler:      Compiler{Mode: CompilerTransparent},
		Runtime:       Runtime{DefaultScope: operation.ScopeRequest},
		Security:      Security{CrossScopeBatching: false, RawKeyObservability: false},
		Observability: Observability{Metrics: false, Tracing: false, Logging: LoggingWarnings},
	}
}

// Operation-level defaults, applied when an operation omits the field.
const (
	// defaultResultMode is the default result mode.
	defaultResultMode = operation.ResultOrdered
	// defaultMissingBehavior treats missing results as a contract violation
	// unless the operation explicitly models them otherwise.
	defaultMissingBehavior = operation.MissingContractViolation
	// defaultErrorMode is the default error attribution mode.
	defaultErrorMode = operation.ErrorPerItem
	// defaultScope is the default partition scope for an operation.
	defaultScope = operation.ScopeRequest
	// defaultFallbackMode is the conservative default fallback mode.
	defaultFallbackMode = operation.FallbackScalar
	// defaultDeduplicationMode disables deduplication by default.
	defaultDeduplicationMode = operation.DeduplicationDisabled
)
