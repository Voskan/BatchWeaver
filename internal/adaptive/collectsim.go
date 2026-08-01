package adaptive

import (
	"sort"
	"time"
)

// CollectSynthetic runs a synthetic workload through a collector with a fake
// clock and returns a finalized, privacy-safe profile bundle. It is used by the
// profile-collection CLI demonstration and by tests: in a real deployment the
// collector is fed by the live runtime instead. The workload is grouped into
// batches under the given settings and backend model so the resulting profile
// carries realistic queue, batch, and backend distributions. It is fully
// deterministic for a given spec and seed.
func CollectSynthetic(spec WorkloadSpec, settings Settings, backend BackendModel) *ProfileBundle {
	events := GenerateWorkload(spec)
	clock := NewFakeClock(time.Unix(0, 0))
	col := NewCollector(CollectorOptions{
		Mode:        CollectHistograms,
		Clock:       clock,
		RuntimeABI:  "batchweaver.bridge/v1alpha1",
		ClasserSeed: spec.Seed + 1, // reproducible labels for deterministic collection
	})

	maxWait := settings.MaxWaitNanos
	maxSize := settings.MaxBatchSize
	if maxSize < 1 {
		maxSize = 1
	}

	// Group events into batches per partition using the windowed rule, mirroring
	// the runtime's flush decision, so batch and queue statistics are realistic.
	type batch struct {
		ready int64
		items []int
	}
	byPart := map[string][]int{}
	var partOrder []string
	for i, e := range events {
		if _, ok := byPart[e.PartitionClass]; !ok {
			partOrder = append(partOrder, e.PartitionClass)
		}
		byPart[e.PartitionClass] = append(byPart[e.PartitionClass], i)
	}
	sort.Strings(partOrder)

	var batches []batch
	for _, part := range partOrder {
		var cur []int
		var start int64
		flush := func() {
			if len(cur) == 0 {
				return
			}
			ready := start + maxWait
			if last := events[cur[len(cur)-1]].ArrivalNanos; last > ready {
				ready = last
			}
			batches = append(batches, batch{ready: ready, items: append([]int(nil), cur...)})
			cur = cur[:0]
		}
		for _, i := range byPart[part] {
			if len(cur) == 0 {
				start = events[i].ArrivalNanos
			}
			if len(cur) > 0 && events[i].ArrivalNanos > start+maxWait {
				flush()
				start = events[i].ArrivalNanos
			}
			cur = append(cur, i)
			if len(cur) >= maxSize {
				flush()
			}
		}
		flush()
	}
	sort.SliceStable(batches, func(i, j int) bool { return batches[i].ready < batches[j].ready })

	// Record calls at their arrival time, then batches at dispatch time.
	arrivalOrder := make([]int, len(events))
	for i := range events {
		arrivalOrder[i] = i
	}
	// Record calls with modeled queue wait derived from batch membership.
	dispatchAt := make([]int64, len(events))
	for _, b := range batches {
		for _, i := range b.items {
			dispatchAt[i] = b.ready
		}
	}
	for _, i := range arrivalOrder {
		e := events[i]
		clock.Set(time.Unix(0, e.ArrivalNanos))
		wait := time.Duration(dispatchAt[i] - e.ArrivalNanos)
		if wait < 0 {
			wait = 0
		}
		latency := time.Duration(backend.FixedNanos + backend.PerItemNanos)
		var slack time.Duration
		if e.DeadlineNanos > 0 {
			slack = time.Duration(e.DeadlineNanos)
		}
		col.RecordCall(CallObservation{
			Operation:      e.Operation,
			PartitionRaw:   e.PartitionClass,
			TenantRaw:      e.TenantClass,
			QueueWait:      wait,
			DeadlineSlack:  slack,
			HasDeadline:    e.DeadlineNanos > 0,
			DeadlineMissed: e.DeadlineNanos > 0 && e.ArrivalNanos+e.DeadlineNanos < dispatchAt[i]+int64(latency),
			PayloadBytes:   e.PayloadBytes,
			Mode:           ModeRuntimeCoalesced,
			Latency:        wait + latency,
			SampleKey:      e.Key,
		})
	}
	for _, b := range batches {
		clock.Set(time.Unix(0, b.ready))
		col.RecordBatch(BatchObservation{
			Operation:      spec.Operation,
			Size:           len(b.items),
			Weight:         int64(len(b.items)),
			Chunks:         1,
			FlushReason:    "wait",
			BackendCall:    true,
			BackendLatency: time.Duration(backend.FixedNanos + backend.PerItemNanos*float64(len(b.items))),
		})
	}
	return col.Bundle()
}
