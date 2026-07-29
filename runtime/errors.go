package runtime

import (
	"errors"
	"fmt"
)

// Sentinel errors for programmatic handling with errors.Is. Structured errors
// below wrap these where useful. None of these expose raw key or partition data.
var (
	// ErrEngineClosed is returned when using an engine that has fully closed.
	ErrEngineClosed = errors.New("batchweaver runtime: engine closed")
	// ErrEngineClosing is returned when submitting new work while the engine is
	// closing.
	ErrEngineClosing = errors.New("batchweaver runtime: engine closing")
	// ErrScopeRequired is returned when an operation is invoked without a scope
	// and no direct fallback policy permits scope-less execution.
	ErrScopeRequired = errors.New("batchweaver runtime: scope required")
	// ErrScopeClosed is returned when submitting to a closed scope.
	ErrScopeClosed = errors.New("batchweaver runtime: scope closed")
	// ErrOperationAlreadyBound is returned when binding an operation ID that is
	// already bound on the engine.
	ErrOperationAlreadyBound = errors.New("batchweaver runtime: operation already bound")
	// ErrItemTooLarge is returned when a single item exceeds a hard batch limit.
	ErrItemTooLarge = errors.New("batchweaver runtime: item too large")
	// ErrDeadlineBeforeDispatch is returned when a caller deadline is too close
	// to enqueue safely under the deadline policy.
	ErrDeadlineBeforeDispatch = errors.New("batchweaver runtime: deadline before dispatch")
	// ErrRecursiveOperation is returned when a provider calls the same operation
	// and partition it is currently executing, which would deadlock.
	ErrRecursiveOperation = errors.New("batchweaver runtime: recursive operation")
	// ErrFlushTimeout is returned when a flush does not complete before its
	// context expires.
	ErrFlushTimeout = errors.New("batchweaver runtime: flush timeout")
	// ErrDrainTimeout is returned when a drain does not complete before its
	// context expires.
	ErrDrainTimeout = errors.New("batchweaver runtime: drain timeout")
	// ErrInvalidBinding is returned when a binding configuration is invalid.
	ErrInvalidBinding = errors.New("batchweaver runtime: invalid binding")
	// ErrInvalidOption is returned when an engine or scope option is invalid.
	ErrInvalidOption = errors.New("batchweaver runtime: invalid option")
)

// QueueFullError reports that a bounded queue rejected an item. It carries the
// operation ID and the limit category but never raw key or partition data.
type QueueFullError struct {
	// Operation is the operation ID.
	Operation string
	// Limit names the exceeded limit category, for example "items" or "bytes".
	Limit string
}

// Error implements error.
func (e *QueueFullError) Error() string {
	return fmt.Sprintf("batchweaver runtime: queue full for operation %q (%s limit)", e.Operation, e.Limit)
}

// Is reports QueueFullError as matching ErrQueueFull.
func (e *QueueFullError) Is(target error) bool { return target == ErrQueueFull }

// ErrQueueFull matches any QueueFullError via errors.Is.
var ErrQueueFull = errors.New("batchweaver runtime: queue full")

// PartitionLimitError reports that the active-partition limit was reached.
type PartitionLimitError struct {
	// Operation is the operation ID.
	Operation string
	// Limit is the configured maximum number of active partitions.
	Limit int
}

// Error implements error.
func (e *PartitionLimitError) Error() string {
	return fmt.Sprintf("batchweaver runtime: partition limit %d reached for operation %q", e.Limit, e.Operation)
}

// Is reports PartitionLimitError as matching ErrPartitionLimit.
func (e *PartitionLimitError) Is(target error) bool { return target == ErrPartitionLimit }

// ErrPartitionLimit matches any PartitionLimitError via errors.Is.
var ErrPartitionLimit = errors.New("batchweaver runtime: partition limit reached")

// ProviderPanicError reports that a provider function panicked. The recovered
// value is rendered as a string; no stack trace is exposed by default.
type ProviderPanicError struct {
	// Operation is the operation ID.
	Operation string
	// Value is the sanitized recovered panic value.
	Value string
}

// Error implements error.
func (e *ProviderPanicError) Error() string {
	return fmt.Sprintf("batchweaver runtime: provider for operation %q panicked: %s", e.Operation, e.Value)
}

// Is reports ProviderPanicError as matching ErrProviderPanic.
func (e *ProviderPanicError) Is(target error) bool { return target == ErrProviderPanic }

// ErrProviderPanic matches any ProviderPanicError via errors.Is.
var ErrProviderPanic = errors.New("batchweaver runtime: provider panic")

// CallbackPanicError reports that a user callback (partitioner, key strategy,
// weight function, or fallback) panicked while handling a request.
type CallbackPanicError struct {
	// Operation is the operation ID.
	Operation string
	// Callback names the callback that panicked.
	Callback string
	// Value is the sanitized recovered panic value.
	Value string
}

// Error implements error.
func (e *CallbackPanicError) Error() string {
	return fmt.Sprintf("batchweaver runtime: %s callback for operation %q panicked: %s", e.Callback, e.Operation, e.Value)
}

// Is reports CallbackPanicError as matching ErrCallbackPanic.
func (e *CallbackPanicError) Is(target error) bool { return target == ErrCallbackPanic }

// ErrCallbackPanic matches any CallbackPanicError via errors.Is.
var ErrCallbackPanic = errors.New("batchweaver runtime: callback panic")

// ContractError reports that a provider response violated the batch contract in
// a way that makes result mapping untrustworthy.
type ContractError struct {
	// Operation is the operation ID.
	Operation string
	// Reason is a stable, non-sensitive description of the violation.
	Reason string
}

// Error implements error.
func (e *ContractError) Error() string {
	return fmt.Sprintf("batchweaver runtime: contract violation for operation %q: %s", e.Operation, e.Reason)
}

// Is reports ContractError as matching ErrBatchContractViolation.
func (e *ContractError) Is(target error) bool { return target == ErrBatchContractViolation }

// ErrBatchContractViolation matches any ContractError via errors.Is.
var ErrBatchContractViolation = errors.New("batchweaver runtime: batch contract violation")

// ErrMissingResult indicates a required result was missing from a provider
// response, when the operation contract treats that as an error.
var ErrMissingResult = errors.New("batchweaver runtime: missing result")
