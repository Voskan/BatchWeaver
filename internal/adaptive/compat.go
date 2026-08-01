package adaptive

import (
	"fmt"
	"time"
)

// CompatibilityRequirement describes the environment a profile must match to be
// used for active warm start. Empty fields are not checked.
type CompatibilityRequirement struct {
	RuntimeABI   string
	ConfigDigest Digest
	Toolchain    ToolchainIdentity
	// MaxAge bounds how old a profile may be for active use. Zero disables the
	// age check.
	MaxAge time.Duration
	// Now is the reference time for the age check; zero uses the system clock.
	Now time.Time
	// OperationDigests maps operation IDs to their current operation digest. A
	// mismatch marks that operation incompatible for warm start.
	OperationDigests map[string]Digest
}

// CompatibilityResult reports whether a profile may be used, distinguishing hard
// incompatibility (never warm-start) from staleness (offline comparison only).
type CompatibilityResult struct {
	Compatible  bool
	Stale       bool
	Diagnostics []Diagnostic
}

// UsableForActive reports whether the profile may drive active tuning.
func (r CompatibilityResult) UsableForActive() bool { return r.Compatible && !r.Stale }

// CheckCompatibility validates a bundle against a requirement. Structural
// mismatches (schema, ABI, config, toolchain, operation digest) are hard
// incompatibilities; excessive age is staleness. Both are reported as
// diagnostics so callers can explain the decision.
func CheckCompatibility(b *ProfileBundle, req CompatibilityRequirement) CompatibilityResult {
	res := CompatibilityResult{Compatible: true}
	fail := func(detail string) {
		res.Compatible = false
		res.Diagnostics = append(res.Diagnostics, newDiag(CodeProfileIncompatible, "warning", "", detail))
	}
	if b.SchemaVersion != ProfileSchemaVersion {
		fail(fmt.Sprintf("schema %q != %q", b.SchemaVersion, ProfileSchemaVersion))
	}
	if req.RuntimeABI != "" && b.RuntimeABI != req.RuntimeABI {
		fail(fmt.Sprintf("runtime ABI %q != %q", b.RuntimeABI, req.RuntimeABI))
	}
	if req.ConfigDigest != "" && b.ConfigDigest != req.ConfigDigest {
		fail(fmt.Sprintf("config digest %s != %s", b.ConfigDigest.Short(), req.ConfigDigest.Short()))
	}
	if req.Toolchain.GoVersion != "" && b.Toolchain.GoVersion != req.Toolchain.GoVersion {
		fail(fmt.Sprintf("toolchain %q != %q", b.Toolchain.GoVersion, req.Toolchain.GoVersion))
	}
	for op, want := range req.OperationDigests {
		if p, ok := b.FindOperation(op); ok && p.OperationDigest != "" && p.OperationDigest != want {
			fail(fmt.Sprintf("operation %q digest %s != %s", op, p.OperationDigest.Short(), want.Short()))
		}
	}
	if req.MaxAge > 0 {
		now := req.Now
		if now.IsZero() {
			now = time.Now()
		}
		age := now.Sub(time.Unix(0, b.CreatedUnixNanos))
		if age > req.MaxAge {
			res.Stale = true
			res.Diagnostics = append(res.Diagnostics, newDiag(CodeProfileStale, "info", "",
				fmt.Sprintf("age %s exceeds max %s", age.Round(time.Second), req.MaxAge)))
		}
	}
	return res
}

// Merge combines multiple compatible profile bundles into one, summing counters
// and merging histograms. Bundles must share schema, runtime ABI, and config
// digest; otherwise merge is rejected because the result would be meaningless.
// The merged window spans the union of the inputs.
func Merge(bundles ...*ProfileBundle) (*ProfileBundle, error) {
	if len(bundles) == 0 {
		return nil, fmt.Errorf("adaptive: merge requires at least one bundle")
	}
	base := bundles[0]
	out := &ProfileBundle{
		Toolchain:        base.Toolchain,
		RuntimeABI:       base.RuntimeABI,
		ConfigDigest:     base.ConfigDigest,
		CreatedUnixNanos: base.CreatedUnixNanos,
		Window:           base.Window,
		Redaction:        base.Redaction,
	}
	merged := map[string]*OperationProfile{}
	var order []string
	for _, b := range bundles {
		if b.RuntimeABI != base.RuntimeABI || b.ConfigDigest != base.ConfigDigest {
			return nil, fmt.Errorf("adaptive: cannot merge profiles with different ABI or config digest")
		}
		if b.Window.StartUnixNanos < out.Window.StartUnixNanos || out.Window.StartUnixNanos == 0 {
			out.Window.StartUnixNanos = b.Window.StartUnixNanos
		}
		if b.Window.EndUnixNanos > out.Window.EndUnixNanos {
			out.Window.EndUnixNanos = b.Window.EndUnixNanos
		}
		if b.CreatedUnixNanos > out.CreatedUnixNanos {
			out.CreatedUnixNanos = b.CreatedUnixNanos
		}
		for i := range b.Operations {
			op := &b.Operations[i]
			existing, ok := merged[op.Operation]
			if !ok {
				clone := *op
				merged[op.Operation] = &clone
				order = append(order, op.Operation)
				continue
			}
			if err := mergeOperation(existing, op); err != nil {
				return nil, err
			}
		}
	}
	for _, name := range order {
		out.Operations = append(out.Operations, *merged[name])
	}
	out.Finalize()
	return out, nil
}

