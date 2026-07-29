package runtime

import "time"

// EventKind identifies a runtime event. Event kinds are stable and may be
// switched on by observers.
type EventKind uint8

const (
	// EventRequestSubmitted is emitted when a request is accepted.
	EventRequestSubmitted EventKind = iota
	// EventRequestRejected is emitted when a request is rejected (queue/partition
	// full or engine/scope closed).
	EventRequestRejected
	// EventRequestJoinedInFlight is emitted when a request joins an existing
	// in-flight item.
	EventRequestJoinedInFlight
	// EventRequestCanceled is emitted when a caller cancels.
	EventRequestCanceled
	// EventMemoHit is emitted when a scope memoization hit serves a request.
	EventMemoHit
	// EventBatchSelected is emitted when items are selected into a batch.
	EventBatchSelected
	// EventBatchDispatched is emitted when a provider call starts.
	EventBatchDispatched
	// EventBatchCompleted is emitted when a provider call completes successfully.
	EventBatchCompleted
	// EventBatchFailed is emitted when a provider call fails globally or panics.
	EventBatchFailed
	// EventContractViolation is emitted when a provider response is malformed.
	EventContractViolation
	// EventFallbackStarted is emitted when a scalar fallback begins.
	EventFallbackStarted
	// EventFallbackCompleted is emitted when a scalar fallback ends.
	EventFallbackCompleted
	// EventScopeOpened is emitted when a scope opens.
	EventScopeOpened
	// EventScopeClosed is emitted when a scope closes.
	EventScopeClosed
	// EventEngineClosing is emitted when the engine begins closing.
	EventEngineClosing
	// EventEngineClosed is emitted when the engine finishes closing.
	EventEngineClosed
)

// Event carries redacted, non-sensitive metadata about a runtime event. It never
// contains raw keys or partition components.
type Event struct {
	// Kind is the event kind.
	Kind EventKind
	// Time is when the event occurred, per the engine clock.
	Time time.Time
	// Operation is the operation ID, when applicable.
	Operation string
	// Partition is a redacted partition token, when applicable.
	Partition string
	// ScopeID is the scope identity, when applicable.
	ScopeID string
	// BatchID is the batch identity, when applicable.
	BatchID uint64
	// Count is an event-specific count, such as batch size.
	Count int
	// FlushReason is the dispatch reason, for batch events.
	FlushReason FlushReason
	// ErrorClass is a non-sensitive error classification, for failure events.
	ErrorClass string
}

// Hooks is a backend-neutral set of observability callbacks. All fields are
// optional. Callbacks are invoked synchronously; a panic in a callback is
// recovered and reported through OnError without affecting runtime correctness.
type Hooks struct {
	// OnEvent receives runtime events. It must not block for long.
	OnEvent func(Event)
	// OnError receives internal or hook-panic errors.
	OnError func(error)
}

// emit dispatches an event to the engine hooks, recovering panics.
func (e *Engine) emit(ev Event) {
	if e.hooks.OnEvent == nil {
		return
	}
	if ev.Time.IsZero() {
		ev.Time = e.clock.Now()
	}
	defer func() {
		if r := recover(); r != nil {
			e.reportHookError(&CallbackPanicError{Callback: "hook", Value: sanitizePanic(r)})
		}
	}()
	e.hooks.OnEvent(ev)
}

// reportHookError delivers an error to the OnError hook, recovering panics.
func (e *Engine) reportHookError(err error) {
	if e.hooks.OnError == nil {
		return
	}
	defer func() { _ = recover() }()
	e.hooks.OnError(err)
}

func (e *Engine) emitEngineClosing() { e.emit(Event{Kind: EventEngineClosing}) }
func (e *Engine) emitEngineClosed()  { e.emit(Event{Kind: EventEngineClosed}) }
