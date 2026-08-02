package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Voskan/BatchWeaver/operation"
)

// WeightFunc returns the weight of a key for batch-weight accounting. Weights
// must be non-negative; the default weight is one.
type WeightFunc[K any] func(K) int64

// Declaration is any operation declaration or spec source. Both
// FunctionDeclaration and MethodDeclaration satisfy it.
type Declaration interface {
	Spec() operation.Spec
}

// Binding configures a typed operation binding. It is passed to Bind. Using a
// struct (rather than variadic options) keeps generic type inference clean while
// K and V are unconstrained.
type Binding[K any, V any] struct {
	// Provider executes batches; required.
	Provider Provider[K, V]
	// Keys is the key strategy; required.
	Keys KeyStrategy[K]
	// Partitioner assigns partitions; defaults to a single partition.
	Partitioner Partitioner[K]
	// Weight computes item weight; defaults to one per item.
	Weight WeightFunc[K]
	// Fallback runs single requests directly; required for the fallback overflow
	// policy and for scope-less execution when RequireScope is false.
	Fallback ScalarFallback[K, V]
	// Limits overrides the engine default queue limits for this operation.
	Limits QueueLimits
	// Overflow overrides the overflow policy for this operation.
	Overflow *OverflowPolicy
	// RequireScope forces scope-less calls to return ErrScopeRequired even when a
	// fallback is configured.
	RequireScope bool
	// EnableMemoization opts into scope-local memoization, subject to operation
	// semantics.
	EnableMemoization bool
}

// bindingConfig is the immutable normalized configuration of a bound operation.
type bindingConfig[K any, V any] struct {
	opID           string
	spec           operation.Spec
	provider       Provider[K, V]
	keys           KeyStrategy[K]
	partitioner    Partitioner[K]
	weight         WeightFunc[K]
	fallback       ScalarFallback[K, V]
	limits         QueueLimits
	overflow       OverflowPolicy
	flushMode      flushMode
	maxBatchSize   int
	maxBatchWeight int64
	maxBatchBytes  int64
	maxWait        time.Duration
	deadlineMargin time.Duration
	maxConcurrency int
	dedupEnabled   bool
	memoEnabled    bool
	memoNegative   bool
	memoMaxItems   int
	memoMaxBytes   int64
	requireScope   bool
}

// BoundOperation is a typed handle for invoking a bound operation. It is safe
// for concurrent use.
type BoundOperation[K any, V any] struct {
	coord  *coordinator[K, V]
	engine *Engine

	memoMu sync.Mutex
	memo   map[uint64]*memoTable[K, V]
}

