package runtime

import (
	"context"
	"sort"
	"sync/atomic"
	"time"

	batchweaver "github.com/Voskan/BatchWeaver"
)

// flushMode is the runtime-normalized scheduling mode.
type flushMode uint8

const (
	flushImmediate flushMode = iota
	flushFixedWindow
	flushDeadlineAware
	flushManual
)

// result is delivered to a waiter. found distinguishes a not-found outcome from
// a successful zero value.
type result[V any] struct {
	value V
	err   error
	found bool
}

// waiter is one logical caller waiting on an item's outcome. The channel is
// buffered so the coordinator never blocks delivering a result.
type waiter[V any] struct {
	id          uint64
	ch          chan result[V]
	active      bool
	deadline    time.Time
	hasDeadline bool
}

// qitem is a unique queued item that may have multiple waiters (duplicates).
type qitem[K, V any] struct {
	id            uint64
	seq           uint64
	key           K
	hash          uint64
	weight        int64
	bytes         int
	enqueue       time.Time
	waiters       []*waiter[V]
	activeWaiters int
	dispatched    bool
	batchID       uint64
}

// earliestDeadline returns the earliest deadline among active waiters.
func (it *qitem[K, V]) earliestDeadline() (time.Time, bool) {
	var best time.Time
	found := false
	for _, w := range it.waiters {
		if w.active && w.hasDeadline {
			if !found || w.deadline.Before(best) {
				best, found = w.deadline, true
			}
		}
	}
	return best, found
}

// latestDeadline returns the latest deadline among active waiters, and whether
// every active waiter has a deadline (so a batch deadline is safe to apply).
func (it *qitem[K, V]) latestDeadline() (time.Time, bool) {
	var best time.Time
	all := true
	any := false
	for _, w := range it.waiters {
		if !w.active {
			continue
		}
		any = true
		if !w.hasDeadline {
			all = false
			continue
		}
		if w.deadline.After(best) {
			best = w.deadline
		}
	}
	return best, any && all
}

// partitionState holds the queue for one partition.
type partitionState[K, V any] struct {
	part    Partition
	pending []*qitem[K, V] // FIFO of undispatched items
	active  map[uint64][]*qitem[K, V]
	bytes   int64
	weight  int64
	waiters int
	live    int // pending + dispatched item count (for partition retirement)
}

// batchInflight tracks a dispatched batch for cancellation accounting.
type batchInflight[K, V any] struct {
	id            uint64
	cancel        context.CancelFunc
	items         []*qitem[K, V]
	activeWaiters int
	partition     string
	minSeq        uint64
}

// submitReq is a submission from a caller goroutine. The caller's deadline is
// carried on the waiter, not here.
type submitReq[K, V any] struct {
	part   Partition
	key    K
	hash   uint64
	weight int64
	bytes  int
	waiter *waiter[V]
	reply  chan error // nil = admitted; sentinel/typed = fallback/reject
}

// cancelReq notifies the coordinator that a caller canceled. The waiter is
// located by ID because the caller does not know the coordinator-assigned item.
type cancelReq struct {
	part     string
	waiterID uint64
	blocked  bool
}

// completeReq is sent by a provider worker when a batch finishes.
type completeReq[V any] struct {
	batchID   uint64
	response  batchweaver.BatchResponse[V]
	globalErr error
	panicked  bool
	panicVal  string
}

// controlReq requests flush/drain/close.
type controlKind uint8

const (
	controlFlush controlKind = iota
	controlDrain
	controlClose
)

type controlReq struct {
	kind  controlKind
	reply chan error
	ctx   context.Context
}

// errFallbackSentinel is returned on the admit reply to instruct the caller to
// run the scalar fallback.
var errFallbackSentinel = &sentinelErr{"fallback"}

type sentinelErr struct{ s string }

func (e *sentinelErr) Error() string { return e.s }

// coordinator owns all mutable state for one bound operation and processes
// events serially on a single goroutine, so the state needs no locks.
type coordinator[K, V any] struct {
	engine *Engine
	cfg    bindingConfig[K, V]

	submitCh   chan *submitReq[K, V]
	cancelCh   chan cancelReq
	completeCh chan *completeReq[V]
	controlCh  chan *controlReq

	partitions map[string]*partitionState[K, V]
	inflight   map[uint64]*batchInflight[K, V]
	blocked    []*submitReq[K, V]

	timer       Timer
	timerActive bool

	activeProvider int
	batchSeq       uint64
	submitSeq      uint64
	closing        bool
	done           chan struct{}

	counters *opCounters
	barriers []*barrier

	// dyn holds optional adaptive settings applied atomically by the adaptive
	// controller. It is written from other goroutines and read only on this
	// coordinator goroutine, so an atomic pointer keeps the hot path lock-free.
	// The runtime always clamps these to the binding's hard configuration limits,
	// so adaptive tuning can never exceed a configured bound.
	dyn atomic.Pointer[dynamicSettings]
}

// run is the coordinator event loop. It exits when the engine context is
// canceled and all provider work has drained.
func (c *coordinator[K, V]) run() {
	defer close(c.done)
	for {
		var timerC <-chan time.Time
		if c.timer != nil && c.timerActive {
			timerC = c.timer.C()
		}
		select {
		case req := <-c.submitCh:
			c.handleSubmit(req)
		case cr := <-c.cancelCh:
			c.handleCancel(cr)
		case done := <-c.completeCh:
			c.handleComplete(done)
		case ctrl := <-c.controlCh:
			c.handleControl(ctrl)
		case <-timerC:
			c.timerActive = false
			c.schedule(false)
		case <-c.engine.ctx.Done():
			c.handleEngineCancel()
			if c.drained() {
				return
			}
		}
		c.schedule(false)
		c.tryCompleteBarriers()
		if c.closing && c.drained() {
			c.failBarriers(nil)
			return
		}
	}
}

// partitionKeys returns partition keys in deterministic sorted order.
func (c *coordinator[K, V]) partitionKeys() []string {
	keys := make([]string, 0, len(c.partitions))
	for k := range c.partitions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// drained reports whether no items or in-flight batches remain.
func (c *coordinator[K, V]) drained() bool {
	if len(c.inflight) > 0 || c.activeProvider > 0 {
		return false
	}
	for _, p := range c.partitions {
		if p.live > 0 {
			return false
		}
	}
	return true
}
