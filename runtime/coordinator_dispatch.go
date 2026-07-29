package runtime

import (
	"context"
	"time"

	batchweaver "github.com/Voskan/BatchWeaver"
)

// barrier is a pending flush, drain, or close completion barrier.
type barrier struct {
	kind         controlKind
	seq          uint64
	reply        chan error
	ctx          context.Context
	dispatchOnly bool
}

// handleControl records a barrier and forces dispatch so it can make progress.
func (c *coordinator[K, V]) handleControl(ctrl *controlReq) {
	if ctrl.kind == controlClose {
		c.closing = true
	}
	c.schedule(true)
	c.barriers = append(c.barriers, &barrier{
		kind:         ctrl.kind,
		seq:          c.submitSeq,
		reply:        ctrl.reply,
		ctx:          ctrl.ctx,
		dispatchOnly: ctrl.kind == controlFlush,
	})
}

// tryCompleteBarriers replies to barriers that are satisfied or whose context
// has expired.
func (c *coordinator[K, V]) tryCompleteBarriers() {
	kept := c.barriers[:0]
	for _, b := range c.barriers {
		if b.ctx != nil && b.ctx.Err() != nil {
			b.reply <- barrierTimeout(b.kind, b.ctx.Err())
			continue
		}
		if c.barrierSatisfied(b) {
			b.reply <- nil
			continue
		}
		kept = append(kept, b)
	}
	c.barriers = kept
}

// failBarriers replies to all barriers, used on shutdown.
func (c *coordinator[K, V]) failBarriers(err error) {
	for _, b := range c.barriers {
		b.reply <- err
	}
	c.barriers = nil
}

// barrierSatisfied reports whether a barrier's completion condition holds.
func (c *coordinator[K, V]) barrierSatisfied(b *barrier) bool {
	if c.anyPending(b.seq) {
		return false
	}
	if b.dispatchOnly {
		return true
	}
	return !c.anyInflight(b.seq)
}

// anyPending reports whether any undispatched item has seq <= seq.
func (c *coordinator[K, V]) anyPending(seq uint64) bool {
	for _, p := range c.partitions {
		for _, it := range p.pending {
			if it.seq <= seq {
				return true
			}
		}
	}
	return false
}

// anyInflight reports whether any in-flight batch contains an item with seq <= seq.
func (c *coordinator[K, V]) anyInflight(seq uint64) bool {
	for _, b := range c.inflight {
		if b.minSeq <= seq {
			return true
		}
	}
	return false
}

// barrierTimeout wraps a context error in the appropriate typed error.
func barrierTimeout(kind controlKind, cause error) error {
	if kind == controlFlush {
		return &wrappedErr{ErrFlushTimeout, cause}
	}
	return &wrappedErr{ErrDrainTimeout, cause}
}

// wrappedErr joins a sentinel and a cause for errors.Is on both.
type wrappedErr struct {
	sentinel error
	cause    error
}

func (e *wrappedErr) Error() string   { return e.sentinel.Error() + ": " + e.cause.Error() }
func (e *wrappedErr) Unwrap() []error { return []error{e.sentinel, e.cause} }

// schedule dispatches eligible batches. When force is true, all pending items
// are eligible regardless of the time-based policy.
func (c *coordinator[K, V]) schedule(force bool) {
	now := c.engine.clock.Now()
	for _, pk := range c.partitionKeys() {
		p := c.partitions[pk]
		if p == nil {
			continue
		}
		for len(p.pending) > 0 && c.activeProvider < c.cfg.maxConcurrency {
			c.prune(p)
			if len(p.pending) == 0 {
				break
			}
			reason, ok := c.shouldDispatch(p, force, now)
			if !ok {
				break
			}
			c.dispatchBatch(p, reason)
		}
		c.maybeRetire(p)
	}
	c.computeNextTimer(now)
}

// prune removes leading pending items whose waiters have all canceled.
func (c *coordinator[K, V]) prune(p *partitionState[K, V]) {
	for len(p.pending) > 0 && p.pending[0].activeWaiters == 0 {
		c.removePendingItem(p, p.pending[0])
	}
}

