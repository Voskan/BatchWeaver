package operation

import (
	"errors"
	"fmt"
)

// ResultMode describes how a native batch provider maps outcomes back to the
// individual logical requests.
type ResultMode uint8

const (
	// ResultOrdered maps outcomes to request items by request ID or position.
	ResultOrdered ResultMode = iota
	// ResultKeyed maps outcomes by canonical key.
	ResultKeyed
	// ResultSparse omits absent keys, resolved through missing-result semantics.
	ResultSparse
)

// ErrInvalidResultMode is returned for an unknown result mode.
var ErrInvalidResultMode = errors.New("invalid result mode")

var resultModeNames = []string{"ordered", "keyed", "sparse"}

// String returns the canonical name of the result mode.
func (m ResultMode) String() string { return enumString(m, resultModeNames) }

// Valid reports whether m is a defined result mode.
func (m ResultMode) Valid() bool { return int(m) < len(resultModeNames) }

// ParseResultMode resolves the canonical name of a result mode.
func ParseResultMode(s string) (ResultMode, error) {
	return parseEnum[ResultMode](s, resultModeNames, ErrInvalidResultMode)
}

// MarshalText implements encoding.TextMarshaler.
func (m ResultMode) MarshalText() ([]byte, error) {
	return marshalEnum(m, resultModeNames, ErrInvalidResultMode)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *ResultMode) UnmarshalText(data []byte) error {
	v, err := ParseResultMode(string(data))
	if err != nil {
		return err
	}
	*m = v
	return nil
}

// MissingBehavior describes what happens when a requested item has no outcome.
type MissingBehavior uint8

const (
	// MissingNotFound represents absence as a not-found outcome.
	MissingNotFound MissingBehavior = iota
	// MissingError represents absence as a per-item error.
	MissingError
	// MissingZeroValue substitutes the zero value; it may hide missing data and
	// requires explicit opt-in.
	MissingZeroValue
	// MissingContractViolation treats absence as a provider contract violation.
	MissingContractViolation
)

// ErrInvalidMissingBehavior is returned for an unknown missing behavior.
var ErrInvalidMissingBehavior = errors.New("invalid missing behavior")

var missingBehaviorNames = []string{"not-found", "error", "zero-value", "contract-violation"}

// String returns the canonical name of the missing behavior.
func (m MissingBehavior) String() string { return enumString(m, missingBehaviorNames) }

// Valid reports whether m is a defined missing behavior.
func (m MissingBehavior) Valid() bool { return int(m) < len(missingBehaviorNames) }

// ParseMissingBehavior resolves the canonical name of a missing behavior.
func ParseMissingBehavior(s string) (MissingBehavior, error) {
	return parseEnum[MissingBehavior](s, missingBehaviorNames, ErrInvalidMissingBehavior)
}

// MarshalText implements encoding.TextMarshaler.
func (m MissingBehavior) MarshalText() ([]byte, error) {
	return marshalEnum(m, missingBehaviorNames, ErrInvalidMissingBehavior)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *MissingBehavior) UnmarshalText(data []byte) error {
	v, err := ParseMissingBehavior(string(data))
	if err != nil {
		return err
	}
	*m = v
	return nil
}

// ErrorMode describes how errors are attributed within a batch response.
type ErrorMode uint8

const (
	// ErrorPerItem attributes errors to individual items.
	ErrorPerItem ErrorMode = iota
	// ErrorGlobal attributes a single error to the whole batch.
	ErrorGlobal
	// ErrorMixed permits both per-item and global errors.
	ErrorMixed
)

// ErrInvalidErrorMode is returned for an unknown error mode.
var ErrInvalidErrorMode = errors.New("invalid error mode")

var errorModeNames = []string{"per-item", "global", "mixed"}

// String returns the canonical name of the error mode.
func (m ErrorMode) String() string { return enumString(m, errorModeNames) }

// Valid reports whether m is a defined error mode.
func (m ErrorMode) Valid() bool { return int(m) < len(errorModeNames) }

// ParseErrorMode resolves the canonical name of an error mode.
func ParseErrorMode(s string) (ErrorMode, error) {
	return parseEnum[ErrorMode](s, errorModeNames, ErrInvalidErrorMode)
}

