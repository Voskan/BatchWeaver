package operation

import (
	"fmt"

	"github.com/Voskan/BatchWeaver/diagnostics"
)

// Operation diagnostic codes. Once committed, a code keeps its meaning; see
// docs/reference/diagnostic-codes.md.
const (
	codeInvalidID            diagnostics.Code = "BWOP001"
	codeInvalidSemantics     diagnostics.Code = "BWOP002"
	codeInvalidResult        diagnostics.Code = "BWOP003"
	codeInvalidPartition     diagnostics.Code = "BWOP004"
	codeInvalidScheduler     diagnostics.Code = "BWOP005"
	codeInvalidDedup         diagnostics.Code = "BWOP006"
	codeInvalidRetry         diagnostics.Code = "BWOP007"
	codeInvalidFallback      diagnostics.Code = "BWOP008"
	codeInvalidKeyContract   diagnostics.Code = "BWOP009"
	codeInvalidExtension     diagnostics.Code = "BWOP010"
	codePartitionSemantics   diagnostics.Code = "BWOP011"
	codeDedupIncompatible    diagnostics.Code = "BWOP012"
	codeRetryIncompatible    diagnostics.Code = "BWOP013"
	codeFallbackIncompatible diagnostics.Code = "BWOP014"
	codeMissingSymbols       diagnostics.Code = "BWOP015"
	codeInvalidSymbol        diagnostics.Code = "BWOP016"
)

const operationSource = "operation"

// ValidationError reports that a Spec failed validation. It carries the full
// diagnostic collection so callers can present every problem, not just the
// first. Retrieve it with errors.As.
type ValidationError struct {
	// ID is the operation identifier, when known.
	ID ID
	// Diagnostics holds every validation finding.
	Diagnostics diagnostics.Collection
}

// Error summarizes the validation failure. The message is deterministic and
// contains no pointer addresses.
func (e *ValidationError) Error() string {
	errs := e.Diagnostics.FilterBySeverity(diagnostics.SeverityError)
	first := ""
	if ds := errs.Sorted().Diagnostics(); len(ds) > 0 {
		first = ds[0].Message
	}
	if e.ID != "" {
		return fmt.Sprintf("operation %q is invalid: %d error(s): %s", e.ID, errs.Len(), first)
	}
	return fmt.Sprintf("operation is invalid: %d error(s): %s", errs.Len(), first)
}

// ValidationError returns a non-nil *ValidationError when the spec has
// validation errors, and nil otherwise. It lets callers check a spec without
// importing the diagnostics package directly.
func (s Spec) ValidationError() error {
	diags := s.Validate()
	if diags.HasErrors() {
		return &ValidationError{ID: s.id, Diagnostics: diags}
	}
	return nil
}

// Validate checks the spec's fields and cross-field invariants and returns every
// finding as a diagnostic collection rather than stopping at the first problem.
// Diagnostics carry the spec's source range when available.
func (s Spec) Validate() diagnostics.Collection {
	var c diagnostics.Collection
	rng := s.source.Range

	add := func(code diagnostics.Code, msg string) {
		c.Add(diagnostics.Diagnostic{
			Code: code, Severity: diagnostics.SeverityError,
			Message: msg, Source: operationSource, Range: rng,
		})
	}

	if err := s.id.Validate(); err != nil {
		add(codeInvalidID, err.Error())
	}
	if err := s.semantics.Validate(); err != nil {
		add(codeInvalidSemantics, err.Error())
	}
	if err := s.keyContract.Validate(); err != nil {
		add(codeInvalidKeyContract, err.Error())
	}
	if err := s.resultContract.Validate(); err != nil {
		add(codeInvalidResult, err.Error())
	}
	if err := s.partitionContract.Validate(); err != nil {
		add(codeInvalidPartition, err.Error())
	}
	if err := s.schedulerPolicy.Validate(); err != nil {
		add(codeInvalidScheduler, err.Error())
	}
	if err := s.deduplicationPolicy.Validate(); err != nil {
		add(codeInvalidDedup, err.Error())
	}
	if err := s.retryPolicy.Validate(); err != nil {
		add(codeInvalidRetry, err.Error())
	}
	if err := s.fallbackPolicy.Validate(); err != nil {
		add(codeInvalidFallback, err.Error())
	}
	for _, e := range s.extensions {
		if err := e.Validate(); err != nil {
			add(codeInvalidExtension, err.Error())
		}
	}
	if !s.scalarSymbol.IsZero() {
		if err := s.scalarSymbol.Validate(); err != nil {
			add(codeInvalidSymbol, "scalar "+err.Error())
		}
	}
	if !s.batchSymbol.IsZero() {
		if err := s.batchSymbol.Validate(); err != nil {
			add(codeInvalidSymbol, "batch "+err.Error())
		}
	}

	s.validateCrossField(&c, add)
	return c
}

// validateCrossField applies invariants that span multiple fields. It only runs
// checks whose inputs are individually valid enough to be meaningful, avoiding
// cascading diagnostics from a single root cause.
func (s Spec) validateCrossField(c *diagnostics.Collection, add func(diagnostics.Code, string)) {
	sem := s.semantics
	part := s.partitionContract

	// Configuration-based specs must declare both symbols; a spec with neither
	// symbol is understood to be backed by a typed declaration.
	if s.ConfigBased() && (s.scalarSymbol.IsZero() || s.batchSymbol.IsZero()) {
		add(codeMissingSymbols, "configuration-based operation must declare both scalar and batch symbols")
	}

	// Transaction- and session-bound operations require matching partitioning.
	if sem.kind == KindTransactionBound && !part.requires(DimensionTransaction) {
		add(codePartitionSemantics, "transaction-bound operation must partition by transaction")
	}
	if sem.kind == KindSessionBound && !part.requires(DimensionSession) {
		add(codePartitionSemantics, "session-bound operation must partition by session")
	}

	// Deduplication must be semantically permitted.
	if s.deduplicationPolicy.Enabled() && !sem.DeduplicationAllowed() {
		add(codeDedupIncompatible, fmt.Sprintf("deduplication is not permitted for %s operations", sem.kind))
	}

	// Retry compatibility.
	if s.retryPolicy.Enabled() {
		if !sem.RetryAllowed() {
			add(codeRetryIncompatible, "retry is not permitted by the operation semantics")
		}
		if sem.idempotency == IdempotencyNonIdempotent {
			add(codeRetryIncompatible, "non-idempotent operations cannot enable retry")
		}
		if sem.idempotency == IdempotencyConditional && s.retryPolicy.IdempotencyKey().IsZero() {
			add(codeRetryIncompatible, "conditional idempotency requires an idempotency-key symbol to enable retry")
		}
	}
	if s.retryPolicy.PartialItemRetry() {
		if em := s.resultContract.errorMode; em != ErrorPerItem && em != ErrorMixed {
			add(codeRetryIncompatible, "partial-item retry requires a per-item or mixed result error mode")
		}
	}

	// Fallback compatibility.
	switch s.fallbackPolicy.Mode() {
	case FallbackParallelScalar:
		if sem.idempotency == IdempotencyNonIdempotent {
			add(codeFallbackIncompatible, "non-idempotent operations cannot use parallel scalar fallback")
		}
		if s.ConfigBased() && s.scalarSymbol.IsZero() {
			add(codeFallbackIncompatible, "scalar fallback requires a scalar symbol")
		}
	case FallbackScalar:
		if s.ConfigBased() && s.scalarSymbol.IsZero() {
			add(codeFallbackIncompatible, "scalar fallback requires a scalar symbol")
		}
	}
}