// Bind binds an operation declaration and provider to an engine, returning a
// typed handle. It validates the configuration against the operation semantics
// and rejects duplicate operation IDs.
func Bind[K any, V any](engine *Engine, decl Declaration, b Binding[K, V]) (*BoundOperation[K, V], error) {
	if err := engine.ensureOpen(); err != nil {
		return nil, err
	}
	spec := decl.Spec()
	if err := spec.ValidationError(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBinding, err)
	}
	if b.Provider == nil {
		return nil, fmt.Errorf("%w: provider is nil", ErrInvalidBinding)
	}
	if b.Keys == nil {
		return nil, fmt.Errorf("%w: key strategy is nil", ErrInvalidBinding)
	}
	if err := b.Limits.Validate(); err != nil {
		return nil, err
	}

	sp := spec.SchedulerPolicy()
	sem := spec.Semantics()

	cfg := bindingConfig[K, V]{
		opID:           spec.ID().String(),
		spec:           spec,
		provider:       b.Provider,
		keys:           b.Keys,
		partitioner:    b.Partitioner,
		weight:         b.Weight,
		fallback:       b.Fallback,
		maxBatchSize:   sp.MaxBatchSize(),
		maxBatchWeight: int64(sp.MaxBatchWeight()),
		maxBatchBytes:  sp.MaxPayloadBytes(),
		maxWait:        sp.MaxWait(),
		deadlineMargin: sp.DeadlineMargin(),
		maxConcurrency: sp.MaxConcurrency(),
		flushMode:      normalizeFlushMode(sp.Mode()),
		requireScope:   b.RequireScope,
	}
	if cfg.partitioner == nil {
		cfg.partitioner = singlePartitioner[K]{}
	}
	if cfg.maxConcurrency < 1 {
		cfg.maxConcurrency = 1
	}
	if cfg.maxBatchSize < 1 {
		cfg.maxBatchSize = 1
	}

	specLimits := QueueLimits{
		MaxItems:      sp.QueueItems(),
		MaxBytes:      sp.QueueBytes(),
		MaxWeight:     int64(sp.MaxBatchWeight()),
		MaxPartitions: sp.ActivePartitions(),
	}
	cfg.limits = b.Limits.merge(specLimits).merge(engine.defaultLimits)

	cfg.overflow = engine.defaultOverflow
	switch sp.Overflow() {
	case operation.OverflowReject:
		cfg.overflow = OverflowReject
	case operation.OverflowBlock:
		cfg.overflow = OverflowBlock
	case operation.OverflowFallback:
		cfg.overflow = OverflowFallback
	}
	if b.Overflow != nil {
		cfg.overflow = *b.Overflow
	}

	if spec.DeduplicationPolicy().InFlight() {
		if !sem.DeduplicationAllowed() {
			return nil, fmt.Errorf("%w: in-flight deduplication is not permitted for %s operations", ErrInvalidBinding, sem.Kind())
		}
		cfg.dedupEnabled = true
	}

	if spec.DeduplicationPolicy().ScopeMemoization() || b.EnableMemoization {
		if sem.Effect() != operation.EffectRead || sem.FreshnessDependent() {
			return nil, fmt.Errorf("%w: memoization is not permitted for %s operations", ErrInvalidBinding, sem.Kind())
		}
		cfg.memoEnabled = true
		cfg.memoNegative = spec.DeduplicationPolicy().NegativeMemoization()
		cfg.memoMaxItems = spec.DeduplicationPolicy().MaxItems()
		cfg.memoMaxBytes = spec.DeduplicationPolicy().MaxBytes()
		if cfg.memoMaxItems < 1 {
			cfg.memoMaxItems = 1024
		}
	}

	if cfg.overflow == OverflowFallback && cfg.fallback == nil {
		return nil, fmt.Errorf("%w: fallback overflow policy requires a scalar fallback", ErrInvalidBinding)
	}

	coord := &coordinator[K, V]{
		engine:     engine,
		cfg:        cfg,
		submitCh:   make(chan *submitReq[K, V]),
		cancelCh:   make(chan cancelReq, 64),
		completeCh: make(chan *completeReq[V], 64),
		controlCh:  make(chan *controlReq),
		partitions: make(map[string]*partitionState[K, V]),
		inflight:   make(map[uint64]*batchInflight[K, V]),
		done:       make(chan struct{}),
		counters:   &opCounters{},
	}
	o := &BoundOperation[K, V]{coord: coord, engine: engine, memo: make(map[uint64]*memoTable[K, V])}
	if err := engine.registerController(o); err != nil {
		return nil, err
	}
	engine.wg.Add(1)
	go func() {
		defer engine.wg.Done()
		coord.run()
	}()
	return o, nil
}

// normalizeFlushMode maps an operation scheduler mode to the runtime flush mode.
// Adaptive, throughput, and latency modes are treated as a fixed window because
// adaptive learning is not implemented in this release.
func normalizeFlushMode(m operation.SchedulerMode) flushMode {
	switch m {
	case operation.SchedulerImmediateWave:
		return flushImmediate
	case operation.SchedulerManual:
		return flushManual
	case operation.SchedulerDeadlineAware:
		return flushDeadlineAware
	default:
		return flushFixedWindow
	}
}