// MarshalText implements encoding.TextMarshaler.
func (m ErrorMode) MarshalText() ([]byte, error) {
	return marshalEnum(m, errorModeNames, ErrInvalidErrorMode)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *ErrorMode) UnmarshalText(data []byte) error {
	v, err := ParseErrorMode(string(data))
	if err != nil {
		return err
	}
	*m = v
	return nil
}

// DuplicateResultBehavior describes how duplicate outcomes for the same request
// are handled.
type DuplicateResultBehavior uint8

const (
	// DuplicateContractViolation rejects duplicates as a contract violation.
	DuplicateContractViolation DuplicateResultBehavior = iota
	// DuplicateFirst keeps the first outcome.
	DuplicateFirst
	// DuplicateLast keeps the last outcome.
	DuplicateLast
	// DuplicateCustom defers resolution to a custom policy defined later.
	DuplicateCustom
)

// ErrInvalidDuplicateBehavior is returned for an unknown duplicate behavior.
var ErrInvalidDuplicateBehavior = errors.New("invalid duplicate-result behavior")

var duplicateBehaviorNames = []string{"contract-violation", "first", "last", "custom"}

// String returns the canonical name of the duplicate behavior.
func (d DuplicateResultBehavior) String() string { return enumString(d, duplicateBehaviorNames) }

// Valid reports whether d is a defined duplicate behavior.
func (d DuplicateResultBehavior) Valid() bool { return int(d) < len(duplicateBehaviorNames) }

// ParseDuplicateResultBehavior resolves the canonical name of a duplicate behavior.
func ParseDuplicateResultBehavior(s string) (DuplicateResultBehavior, error) {
	return parseEnum[DuplicateResultBehavior](s, duplicateBehaviorNames, ErrInvalidDuplicateBehavior)
}

