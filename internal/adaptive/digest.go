package adaptive

import (
	"fmt"
	"sort"
)

// computeDigest returns the content-addressed digest of an operation profile. It
// covers the operation identity, adapter, sampling, and every distribution and
// counter, but not wall-clock metadata. The result is deterministic for a given
// logical profile regardless of collection order.
func (p *OperationProfile) computeDigest() Digest {
	var parts []string
	parts = append(parts, "op", p.Operation, "adapter", p.Adapter, "opdigest", string(p.OperationDigest))
	for _, m := range sortedModes(p.ExecutionModes) {
		parts = append(parts, "mode", string(m.Mode),
			fmt.Sprintf("calls=%d", m.Calls),
			fmt.Sprintf("backend=%d", m.BackendCalls))
		parts = append(parts, m.LatencyNanos.digestParts()...)
	}
	parts = append(parts, "arrivals", fmt.Sprintf("logical=%d", p.Arrivals.LogicalCalls))
	parts = append(parts, p.Arrivals.InterArrival.digestParts()...)
	parts = append(parts, "queue", fmt.Sprintf("rej=%d", p.Queue.Rejections))
	parts = append(parts, p.Queue.WaitNanos.digestParts()...)
	parts = append(parts, p.Queue.DepthItems.digestParts()...)
	parts = append(parts, "batches", fmt.Sprintf("n=%d", p.Batches.Batches), fmt.Sprintf("bc=%d", p.Batches.BackendCalls))
	parts = append(parts, p.Batches.Size.digestParts()...)
	parts = append(parts, p.Batches.Weight.digestParts()...)
	parts = append(parts, p.Batches.FlushReasons.digestParts("flush")...)
	parts = append(parts, "backend",
		fmt.Sprintf("fixed=%g", p.Backend.FixedCostNanos),
		fmt.Sprintf("peritem=%g", p.Backend.PerItemCostNanos),
		fmt.Sprintf("throttle=%d", p.Backend.ThrottleEvents))
	parts = append(parts, p.Backend.LatencyNanos.digestParts()...)
	parts = append(parts, p.Backend.SerializationNanos.digestParts()...)
	parts = append(parts, p.Backend.MappingNanos.digestParts()...)
	parts = append(parts, "deadlines",
		fmt.Sprintf("with=%d", p.Deadlines.WithDeadline),
		fmt.Sprintf("miss=%d", p.Deadlines.Misses))
	parts = append(parts, p.Deadlines.SlackNanos.digestParts()...)
	parts = append(parts, "errors", fmt.Sprintf("total=%d", p.Errors.Total))
	parts = append(parts, p.Errors.ByClass.digestParts("errclass")...)
	parts = append(parts, "dup", fmt.Sprintf("dup=%d", p.Duplicates.Duplicates), fmt.Sprintf("uniq=%d", p.Duplicates.Unique))
	parts = append(parts, p.Payloads.Bytes.digestParts()...)
	parts = append(parts, "partitions", fmt.Sprintf("distinct=%d", p.Partitions.DistinctClasses))
	parts = append(parts, p.Partitions.ByClass.digestParts("partclass")...)
	parts = append(parts, "fallbacks", fmt.Sprintf("total=%d", p.Fallbacks.Total))
	parts = append(parts, p.Fallbacks.ByReason.digestParts("fbreason")...)
	parts = append(parts, p.Chunks.Count.digestParts()...)
	parts = append(parts, p.Fairness.ServiceShare.digestParts("fairshare")...)
	parts = append(parts, p.Fairness.WaitNanos.digestParts()...)
	parts = append(parts, p.Sampling.digestParts()...)
	return hashParts(parts...)
}

// Finalize computes and stores the digest of every operation profile and of the
// bundle, and stamps the schema version. It must be called before persisting or
// comparing a bundle.
func (b *ProfileBundle) Finalize() {
	b.SchemaVersion = ProfileSchemaVersion
	sort.Slice(b.Operations, func(i, j int) bool {
		return b.Operations[i].Operation < b.Operations[j].Operation
	})
	var parts []string
	parts = append(parts, "bundle", b.SchemaVersion, "abi", b.RuntimeABI, "config", string(b.ConfigDigest))
	parts = append(parts, b.Toolchain.digestParts()...)
	parts = append(parts, b.Redaction.digestParts()...)
	for i := range b.Operations {
		b.Operations[i].Digest = b.Operations[i].computeDigest()
		parts = append(parts, "opref", b.Operations[i].Operation, string(b.Operations[i].Digest))
	}
	b.Digest = hashParts(parts...)
	if b.ID == "" {
		b.ID = shortID("bwprof", string(b.Digest))
	}
}

// sortedModes returns execution-mode profiles in stable rank order.
func sortedModes(modes []ExecutionModeProfile) []ExecutionModeProfile {
	out := make([]ExecutionModeProfile, len(modes))
	copy(out, modes)
	sort.Slice(out, func(i, j int) bool {
		ri, rj := executionModeOrder[out[i].Mode], executionModeOrder[out[j].Mode]
		if ri != rj {
			return ri < rj
		}
		return out[i].Mode < out[j].Mode
	})
	return out
}

// FindOperation returns the profile for the given operation ID, if present.
func (b *ProfileBundle) FindOperation(operation string) (*OperationProfile, bool) {
	for i := range b.Operations {
		if b.Operations[i].Operation == operation {
			return &b.Operations[i], true
		}
	}
	return nil, false
}
