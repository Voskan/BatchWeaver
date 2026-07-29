package runtime

import (
	"context"
	"errors"
	"strconv"
	"sync"
)

// ErrNestedScope is returned when a nested scope is created under the
// RejectNested policy.
var ErrNestedScope = errors.New("batchweaver runtime: nested scope rejected")

// ScopeState is the lifecycle state of a Scope.
type ScopeState uint8

const (
	// ScopeStateOpen accepts submissions.
	ScopeStateOpen ScopeState = iota
	// ScopeStateClosed rejects submissions and has released scope state.
	ScopeStateClosed
)

// String returns the canonical name of the scope state.
func (s ScopeState) String() string {
	if s == ScopeStateOpen {
		return "open"
	}
	return "closed"
}

// ScopeID identifies a scope within an engine.
type ScopeID uint64

// String renders the scope ID.
func (id ScopeID) String() string { return strconv.FormatUint(uint64(id), 10) }

// NestedPolicy controls what happens when a scope is created under a context
// that already carries one.
type NestedPolicy uint8

const (
	// NestedIsolateChild creates an independent child scope (the default).
	NestedIsolateChild NestedPolicy = iota
	// NestedReuseParent reuses the existing scope.
	NestedReuseParent
	// NestedRejectNested returns ErrNestedScope.
	NestedRejectNested
)

// ScopeOption configures a scope.
type ScopeOption func(*scopeConfig)

type scopeConfig struct {
	nested NestedPolicy
}

// WithNestedPolicy sets the nested-scope policy.
func WithNestedPolicy(p NestedPolicy) ScopeOption {
	return func(c *scopeConfig) { c.nested = p }
}

// Scope is an explicit batching boundary carried through context. Requests made
// through bound operations within a scope are eligible for coalescing. A Scope
// is safe for concurrent use by the goroutines running inside it.
type Scope struct {
	id     ScopeID
	engine *Engine
	parent context.Context

	mu    sync.Mutex
	state ScopeState
	ops   map[operationController]struct{}
}

// ID returns the scope identity.
func (s *Scope) ID() ScopeID { return s.id }

// State returns the current scope state.
func (s *Scope) State() ScopeState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// checkOpen reports whether the scope still accepts submissions.
func (s *Scope) checkOpen() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != ScopeStateOpen {
		return ErrScopeClosed
	}
	return nil
}

// trackOp records that an operation was used within the scope, for flush, drain,
// and memoization cleanup.
func (s *Scope) trackOp(op operationController) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ops == nil {
		s.ops = make(map[operationController]struct{})
	}
	s.ops[op] = struct{}{}
}

// trackedOps returns a snapshot of the tracked operations.
func (s *Scope) trackedOps() []operationController {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]operationController, 0, len(s.ops))
	for op := range s.ops {
		out = append(out, op)
	}
	return out
}

// Flush dispatches eligible queued work for operations used in this scope.
func (s *Scope) Flush(ctx context.Context) error {
	for _, op := range s.trackedOps() {
		if err := op.flush(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Drain waits for work submitted through this scope's operations to complete.
func (s *Scope) Drain(ctx context.Context) error {
	for _, op := range s.trackedOps() {
		if err := op.drain(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Close rejects new submissions to this scope, flushes eligible work, releases
// scope-local memoization, and detaches from the engine. It is idempotent.
// Because operations invoked through Do complete synchronously, a scope usually
// has no in-flight work of its own at close; Close does not drain shared
// operations used by other scopes.
func (s *Scope) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.state == ScopeStateClosed {
		s.mu.Unlock()
		return nil
	}
	s.state = ScopeStateClosed
	ops := make([]operationController, 0, len(s.ops))
	for op := range s.ops {
		ops = append(ops, op)
	}
	s.ops = nil
	s.mu.Unlock()

	var firstErr error
	for _, op := range ops {
		if err := op.flush(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		op.dropMemo(uint64(s.id))
	}
	s.engine.counters.activeScopes.Add(-1)
	s.engine.emit(Event{Kind: EventScopeClosed, ScopeID: s.id.String()})
	return firstErr
}

// scopeContextKey is the unexported context key for scopes.
type scopeContextKey struct{}

// ContextWithScope returns a context carrying scope. The context key is not
// exported.
func ContextWithScope(ctx context.Context, scope *Scope) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, scope)
}

// ScopeFromContext returns the scope carried by ctx, if any.
func ScopeFromContext(ctx context.Context) (*Scope, bool) {
	s, ok := ctx.Value(scopeContextKey{}).(*Scope)
	return s, ok
}

// RequireScope returns the scope carried by ctx or ErrScopeRequired.
func RequireScope(ctx context.Context) (*Scope, error) {
	if s, ok := ScopeFromContext(ctx); ok {
		return s, nil
	}
	return nil, ErrScopeRequired
}

// NewScope creates a scope and returns a derived context that carries it. The
// caller owns the returned scope and must Close it. Closing a child scope never
// closes its parent.
func (e *Engine) NewScope(parent context.Context, options ...ScopeOption) (context.Context, *Scope, error) {
	if err := e.acceptingSubmissions(); err != nil {
		return parent, nil, err
	}
	cfg := scopeConfig{nested: NestedIsolateChild}
	for _, o := range options {
		o(&cfg)
	}
	if existing, ok := ScopeFromContext(parent); ok {
		switch cfg.nested {
		case NestedReuseParent:
			return parent, existing, nil
		case NestedRejectNested:
			return parent, nil, ErrNestedScope
		}
	}
	s := &Scope{
		id:     ScopeID(e.scopeSeq.Add(1)),
		engine: e,
		parent: parent,
		state:  ScopeStateOpen,
		ops:    make(map[operationController]struct{}),
	}
	e.counters.activeScopes.Add(1)
	e.emit(Event{Kind: EventScopeOpened, ScopeID: s.id.String()})
	return ContextWithScope(parent, s), s, nil
}

// Run creates a scope, runs fn with the scope-carrying context, and closes the
// scope. It is the most convenient entry point for a batching boundary. Because
// Go does not allow methods with their own type parameters, Run is a top-level
// function rather than a method. The engine is the first argument by design, so
// the call reads like a method on the engine.
//
//nolint:revive // engine precedes context intentionally to mirror engine.Run.
func Run[T any](engine *Engine, parent context.Context, fn func(context.Context) (T, error), options ...ScopeOption) (T, error) {
	var zero T
	ctx, scope, err := engine.NewScope(parent, options...)
	if err != nil {
		return zero, err
	}
	defer func() { _ = scope.Close(context.Background()) }()
	return fn(ctx)
}
