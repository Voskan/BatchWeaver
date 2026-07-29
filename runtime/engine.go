package runtime

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
)

// EngineState is the lifecycle state of an Engine.
type EngineState uint8

const (
	// EngineStateOpen accepts bindings and submissions.
	EngineStateOpen EngineState = iota
	// EngineStateClosing rejects new submissions and bindings while draining.
	EngineStateClosing
	// EngineStateClosed has released all resources.
	EngineStateClosed
)

var engineStateNames = []string{"open", "closing", "closed"}

// String returns the canonical name of the engine state.
func (s EngineState) String() string {
	if int(s) < len(engineStateNames) {
		return engineStateNames[s]
	}
	return "unknown"
}

// PanicPolicy selects how provider and callback panics are handled.
type PanicPolicy uint8

const (
	// PanicPolicyRecover recovers panics and converts them into typed errors.
	// This is the safe default for a library runtime.
	PanicPolicyRecover PanicPolicy = iota
	// PanicPolicyRepanic re-panics after best-effort cleanup, for callers that
	// require crash semantics.
	PanicPolicyRepanic
)

// operationController is the engine's non-generic view of a bound operation,
// used for lifecycle and statistics without knowing the operation's key/value
// types.
type operationController interface {
	id() string
	flush(ctx context.Context) error
	drain(ctx context.Context) error
	closeController(ctx context.Context) error
	stats() OperationStats
	dropMemo(scopeID uint64)
}

// Engine is an instance-scoped BatchWeaver runtime. It owns bound operations,
// their coordinator goroutines, lifecycle cancellation, and statistics. There is
// no global mutable registry; every binding belongs to an explicit Engine.
//
// An Engine is safe for concurrent use.
type Engine struct {
	mu    sync.Mutex
	state EngineState
	ops   map[string]operationController

	clock           Clock
	defaultLimits   QueueLimits
	defaultOverflow OverflowPolicy
	hooks           Hooks
	panicPolicy     PanicPolicy
	statsEnabled    bool

	engineID string
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	scopeSeq   atomic.Uint64
	requestSeq atomic.Uint64
	counters   engineCounters
}

// NewEngine constructs an Engine, validating options. It starts no goroutines
// until an operation is bound, so a failed construction leaks nothing.
func NewEngine(options ...EngineOption) (*Engine, error) {
	e := &Engine{
		state:           EngineStateOpen,
		ops:             make(map[string]operationController),
		clock:           SystemClock(),
		defaultOverflow: OverflowReject,
		defaultLimits: QueueLimits{
			MaxItems:      100_000,
			MaxBytes:      64 << 20,
			MaxPartitions: 10_000,
		},
		panicPolicy:  PanicPolicyRecover,
		statsEnabled: true,
	}
	for _, opt := range options {
		if err := opt(e); err != nil {
			return nil, err
		}
	}
	if err := e.defaultLimits.Validate(); err != nil {
		return nil, err
	}
	if e.clock == nil {
		return nil, fmt.Errorf("%w: clock is nil", ErrInvalidOption)
	}
	e.engineID = strconv.FormatUint(newEngineID(), 36)
	e.ctx, e.cancel = context.WithCancel(context.Background())
	return e, nil
}

// State returns the current engine state.
func (e *Engine) State() EngineState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

// nextRequestID returns a unique, engine-local request ID. It never returns
// zero, which is reserved as invalid.
func (e *Engine) nextRequestID() uint64 {
	for {
		id := e.requestSeq.Add(1)
		if id != 0 {
			return id
		}
	}
}

// registerController adds a controller under its operation ID, rejecting
// duplicates and bindings on a non-open engine.
func (e *Engine) registerController(c operationController) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != EngineStateOpen {
		return ErrEngineClosing
	}
	if _, ok := e.ops[c.id()]; ok {
		return fmt.Errorf("%w: %q", ErrOperationAlreadyBound, c.id())
	}
	e.ops[c.id()] = c
	return nil
}

// controllersSnapshot returns the controllers in deterministic ID order.
func (e *Engine) controllersSnapshot() []operationController {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]operationController, 0, len(e.ops))
	for _, c := range e.ops {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id() < out[j].id() })
	return out
}

// acceptingSubmissions reports whether new submissions are allowed.
func (e *Engine) acceptingSubmissions() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch e.state {
	case EngineStateOpen:
		return nil
	case EngineStateClosing:
		return ErrEngineClosing
	default:
		return ErrEngineClosed
	}
}

// Flush dispatches currently eligible queued work across all operations. It
// returns when the dispatch barrier is reached or the context expires.
func (e *Engine) Flush(ctx context.Context) error {
	for _, c := range e.controllersSnapshot() {
		if err := c.flush(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Close transitions the engine through closing to closed. It rejects new
// submissions, drains in-flight work bounded by ctx, stops coordinator
// goroutines, and is idempotent. It returns a context error if ctx expires
// before draining completes; the engine still reaches a valid closed state.
func (e *Engine) Close(ctx context.Context) error {
	e.mu.Lock()
	if e.state == EngineStateClosed {
		e.mu.Unlock()
		return nil
	}
	if e.state == EngineStateOpen {
		e.state = EngineStateClosing
	}
	e.mu.Unlock()

	e.emitEngineClosing()

	var closeErr error
	for _, c := range e.controllersSnapshot() {
		if err := c.closeController(ctx); err != nil && closeErr == nil {
			closeErr = err
		}
	}

	// Cancel the engine lifecycle and wait for coordinator goroutines, bounded
	// by the caller context.
	e.cancel()
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		if closeErr == nil {
			closeErr = fmt.Errorf("%w: %w", ErrDrainTimeout, ctx.Err())
		}
	}

	e.mu.Lock()
	e.state = EngineStateClosed
	e.mu.Unlock()
	e.emitEngineClosed()
	return closeErr
}

// newEngineID returns a process-unique engine identity component.
var engineIDSeq atomic.Uint64

func newEngineID() uint64 { return engineIDSeq.Add(1) }

// ensureOpen returns an error if the engine is not open, for binding.
func (e *Engine) ensureOpen() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != EngineStateOpen {
		return ErrEngineClosing
	}
	return nil
}
