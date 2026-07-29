package runtime

import "fmt"

// QueueLimits bounds a per-operation queue. A zero field means "no limit" for
// that dimension; to avoid accidentally unbounded queues, engine defaults and
// operation specs supply bounded values, and at least one size dimension should
// be set. Negative values are invalid.
type QueueLimits struct {
	// MaxItems bounds the number of unique queued items.
	MaxItems int
	// MaxWaiters bounds the number of logical waiters (including duplicates).
	MaxWaiters int
	// MaxBytes bounds the estimated queued key bytes.
	MaxBytes int64
	// MaxWeight bounds the aggregate queued item weight.
	MaxWeight int64
	// MaxPartitions bounds the number of active partitions.
	MaxPartitions int
}

// Validate reports whether the limits are well-formed (no negative values).
func (l QueueLimits) Validate() error {
	if l.MaxItems < 0 || l.MaxWaiters < 0 || l.MaxBytes < 0 || l.MaxWeight < 0 || l.MaxPartitions < 0 {
		return fmt.Errorf("%w: queue limits must not be negative", ErrInvalidOption)
	}
	return nil
}

// merge returns a copy of l with any zero field replaced by the corresponding
// field of other.
func (l QueueLimits) merge(other QueueLimits) QueueLimits {
	if l.MaxItems == 0 {
		l.MaxItems = other.MaxItems
	}
	if l.MaxWaiters == 0 {
		l.MaxWaiters = other.MaxWaiters
	}
	if l.MaxBytes == 0 {
		l.MaxBytes = other.MaxBytes
	}
	if l.MaxWeight == 0 {
		l.MaxWeight = other.MaxWeight
	}
	if l.MaxPartitions == 0 {
		l.MaxPartitions = other.MaxPartitions
	}
	return l
}

// OverflowPolicy selects what happens when a queue limit is reached.
type OverflowPolicy uint8

const (
	// OverflowReject rejects the submission with a QueueFullError.
	OverflowReject OverflowPolicy = iota
	// OverflowFallback runs the scalar fallback for the submission.
	OverflowFallback
	// OverflowBlock blocks the caller until capacity is available or its context
	// ends.
	OverflowBlock
)

var overflowPolicyNames = []string{"reject", "fallback", "block"}

// String returns the canonical name of the overflow policy.
func (p OverflowPolicy) String() string {
	if int(p) < len(overflowPolicyNames) {
		return overflowPolicyNames[p]
	}
	return "unknown"
}

// FlushReason records why a batch was dispatched. It appears in hooks and
// statistics.
type FlushReason uint8

const (
	// FlushManual is an explicit flush, drain, or close.
	FlushManual FlushReason = iota
	// FlushSize is the maximum item count trigger.
	FlushSize
	// FlushWeight is the maximum aggregate weight trigger.
	FlushWeight
	// FlushBytes is the maximum estimated bytes trigger.
	FlushBytes
	// FlushWait is the maximum wait duration trigger.
	FlushWait
	// FlushDeadline is the earliest-deadline safety-margin trigger.
	FlushDeadline
	// FlushScopeClose is dispatch caused by scope closure.
	FlushScopeClose
	// FlushEngineClose is dispatch caused by engine closure.
	FlushEngineClose
	// FlushOverflow is early dispatch requested by an overflow policy.
	FlushOverflow
)

var flushReasonNames = []string{
	"manual", "size", "weight", "bytes", "wait", "deadline",
	"scope-close", "engine-close", "overflow",
}

// String returns the canonical name of the flush reason.
func (r FlushReason) String() string {
	if int(r) < len(flushReasonNames) {
		return flushReasonNames[r]
	}
	return "unknown"
}