// shouldDispatch decides whether and why to dispatch from a partition.
func (c *coordinator[K, V]) shouldDispatch(p *partitionState[K, V], force bool, now time.Time) (FlushReason, bool) {
	n := len(p.pending)
	if n == 0 {
		return 0, false
	}
	if force {
		return FlushManual, true
	}
	if n >= c.cfg.maxBatchSize {
		return FlushSize, true
	}
	if c.cfg.maxBatchWeight > 0 && p.weight >= c.cfg.maxBatchWeight {
		return FlushWeight, true
	}
	if c.cfg.maxBatchBytes > 0 && p.bytes >= c.cfg.maxBatchBytes {
		return FlushBytes, true
	}
	switch c.cfg.flushMode {
	case flushImmediate:
		return FlushWait, true
	case flushManual:
		return 0, false
	case flushFixedWindow:
		if c.cfg.maxWait > 0 && !now.Before(p.pending[0].enqueue.Add(c.cfg.maxWait)) {
			return FlushWait, true
		}
	case flushDeadlineAware:
		if c.cfg.maxWait > 0 && !now.Before(p.pending[0].enqueue.Add(c.cfg.maxWait)) {
			return FlushWait, true
		}
		if t, ok := c.earliestDeadline(p); ok && !now.Before(t.Add(-c.cfg.deadlineMargin)) {
			return FlushDeadline, true
		}
	}
	return 0, false
}

// earliestDeadline returns the earliest active caller deadline among pending items.
func (c *coordinator[K, V]) earliestDeadline(p *partitionState[K, V]) (time.Time, bool) {
	var best time.Time
	found := false
	for _, it := range p.pending {
		if t, ok := it.earliestDeadline(); ok {
			if !found || t.Before(best) {
				best, found = t, true
			}
		}
	}
	return best, found
}

// dispatchBatch selects items from a partition and starts a provider call.
func (c *coordinator[K, V]) dispatchBatch(p *partitionState[K, V], reason FlushReason) {
	var selected []*qitem[K, V]
	var items []batchweaver.BatchItem[K]
	var count int
	var weight int64
	var bytes int64
	minSeq := ^uint64(0)

	remaining := p.pending[:0]
	for _, it := range p.pending {
		if it.activeWaiters == 0 {
			c.removeActive(p, it)
			p.live--
			c.counters.queuedItems.Add(-1)
			continue
		}
		// Oversized single item: fail deterministically.
		if len(selected) == 0 && c.itemTooLarge(it) {
			c.failItem(p, it, &wrappedErr{ErrItemTooLarge, nil})
			c.counters.queuedItems.Add(-1)
			continue
		}
		if len(selected) > 0 && !c.fits(count, weight, bytes, it) {
			remaining = append(remaining, it)
			continue
		}
		selected = append(selected, it)
		bi := batchweaver.NewBatchItem[K](batchweaver.RequestID(it.id), it.key).WithWeight(clampWeight(it.weight))
		if t, ok := it.earliestDeadline(); ok {
			bi = bi.WithDeadline(t)
		}
		items = append(items, bi)
		count++
		weight += it.weight
		bytes += int64(it.bytes)
		if it.seq < minSeq {
			minSeq = it.seq
		}
	}
	p.pending = remaining
	if len(selected) == 0 {
		return
	}

	req, err := batchweaver.NewBatchRequest(items)
	if err != nil {
		for _, it := range selected {
			c.failItem(p, it, &ContractError{Operation: c.cfg.opID, Reason: err.Error()})
			c.counters.queuedItems.Add(-1)
		}
		return
	}

	// Adjust pending accounting for the removed items.
	p.bytes -= bytes
	p.weight -= weight
	c.counters.queuedItems.Add(int64(-len(selected)))

	batchID := c.nextBatchID()
	batchCtx, cancel := c.batchContext(selected)
	batchCtx = markExec(batchCtx, c.cfg.opID, p.part.encoded)
	inflightWaiters := 0
	for _, it := range selected {
		it.dispatched = true
		it.batchID = batchID
		inflightWaiters += it.activeWaiters
	}
	c.inflight[batchID] = &batchInflight[K, V]{
		id: batchID, cancel: cancel, items: selected,
		activeWaiters: inflightWaiters, partition: p.part.encoded, minSeq: minSeq,
	}
	c.activeProvider++
	c.counters.activeProviderCalls.Store(int64(c.activeProvider))
	c.engine.counters.totalProviderCalls.Add(1)
	c.engine.counters.totalBackendItems.Add(uint64(len(selected)))
	c.counters.recordBatch(len(selected), reason)
	c.emit(EventBatchSelected, p.part.encoded, batchID, len(selected), reason, "")
	c.emit(EventBatchDispatched, p.part.encoded, batchID, len(selected), reason, "")

	c.startWorker(batchCtx, batchID, req)
}