// MarshalText implements encoding.TextMarshaler.
func (d DuplicateResultBehavior) MarshalText() ([]byte, error) {
	return marshalEnum(d, duplicateBehaviorNames, ErrInvalidDuplicateBehavior)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *DuplicateResultBehavior) UnmarshalText(data []byte) error {
	v, err := ParseDuplicateResultBehavior(string(data))
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// UnexpectedResultBehavior describes how a provider returning an unrequested
// key or request ID is handled. Version 1 provides no ignore-by-default mode.
type UnexpectedResultBehavior uint8

const (
	// UnexpectedContractViolation rejects unexpected outcomes as a contract
	// violation. This is the version-1 default.
	UnexpectedContractViolation UnexpectedResultBehavior = iota
	// UnexpectedError reports unexpected outcomes as an error to the caller.
	UnexpectedError
)

// ErrInvalidUnexpectedBehavior is returned for an unknown unexpected behavior.
var ErrInvalidUnexpectedBehavior = errors.New("invalid unexpected-result behavior")

var unexpectedBehaviorNames = []string{"contract-violation", "error"}

// String returns the canonical name of the unexpected behavior.
func (u UnexpectedResultBehavior) String() string { return enumString(u, unexpectedBehaviorNames) }

// Valid reports whether u is a defined unexpected behavior.
func (u UnexpectedResultBehavior) Valid() bool { return int(u) < len(unexpectedBehaviorNames) }

// ParseUnexpectedResultBehavior resolves the canonical name.
func ParseUnexpectedResultBehavior(s string) (UnexpectedResultBehavior, error) {
	return parseEnum[UnexpectedResultBehavior](s, unexpectedBehaviorNames, ErrInvalidUnexpectedBehavior)
}

// MarshalText implements encoding.TextMarshaler.
func (u UnexpectedResultBehavior) MarshalText() ([]byte, error) {
	return marshalEnum(u, unexpectedBehaviorNames, ErrInvalidUnexpectedBehavior)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (u *UnexpectedResultBehavior) UnmarshalText(data []byte) error {
	v, err := ParseUnexpectedResultBehavior(string(data))
	if err != nil {
		return err
	}
	*u = v
	return nil
}

// ErrInvalidResultContract is returned when a ResultContract fails validation.
var ErrInvalidResultContract = errors.New("invalid result contract")

// ResultContract is a validated description of how batch outcomes map back to
// logical callers. Construct it through OrderedResults, KeyedResults, or
// SparseResults.
type ResultContract struct {
	mode                 ResultMode
	missing              MissingBehavior
	errorMode            ErrorMode
	duplicate            DuplicateResultBehavior
	unexpected           UnexpectedResultBehavior
	orderSignificant     bool
	unknownRepresentable bool
}

// Mode returns the result mode.
func (c ResultContract) Mode() ResultMode { return c.mode }

// Missing returns the missing-result behavior.
func (c ResultContract) Missing() MissingBehavior { return c.missing }

// ErrorMode returns the error attribution mode.
func (c ResultContract) ErrorMode() ErrorMode { return c.errorMode }

// Duplicate returns the duplicate-result behavior.
func (c ResultContract) Duplicate() DuplicateResultBehavior { return c.duplicate }

// Unexpected returns the unexpected-result behavior.
func (c ResultContract) Unexpected() UnexpectedResultBehavior { return c.unexpected }

// OrderSignificant reports whether response order carries meaning.
func (c ResultContract) OrderSignificant() bool { return c.orderSignificant }

// UnknownRepresentable reports whether unknown outcomes can be represented.
func (c ResultContract) UnknownRepresentable() bool { return c.unknownRepresentable }

// WithErrorMode returns a copy using the given error mode.
func (c ResultContract) WithErrorMode(m ErrorMode) ResultContract {
	c.errorMode = m
	return c
}

// WithDuplicate returns a copy using the given duplicate behavior.
func (c ResultContract) WithDuplicate(d DuplicateResultBehavior) ResultContract {
	c.duplicate = d
	return c
}

// WithMissing returns a copy using the given missing behavior.
func (c ResultContract) WithMissing(m MissingBehavior) ResultContract {
	c.missing = m
	return c
}

// WithUnexpected returns a copy using the given unexpected behavior.
func (c ResultContract) WithUnexpected(u UnexpectedResultBehavior) ResultContract {
	c.unexpected = u
	return c
}

// Validate reports whether the result contract is internally consistent.
func (c ResultContract) Validate() error {
	if !c.mode.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidResultContract, ErrInvalidResultMode)
	}
	if !c.missing.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidResultContract, ErrInvalidMissingBehavior)
	}
	if !c.errorMode.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidResultContract, ErrInvalidErrorMode)
	}
	if !c.duplicate.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidResultContract, ErrInvalidDuplicateBehavior)
	}
	if !c.unexpected.Valid() {
		return fmt.Errorf("%w: %w", ErrInvalidResultContract, ErrInvalidUnexpectedBehavior)
	}
	if c.mode == ResultSparse && c.missing == MissingContractViolation {
		return fmt.Errorf("%w: sparse results cannot treat missing as a contract violation", ErrInvalidResultContract)
	}
	if c.mode == ResultOrdered && !c.orderSignificant {
		return fmt.Errorf("%w: ordered results must treat order as significant", ErrInvalidResultContract)
	}
	return nil
}

// OrderedResults returns a contract where outcomes correspond to request order.
func OrderedResults() ResultContract {
	return ResultContract{
		mode: ResultOrdered, missing: MissingError, errorMode: ErrorPerItem,
		duplicate: DuplicateContractViolation, unexpected: UnexpectedContractViolation,
		orderSignificant: true,
	}
}

// KeyedResults returns a contract where outcomes are associated with keys.
func KeyedResults() ResultContract {
	return ResultContract{
		mode: ResultKeyed, missing: MissingError, errorMode: ErrorPerItem,
		duplicate: DuplicateContractViolation, unexpected: UnexpectedContractViolation,
	}
}

// SparseResults returns a contract where absent keys are omitted and resolved
// through the given missing behavior. It panics if missing is
// MissingContractViolation, which is nonsensical for sparse results.
func SparseResults(missing MissingBehavior) ResultContract {
	if missing == MissingContractViolation {
		panic("operation.SparseResults: missing behavior cannot be contract-violation")
	}
	return ResultContract{
		mode: ResultSparse, missing: missing, errorMode: ErrorPerItem,
		duplicate: DuplicateContractViolation, unexpected: UnexpectedContractViolation,
	}
}