// Do submits a single request and returns its result. Compatible calls made
// within the same scope are coalesced according to the operation's scheduling
// policy. Do respects the caller's context for cancellation and deadline.
func (o *BoundOperation[K, V]) Do(ctx context.Context, key K) (V, error) {
	var zero V
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	c := o.coord

	part, err := c.callPartitioner(ctx, key)
	if err != nil {
		return zero, err
	}

	scope, hasScope := ScopeFromContext(ctx)
	if !hasScope {
		if c.cfg.fallback != nil && !c.cfg.requireScope {
			return c.runFallback(ctx, key)
		}
		return zero, ErrScopeRequired
	}
	if err := scope.checkOpen(); err != nil {
		return zero, err
	}
	if err := o.engine.acceptingSubmissions(); err != nil {
		if c.cfg.fallback != nil && !c.cfg.requireScope {
			return c.runFallback(ctx, key)
		}
		return zero, err
	}

	keyClone, err := c.callClone(key)
	if err != nil {
		return zero, err
	}

	if c.cfg.memoEnabled {
		if r, ok := o.memoTableFor(uint64(scope.ID())).get(part, keyClone); ok {
			c.counters.memoHits.Add(1)
			o.engine.counters.totalCallsSaved.Add(1)
			o.engine.emit(Event{Kind: EventMemoHit, Operation: c.cfg.opID, Partition: part.encoded, ScopeID: scope.ID().String()})
			return o.finishMemo(r)
		}
	}

	if executing(ctx, c.cfg.opID, part.encoded) {
		if c.cfg.fallback != nil {
			return c.runFallback(ctx, key)
		}
		return zero, ErrRecursiveOperation
	}

	hash, err := c.callHash(keyClone)
	if err != nil {
		return zero, err
	}
	weight, err := c.callWeight(keyClone)
	if err != nil {
		return zero, err
	}
	bytes, err := c.callBytes(keyClone)
	if err != nil {
		return zero, err
	}

	w := &waiter[V]{id: o.engine.nextRequestID(), ch: make(chan result[V], 1), active: true}
	if dl, ok := ctx.Deadline(); ok {
		w.deadline, w.hasDeadline = dl, true
	}
	scope.trackOp(o)

	req := &submitReq[K, V]{
		part: part, key: keyClone, hash: hash, weight: weight, bytes: bytes,
		waiter: w, reply: make(chan error, 1),
	}
	select {
	case c.submitCh <- req:
	case <-ctx.Done():
		return zero, ctx.Err()
	case <-c.done:
		return zero, ErrEngineClosed
	}

	var admit error
	select {
	case admit = <-req.reply:
	case <-ctx.Done():
		select {
		case admit = <-req.reply:
		default:
			c.sendCancel(cancelReq{blocked: true, waiterID: w.id})
			return zero, ctx.Err()
		}
	case <-c.done:
		return zero, ErrEngineClosed
	}
	if admit != nil {
		if errors.Is(admit, errFallbackSentinel) {
			return c.runFallback(ctx, key)
		}
		return zero, admit
	}

	select {
	case r := <-w.ch:
		return o.finish(scope, part, keyClone, r)
	case <-ctx.Done():
		select {
		case r := <-w.ch:
			return o.finish(scope, part, keyClone, r)
		default:
			c.sendCancel(cancelReq{part: part.encoded, waiterID: w.id})
			return zero, ctx.Err()
		}
	case <-c.done:
		return zero, ErrEngineClosed
	}
}

// finish converts a delivered result into the scalar return, applying memoization.
func (o *BoundOperation[K, V]) finish(scope *Scope, part Partition, key K, r result[V]) (V, error) {
	var zero V
	switch {
	case r.err != nil:
		return zero, r.err
	case r.found:
		if o.coord.cfg.memoEnabled {
			o.memoTableFor(uint64(scope.ID())).put(part, key, r.value, true, o.coord.cfg)
		}
		return r.value, nil
	default:
		if o.coord.cfg.memoEnabled && o.coord.cfg.memoNegative {
			o.memoTableFor(uint64(scope.ID())).put(part, key, zero, false, o.coord.cfg)
		}
		return zero, ErrMissingResult
	}
}

// finishMemo converts a memoized record into the scalar return.
func (o *BoundOperation[K, V]) finishMemo(r result[V]) (V, error) {
	var zero V
	if r.found {
		return r.value, nil
	}
	return zero, ErrMissingResult
}

// sendCancel delivers a cancellation to the coordinator without blocking on it.
func (c *coordinator[K, V]) sendCancel(cr cancelReq) {
	select {
	case c.cancelCh <- cr:
	case <-c.done:
	}
}

// --- operationController implementation ---

func (o *BoundOperation[K, V]) id() string { return o.coord.cfg.opID }

func (o *BoundOperation[K, V]) flush(ctx context.Context) error {
	return o.control(ctx, controlFlush)
}
func (o *BoundOperation[K, V]) drain(ctx context.Context) error {
	return o.control(ctx, controlDrain)
}
func (o *BoundOperation[K, V]) closeController(ctx context.Context) error {
	return o.control(ctx, controlClose)
}

