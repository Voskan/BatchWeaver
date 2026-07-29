package runtime

import "sync/atomic"

// engineCounters holds engine-wide cumulative counters as atomics so snapshots
// are race-safe without stopping the world.
type engineCounters struct {
	totalRequests           atomic.Uint64
	totalProviderCalls      atomic.Uint64
	totalBackendItems       atomic.Uint64
	totalCallsSaved         atomic.Uint64
	totalFallbacks          atomic.Uint64
	totalCancellations      atomic.Uint64
	totalContractViolations atomic.Uint64
	activeScopes            atomic.Int64
}

// opCounters holds per-operation counters and gauges as atomics, updated by the
// operation coordinator goroutine and read by snapshots.
type opCounters struct {
	// Gauges (current values).
	activePartitions    atomic.Int64
	queuedItems         atomic.Int64
	queuedWaiters       atomic.Int64
	activeProviderCalls atomic.Int64

	// Cumulative counters.
	batches         atomic.Uint64
	totalBatchItems atomic.Uint64
	maxBatchSize    atomic.Int64
	inFlightJoins   atomic.Uint64
	memoHits        atomic.Uint64
	fallbacks       atomic.Uint64
	providerErrors  atomic.Uint64
	itemErrors      atomic.Uint64
	queueRejections atomic.Uint64

	// flushReasons is indexed by FlushReason.
	flushReasons [16]atomic.Uint64
}

// recordBatch records a dispatched batch of the given size and reason.
func (c *opCounters) recordBatch(size int, reason FlushReason) {
	c.batches.Add(1)
	c.totalBatchItems.Add(uint64(size))
	for {
		cur := c.maxBatchSize.Load()
		if int64(size) <= cur || c.maxBatchSize.CompareAndSwap(cur, int64(size)) {
			break
		}
	}
	if int(reason) < len(c.flushReasons) {
		c.flushReasons[reason].Add(1)
	}
}

// StatisticsSnapshot is an immutable, point-in-time view of engine statistics.
// Gauge fields are eventually consistent; cumulative fields are monotonic. It
// never exposes raw keys or partition components.
type StatisticsSnapshot struct {
	// State is the engine state at snapshot time.
	State EngineState
	// ActiveScopes is the number of open scopes.
	ActiveScopes int
	// BoundOperations is the number of bound operations.
	BoundOperations int
	// ActivePartitions is the total active partitions across operations.
	ActivePartitions int
	// QueuedItems is the total unique queued items.
	QueuedItems int
	// QueuedWaiters is the total logical waiters.
	QueuedWaiters int
	// ActiveProviderCalls is the number of in-flight provider calls.
	ActiveProviderCalls int
	// TotalProviderCalls is the cumulative number of provider calls.
	TotalProviderCalls uint64
	// TotalRequests is the cumulative number of logical requests.
	TotalRequests uint64
	// TotalBackendItems is the cumulative number of unique items sent to providers.
	TotalBackendItems uint64
	// TotalCallsSaved is the cumulative number of requests served without a
	// dedicated backend item through coalescing, deduplication, or memoization.
	TotalCallsSaved uint64
	// TotalFallbacks is the cumulative number of scalar fallbacks.
	TotalFallbacks uint64
	// TotalCancellations is the cumulative number of caller cancellations.
	TotalCancellations uint64
	// TotalContractViolations is the cumulative number of contract violations.
	TotalContractViolations uint64
	// Operations holds per-operation statistics, sorted by operation ID.
	Operations []OperationStats
}

// OperationStats is a per-operation statistics snapshot.
type OperationStats struct {
	// Operation is the operation ID.
	Operation string
	// ActivePartitions is the current number of active partitions.
	ActivePartitions int
	// QueuedItems is the current number of unique queued items.
	QueuedItems int
	// QueuedWaiters is the current number of logical waiters.
	QueuedWaiters int
	// ActiveProviderCalls is the current number of in-flight provider calls.
	ActiveProviderCalls int
	// Batches is the cumulative number of dispatched batches.
	Batches uint64
	// TotalBatchItems is the cumulative number of items across all batches.
	TotalBatchItems uint64
	// MaxBatchSize is the largest batch dispatched.
	MaxBatchSize int
	// InFlightJoins is the cumulative number of in-flight deduplication joins.
	InFlightJoins uint64
	// MemoHits is the cumulative number of scope memoization hits.
	MemoHits uint64
	// Fallbacks is the cumulative number of scalar fallbacks.
	Fallbacks uint64
	// ProviderErrors is the cumulative number of global provider errors.
	ProviderErrors uint64
	// ItemErrors is the cumulative number of per-item errors.
	ItemErrors uint64
	// QueueRejections is the cumulative number of rejected submissions.
	QueueRejections uint64
	// FlushReasons maps a flush reason name to its count.
	FlushReasons map[string]uint64
}

// snapshot builds an OperationStats from the atomic counters.
func (c *opCounters) snapshot(operation string) OperationStats {
	s := OperationStats{
		Operation:           operation,
		ActivePartitions:    int(c.activePartitions.Load()),
		QueuedItems:         int(c.queuedItems.Load()),
		QueuedWaiters:       int(c.queuedWaiters.Load()),
		ActiveProviderCalls: int(c.activeProviderCalls.Load()),
		Batches:             c.batches.Load(),
		TotalBatchItems:     c.totalBatchItems.Load(),
		MaxBatchSize:        int(c.maxBatchSize.Load()),
		InFlightJoins:       c.inFlightJoins.Load(),
		MemoHits:            c.memoHits.Load(),
		Fallbacks:           c.fallbacks.Load(),
		ProviderErrors:      c.providerErrors.Load(),
		ItemErrors:          c.itemErrors.Load(),
		QueueRejections:     c.queueRejections.Load(),
		FlushReasons:        make(map[string]uint64),
	}
	for r := range c.flushReasons {
		if v := c.flushReasons[r].Load(); v > 0 {
			s.FlushReasons[FlushReason(r).String()] = v
		}
	}
	return s
}

// Snapshot returns an immutable statistics snapshot of the engine.
func (e *Engine) Snapshot() StatisticsSnapshot {
	ops := e.controllersSnapshot()
	snap := StatisticsSnapshot{
		State:                   e.State(),
		ActiveScopes:            int(e.counters.activeScopes.Load()),
		BoundOperations:         len(ops),
		TotalProviderCalls:      e.counters.totalProviderCalls.Load(),
		TotalRequests:           e.counters.totalRequests.Load(),
		TotalBackendItems:       e.counters.totalBackendItems.Load(),
		TotalCallsSaved:         e.counters.totalCallsSaved.Load(),
		TotalFallbacks:          e.counters.totalFallbacks.Load(),
		TotalCancellations:      e.counters.totalCancellations.Load(),
		TotalContractViolations: e.counters.totalContractViolations.Load(),
	}
	for _, c := range ops { // already ID-sorted
		os := c.stats()
		snap.Operations = append(snap.Operations, os)
		snap.ActivePartitions += os.ActivePartitions
		snap.QueuedItems += os.QueuedItems
		snap.QueuedWaiters += os.QueuedWaiters
		snap.ActiveProviderCalls += os.ActiveProviderCalls
	}
	return snap
}