// mergeOperation folds src into dst by summing counters and merging histograms.
func mergeOperation(dst, src *OperationProfile) error {
	pairs := []struct{ d, s *HistogramData }{
		{&dst.Arrivals.InterArrival, &src.Arrivals.InterArrival},
		{&dst.Queue.WaitNanos, &src.Queue.WaitNanos},
		{&dst.Queue.DepthItems, &src.Queue.DepthItems},
		{&dst.Batches.Size, &src.Batches.Size},
		{&dst.Batches.Weight, &src.Batches.Weight},
		{&dst.Backend.LatencyNanos, &src.Backend.LatencyNanos},
		{&dst.Backend.SerializationNanos, &src.Backend.SerializationNanos},
		{&dst.Backend.MappingNanos, &src.Backend.MappingNanos},
		{&dst.Deadlines.SlackNanos, &src.Deadlines.SlackNanos},
		{&dst.Payloads.Bytes, &src.Payloads.Bytes},
		{&dst.Chunks.Count, &src.Chunks.Count},
		{&dst.Fairness.WaitNanos, &src.Fairness.WaitNanos},
	}
	for _, p := range pairs {
		merged, err := mergeHistData(*p.d, *p.s)
		if err != nil {
			return err
		}
		*p.d = merged
	}
	dst.Arrivals.LogicalCalls += src.Arrivals.LogicalCalls
	dst.Queue.Rejections += src.Queue.Rejections
	dst.Batches.Batches += src.Batches.Batches
	dst.Batches.BackendCalls += src.Batches.BackendCalls
	dst.Backend.ThrottleEvents += src.Backend.ThrottleEvents
	dst.Deadlines.WithDeadline += src.Deadlines.WithDeadline
	dst.Deadlines.Misses += src.Deadlines.Misses
	dst.Errors.Total += src.Errors.Total
	dst.Duplicates.Duplicates += src.Duplicates.Duplicates
	dst.Duplicates.Unique += src.Duplicates.Unique
	dst.Fallbacks.Total += src.Fallbacks.Total
	mergeCounts(dst.Batches.FlushReasons, src.Batches.FlushReasons)
	mergeCounts(dst.Errors.ByClass, src.Errors.ByClass)
	mergeCounts(dst.Partitions.ByClass, src.Partitions.ByClass)
	mergeCounts(dst.Fallbacks.ByReason, src.Fallbacks.ByReason)
	mergeCounts(dst.Fairness.ServiceShare, src.Fairness.ServiceShare)
	fixed, perItem := estimateBackendCostsFromData(dst.Backend.LatencyNanos, dst.Batches.Size)
	dst.Backend.FixedCostNanos = fixed
	dst.Backend.PerItemCostNanos = perItem
	return nil
}

// mergeHistData merges two encoded histograms and returns the encoded result.
func mergeHistData(a, b HistogramData) (HistogramData, error) {
	ha, err := a.Decode()
	if err != nil {
		return HistogramData{}, err
	}
	hb, err := b.Decode()
	if err != nil {
		return HistogramData{}, err
	}
	if err := ha.Merge(hb); err != nil {
		return HistogramData{}, err
	}
	return ha.Encode(), nil
}

// mergeCounts adds src counts into dst, honoring the categorical bound.
func mergeCounts(dst, src CategoricalCounts) {
	for k, v := range src {
		if _, ok := dst[k]; !ok && len(dst) >= maxCategoricalKeys {
			dst[overflowClass] += v
			continue
		}
		dst[k] += v
	}
}