func (o *BoundOperation[K, V]) control(ctx context.Context, kind controlKind) error {
	reply := make(chan error, 1)
	req := &controlReq{kind: kind, reply: reply, ctx: ctx}
	select {
	case o.coord.controlCh <- req:
	case <-o.coord.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-reply:
		return err
	case <-o.coord.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (o *BoundOperation[K, V]) stats() OperationStats {
	return o.coord.counters.snapshot(o.coord.cfg.opID)
}

func (o *BoundOperation[K, V]) dropMemo(scopeID uint64) {
	o.memoMu.Lock()
	delete(o.memo, scopeID)
	o.memoMu.Unlock()
}

// memoTableFor returns the scope-local memo table, creating it on demand.
func (o *BoundOperation[K, V]) memoTableFor(scopeID uint64) *memoTable[K, V] {
	o.memoMu.Lock()
	defer o.memoMu.Unlock()
	t := o.memo[scopeID]
	if t == nil {
		t = &memoTable[K, V]{keys: o.coord.cfg.keys, byHash: make(map[uint64][]*memoRecord[K, V])}
		o.memo[scopeID] = t
	}
	return t
}

// runFallback executes the scalar fallback with the original caller context.
func (c *coordinator[K, V]) runFallback(ctx context.Context, key K) (V, error) {
	var zero V
	if c.cfg.fallback == nil {
		return zero, ErrScopeRequired
	}
	c.counters.fallbacks.Add(1)
	c.engine.counters.totalFallbacks.Add(1)
	c.engine.emit(Event{Kind: EventFallbackStarted, Operation: c.cfg.opID})
	v, err := c.cfg.fallback.Execute(ctx, key)
	c.engine.emit(Event{Kind: EventFallbackCompleted, Operation: c.cfg.opID})
	return v, err
}

// --- callback wrappers with panic isolation ---

func (c *coordinator[K, V]) callPartitioner(ctx context.Context, key K) (p Partition, err error) {
	defer c.recoverCallback("partitioner", &err)
	return c.cfg.partitioner.Partition(ctx, key)
}

func (c *coordinator[K, V]) callClone(key K) (k K, err error) {
	defer c.recoverCallback("key clone", &err)
	return c.cfg.keys.Clone(key), nil
}

func (c *coordinator[K, V]) callHash(key K) (h uint64, err error) {
	defer c.recoverCallback("key hash", &err)
	return c.cfg.keys.Hash(key), nil
}

func (c *coordinator[K, V]) callWeight(key K) (w int64, err error) {
	defer c.recoverCallback("weight", &err)
	if c.cfg.weight == nil {
		return 1, nil
	}
	w = c.cfg.weight(key)
	if w < 0 {
		return 0, fmt.Errorf("%w: weight callback returned a negative value", ErrInvalidBinding)
	}
	return w, nil
}

func (c *coordinator[K, V]) callBytes(key K) (n int, err error) {
	defer c.recoverCallback("key size", &err)
	n = c.cfg.keys.EstimateBytes(key)
	if n < 0 {
		n = 0
	}
	return n, nil
}

// recoverCallback converts a panic in a request-time callback into a typed error.
func (c *coordinator[K, V]) recoverCallback(name string, err *error) {
	if r := recover(); r != nil {
		if c.engine.panicPolicy == PanicPolicyRepanic {
			panic(r)
		}
		*err = &CallbackPanicError{Operation: c.cfg.opID, Callback: name, Value: sanitizePanic(r)}
	}
}

// --- recursion detection ---

type execKey struct{}

type execFrame struct {
	opID   string
	part   string
	parent *execFrame
}

// markExec returns a context marked as executing opID within part.
func markExec(ctx context.Context, opID, part string) context.Context {
	parent, _ := ctx.Value(execKey{}).(*execFrame)
	return context.WithValue(ctx, execKey{}, &execFrame{opID: opID, part: part, parent: parent})
}

// executing reports whether ctx is already executing opID within part.
func executing(ctx context.Context, opID, part string) bool {
	f, _ := ctx.Value(execKey{}).(*execFrame)
	for f != nil {
		if f.opID == opID && f.part == part {
			return true
		}
		f = f.parent
	}
	return false
}
