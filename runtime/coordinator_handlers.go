package runtime

// emit sends a runtime event for this operation with a redacted partition token.
func (c *coordinator[K, V]) emit(kind EventKind, part string, batchID uint64, count int, reason FlushReason, errClass string) {
	if c.engine.hooks.OnEvent == nil {
		return
	}
	c.engine.emit(Event{
		Kind: kind, Operation: c.cfg.opID, Partition: part,
		BatchID: batchID, Count: count, FlushReason: reason, ErrorClass: errClass,
	})
}

// handleSubmit enqueues a new request or joins an existing in-flight item.
func (c *coordinator[K, V]) handleSubmit(req *submitReq[K, V]) {
	if c.closing {
		c.counters.queueRejections.Add(1)
		req.reply <- ErrEngineClosing
		return
	}
	partKey := req.part.encoded
	p := c.partitions[partKey]
	if p == nil {
		if c.cfg.limits.MaxPartitions > 0 && len(c.partitions) >= c.cfg.limits.MaxPartitions {
			c.overflow(req, "partitions", true)
			return
		}
		p = &partitionState[K, V]{part: req.part, active: make(map[uint64][]*qitem[K, V])}
		c.partitions[partKey] = p
		c.counters.activePartitions.Store(int64(len(c.partitions)))
	}

	// In-flight deduplication: join an existing item with an equal key.
	if c.cfg.dedupEnabled {
		for _, existing := range p.active[req.hash] {
			if c.cfg.keys.Equal(existing.key, req.key) {
				existing.waiters = append(existing.waiters, req.waiter)
				existing.activeWaiters++
				p.waiters++
				c.counters.queuedWaiters.Add(1)
				c.counters.inFlightJoins.Add(1)
				c.engine.counters.totalCallsSaved.Add(1)
				c.engine.counters.totalRequests.Add(1)
				req.reply <- nil
				c.emit(EventRequestJoinedInFlight, partKey, 0, 0, 0, "")
				return
			}
		}
	}

	// New unique item: enforce queue limits.
	if over, category := c.overLimits(p, req); over {
		c.overflow(req, category, false)
		return
	}

	it := &qitem[K, V]{
		id:            c.engine.nextRequestID(),
		key:           req.key,
		hash:          req.hash,
		weight:        req.weight,
		bytes:         req.bytes,
		enqueue:       c.engine.clock.Now(),
		waiters:       []*waiter[V]{req.waiter},
		activeWaiters: 1,
		seq:           c.nextSeq(),
	}
	p.active[req.hash] = append(p.active[req.hash], it)
	p.pending = append(p.pending, it)
	p.bytes += int64(it.bytes)
	p.weight += it.weight
	p.waiters++
	p.live++
	c.counters.queuedItems.Add(1)
	c.counters.queuedWaiters.Add(1)
	c.engine.counters.totalRequests.Add(1)
	req.reply <- nil
	c.emit(EventRequestSubmitted, partKey, 0, 0, 0, "")
}

// overLimits reports whether adding a new item to p would exceed a limit.
func (c *coordinator[K, V]) overLimits(p *partitionState[K, V], req *submitReq[K, V]) (bool, string) {
	l := c.cfg.limits
	if l.MaxItems > 0 && len(p.pending)+1 > l.MaxItems {
		return true, "items"
	}
	if l.MaxWaiters > 0 && p.waiters+1 > l.MaxWaiters {
		return true, "waiters"
	}
	if l.MaxBytes > 0 && p.bytes+int64(req.bytes) > l.MaxBytes {
		return true, "bytes"
	}
	if l.MaxWeight > 0 && p.weight+req.weight > l.MaxWeight {
		return true, "weight"
	}
	return false, ""
}

// overflow applies the configured overflow policy to a rejected submission.
func (c *coordinator[K, V]) overflow(req *submitReq[K, V], category string, partition bool) {
	switch c.cfg.overflow {
	case OverflowFallback:
		req.reply <- errFallbackSentinel
	case OverflowBlock:
		c.blocked = append(c.blocked, req)
	default: // OverflowReject
		c.counters.queueRejections.Add(1)
		c.emit(EventRequestRejected, req.part.encoded, 0, 0, 0, category)
		if partition {
			req.reply <- &PartitionLimitError{Operation: c.cfg.opID, Limit: c.cfg.limits.MaxPartitions}
		} else {
			req.reply <- &QueueFullError{Operation: c.cfg.opID, Limit: category}
		}
	}
}

// retryBlocked re-processes parked block-policy submissions after capacity frees.
func (c *coordinator[K, V]) retryBlocked() {
	if len(c.blocked) == 0 {
		return
	}
	pending := c.blocked
	c.blocked = nil
	for _, req := range pending {
		c.handleSubmit(req)
	}
}

// handleCancel processes a caller cancellation.
func (c *coordinator[K, V]) handleCancel(cr cancelReq) {
	if cr.blocked {
		for i, req := range c.blocked {
			if req.waiter.id == cr.waiterID {
				c.blocked = append(c.blocked[:i], c.blocked[i+1:]...)
				break
			}
		}
		return
	}
	p := c.partitions[cr.part]
	if p == nil {
		return
	}
	for _, items := range p.active {
		for _, it := range items {
			for _, w := range it.waiters {
				if w.id != cr.waiterID || !w.active {
					continue
				}
				w.active = false
				it.activeWaiters--
				p.waiters--
				c.counters.queuedWaiters.Add(-1)
				c.engine.counters.totalCancellations.Add(1)
				c.emit(EventRequestCanceled, cr.part, 0, 0, 0, "")
				if it.activeWaiters == 0 {
					if !it.dispatched {
						c.removePendingItem(p, it)
					} else if b := c.inflight[it.batchID]; b != nil {
						b.activeWaiters--
						if b.activeWaiters == 0 {
							b.cancel()
						}
					}
				}
				return
			}
		}
	}
}

// removePendingItem removes an undispatched item with no active waiters.
func (c *coordinator[K, V]) removePendingItem(p *partitionState[K, V], it *qitem[K, V]) {
	for i, x := range p.pending {
		if x == it {
			p.pending = append(p.pending[:i], p.pending[i+1:]...)
			break
		}
	}
	c.removeActive(p, it)
	p.bytes -= int64(it.bytes)
	p.weight -= it.weight
	p.live--
	c.counters.queuedItems.Add(-1)
	c.maybeRetire(p)
}

// removeActive removes an item from the dedup index.
func (c *coordinator[K, V]) removeActive(p *partitionState[K, V], it *qitem[K, V]) {
	bucket := p.active[it.hash]
	for i, x := range bucket {
		if x == it {
			bucket = append(bucket[:i], bucket[i+1:]...)
			break
		}
	}
	if len(bucket) == 0 {
		delete(p.active, it.hash)
	} else {
		p.active[it.hash] = bucket
	}
}

// maybeRetire removes a partition with no queued, in-flight, or memoized work.
func (c *coordinator[K, V]) maybeRetire(p *partitionState[K, V]) {
	if p.live == 0 && len(p.active) == 0 {
		delete(c.partitions, p.part.encoded)
		c.counters.activePartitions.Store(int64(len(c.partitions)))
	}
}

// nextSeq returns a monotonically increasing submission sequence.
func (c *coordinator[K, V]) nextSeq() uint64 {
	c.submitSeq++
	return c.submitSeq
}

// deliver sends a result to an active waiter and marks it delivered.
func (c *coordinator[K, V]) deliver(w *waiter[V], r result[V]) {
	if !w.active {
		return
	}
	w.active = false
	w.ch <- r
	c.counters.queuedWaiters.Add(-1)
}