// fits reports whether adding it keeps the batch within hard limits.
func (c *coordinator[K, V]) fits(count int, weight, bytes int64, it *qitem[K, V]) bool {
	if c.cfg.maxBatchSize > 0 && count+1 > c.cfg.maxBatchSize {
		return false
	}
	if c.cfg.maxBatchWeight > 0 && weight+it.weight > c.cfg.maxBatchWeight {
		return false
	}
	if c.cfg.maxBatchBytes > 0 && bytes+int64(it.bytes) > c.cfg.maxBatchBytes {
		return false
	}
	return true
}

// itemTooLarge reports whether a single item exceeds a hard batch limit.
func (c *coordinator[K, V]) itemTooLarge(it *qitem[K, V]) bool {
	if c.cfg.maxBatchWeight > 0 && it.weight > c.cfg.maxBatchWeight {
		return true
	}
	if c.cfg.maxBatchBytes > 0 && int64(it.bytes) > c.cfg.maxBatchBytes {
		return true
	}
	return false
}

// failItem delivers err to all active waiters of an item and removes it.
func (c *coordinator[K, V]) failItem(p *partitionState[K, V], it *qitem[K, V], err error) {
	for _, w := range it.waiters {
		c.deliver(w, result[V]{err: err})
	}
	c.removeActive(p, it)
	p.live--
}

// batchContext builds the dedicated provider context. It is canceled on engine
// shutdown and when every active waiter cancels; it carries the latest active
// caller deadline only when every active waiter has one, so a short caller
// cannot cancel longer callers.
func (c *coordinator[K, V]) batchContext(selected []*qitem[K, V]) (context.Context, context.CancelFunc) {
	var latest time.Time
	allDeadlined := true
	any := false
	for _, it := range selected {
		t, ok := it.latestDeadline()
		if !ok {
			allDeadlined = false
			continue
		}
		any = true
		if t.After(latest) {
			latest = t
		}
	}
	if any && allDeadlined {
		return context.WithDeadline(c.engine.ctx, latest)
	}
	return context.WithCancel(c.engine.ctx)
}

// startWorker runs the provider on a bounded worker goroutine.
func (c *coordinator[K, V]) startWorker(ctx context.Context, batchID uint64, req batchweaver.BatchRequest[K]) {
	c.engine.wg.Add(1)
	go func() {
		defer c.engine.wg.Done()
		resp, panicked, pv, gerr := c.callProvider(ctx, req)
		select {
		case c.completeCh <- &completeReq[V]{batchID: batchID, response: resp, globalErr: gerr, panicked: panicked, panicVal: pv}:
		case <-c.done:
		}
	}()
}

// callProvider invokes the provider, recovering panics per the engine policy.
func (c *coordinator[K, V]) callProvider(ctx context.Context, req batchweaver.BatchRequest[K]) (resp batchweaver.BatchResponse[V], panicked bool, pv string, err error) {
	defer func() {
		if r := recover(); r != nil {
			if c.engine.panicPolicy == PanicPolicyRepanic {
				panic(r)
			}
			panicked = true
			pv = sanitizePanic(r)
		}
	}()
	resp, err = c.cfg.provider.Execute(ctx, req)
	return
}

// handleComplete distributes a finished batch's outcomes to waiters.
func (c *coordinator[K, V]) handleComplete(done *completeReq[V]) {
	b := c.inflight[done.batchID]
	if b == nil {
		return
	}
	delete(c.inflight, done.batchID)
	c.activeProvider--
	c.counters.activeProviderCalls.Store(int64(c.activeProvider))
	b.cancel()

	p := c.partitions[b.partition]

	switch {
	case done.panicked:
		err := &ProviderPanicError{Operation: c.cfg.opID, Value: done.panicVal}
		c.failBatch(p, b, err)
		c.counters.providerErrors.Add(1)
		c.emit(EventBatchFailed, b.partition, b.id, len(b.items), 0, "panic")
	case done.globalErr != nil:
		c.failBatch(p, b, done.globalErr)
		c.counters.providerErrors.Add(1)
		c.emit(EventBatchFailed, b.partition, b.id, len(b.items), 0, "provider")
	default:
		c.distribute(p, b, done.response)
	}
	c.retryBlocked()
}

// failBatch delivers err to every active waiter of every item in the batch.
func (c *coordinator[K, V]) failBatch(p *partitionState[K, V], b *batchInflight[K, V], err error) {
	for _, it := range b.items {
		for _, w := range it.waiters {
			c.deliver(w, result[V]{err: err})
		}
		if p != nil {
			c.removeActive(p, it)
			p.live--
		}
	}
	if p != nil {
		c.maybeRetire(p)
	}
}

// distribute validates the response and maps outcomes to waiters.
func (c *coordinator[K, V]) distribute(p *partitionState[K, V], b *batchInflight[K, V], resp batchweaver.BatchResponse[V]) {
	ids := make([]batchweaver.RequestID, len(b.items))
	for i, it := range b.items {
		ids[i] = batchweaver.RequestID(it.id)
	}
	if err := resp.ValidateAgainst(ids); err != nil {
		cerr := &ContractError{Operation: c.cfg.opID, Reason: err.Error()}
		c.failBatch(p, b, cerr)
		c.counters.providerErrors.Add(1)
		c.engine.counters.totalContractViolations.Add(1)
		c.emit(EventContractViolation, b.partition, b.id, len(b.items), 0, "contract")
		return
	}
	byID := make(map[batchweaver.RequestID]batchweaver.Outcome[V], resp.Len())
	for _, o := range resp.Outcomes() {
		byID[o.RequestID] = o
	}
	for _, it := range b.items {
		o := byID[batchweaver.RequestID(it.id)]
		var r result[V]
		switch {
		case o.IsSuccess():
			r = result[V]{value: o.Value, found: true}
		case o.IsFailure():
			r = result[V]{err: o.Err}
			c.counters.itemErrors.Add(1)
		default: // not found
			r = result[V]{found: false}
		}
		for _, w := range it.waiters {
			c.deliver(w, r)
		}
		if p != nil {
			c.removeActive(p, it)
			p.live--
		}
	}
	if p != nil {
		c.maybeRetire(p)
	}
	c.emit(EventBatchCompleted, b.partition, b.id, len(b.items), 0, "")
}

// handleEngineCancel delivers ErrEngineClosed to all remaining active waiters and
// cancels in-flight provider contexts, so no caller blocks forever after an
// abrupt engine shutdown.
func (c *coordinator[K, V]) handleEngineCancel() {
	c.closing = true
	for _, p := range c.partitions {
		for _, items := range p.active {
			for _, it := range items {
				for _, w := range it.waiters {
					c.deliver(w, result[V]{err: ErrEngineClosed})
				}
			}
		}
	}
	for _, b := range c.inflight {
		b.cancel()
	}
	c.failBarriers(ErrEngineClosed)
}

// computeNextTimer sets a single timer for the earliest future time-based flush.
func (c *coordinator[K, V]) computeNextTimer(now time.Time) {
	var next time.Time
	has := false
	consider := func(t time.Time) {
		if !has || t.Before(next) {
			next, has = t, true
		}
	}
	for _, p := range c.partitions {
		if len(p.pending) == 0 {
			continue
		}
		switch c.cfg.flushMode {
		case flushFixedWindow:
			if c.cfg.maxWait > 0 {
				consider(p.pending[0].enqueue.Add(c.cfg.maxWait))
			}
		case flushDeadlineAware:
			if c.cfg.maxWait > 0 {
				consider(p.pending[0].enqueue.Add(c.cfg.maxWait))
			}
			if t, ok := c.earliestDeadline(p); ok {
				consider(t.Add(-c.cfg.deadlineMargin))
			}
		}
	}
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.timerActive = false
	if has {
		d := next.Sub(now)
		if d < 0 {
			d = 0
		}
		c.timer = c.engine.clock.NewTimer(d)
		c.timerActive = true
	}
}

// nextBatchID returns a unique batch ID for this operation.
func (c *coordinator[K, V]) nextBatchID() uint64 {
	c.batchSeq++
	return c.batchSeq
}

// clampWeight converts an item weight to the batch item weight, ensuring it is
// at least one.
func clampWeight(w int64) int {
	if w < 1 {
		return 1
	}
	if w > int64(^uint(0)>>1) {
		return int(^uint(0) >> 1)
	}
	return int(w)
}
