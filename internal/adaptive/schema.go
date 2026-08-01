package adaptive

import (
	"fmt"
	"sort"
)

// ExecutionMode is a member of the closed set of batching execution modes an
// operation may run under. Mode availability is declared by the operation
// contract and adapter, never inferred by the controller.
type ExecutionMode string

const (
	// ModeDirectScalar runs each call directly without batching.
	ModeDirectScalar ExecutionMode = "direct-scalar"
	// ModeRuntimeCoalesced coalesces concurrent scope calls at runtime.
	ModeRuntimeCoalesced ExecutionMode = "runtime-coalesced"
	// ModeBatchOfOne dispatches immediately with a batch size of one.
	ModeBatchOfOne ExecutionMode = "batch-of-one"
	// ModeStaticPrefetch uses a compiler-inserted static prefetch batch.
	ModeStaticPrefetch ExecutionMode = "static-prefetch"
	// ModeNativeBatch calls a native backend batch endpoint.
	ModeNativeBatch ExecutionMode = "native-batch"
	// ModePipeline pipelines requests over a persistent backend session.
	ModePipeline ExecutionMode = "pipeline"
	// ModeFallback is scalar execution used as a safety fallback.
	ModeFallback ExecutionMode = "fallback"
)

// executionModeOrder gives every mode a stable rank for deterministic output.
var executionModeOrder = map[ExecutionMode]int{
	ModeDirectScalar: 0, ModeRuntimeCoalesced: 1, ModeBatchOfOne: 2,
	ModeStaticPrefetch: 3, ModeNativeBatch: 4, ModePipeline: 5, ModeFallback: 6,
}

// KnownExecutionMode reports whether m is a recognized execution mode.
func KnownExecutionMode(m ExecutionMode) bool {
	_, ok := executionModeOrder[m]
	return ok
}

// ToolchainIdentity captures the build identity a profile was collected under so
// stale-toolchain profiles are detected on load.
type ToolchainIdentity struct {
	GoVersion string `json:"go_version"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

func (t ToolchainIdentity) digestParts() []string {
	return []string{"toolchain", t.GoVersion, t.Version, t.GOOS, t.GOARCH}
}

// TimeWindow is the closed observation window of a profile. The times are
// metadata and excluded from the profile digest so that two structurally
// identical profiles compare equal regardless of when they were collected.
type TimeWindow struct {
	StartUnixNanos int64 `json:"start_unix_nanos"`
	EndUnixNanos   int64 `json:"end_unix_nanos"`
}

// SamplingMetadata describes how a profile was sampled so counts can be
// interpreted (and scaled) correctly. Presenting sampled counts as exact is a
// bug; consumers must consult this.
type SamplingMetadata struct {
	Strategy    string  `json:"strategy"`
	Rate        float64 `json:"rate"`
	Reservoir   int     `json:"reservoir_size"`
	TailBiased  bool    `json:"tail_biased"`
	ScaleFactor float64 `json:"scale_factor"`
	Dropped     uint64  `json:"dropped_samples"`
}

func (s SamplingMetadata) digestParts() []string {
	return []string{
		"sampling", s.Strategy,
		fmt.Sprintf("rate=%g", s.Rate),
		fmt.Sprintf("res=%d", s.Reservoir),
		fmt.Sprintf("tail=%t", s.TailBiased),
		fmt.Sprintf("scale=%g", s.ScaleFactor),
	}
}

// RedactionSummary records the privacy transformation applied to a profile.
type RedactionSummary struct {
	// PartitionClassing describes how raw partitions were anonymized.
	PartitionClassing string `json:"partition_classing"`
	// TenantClassing describes how raw tenants were anonymized.
	TenantClassing string `json:"tenant_classing"`
	// RedactedFields counts fields dropped or anonymized during collection.
	RedactedFields int `json:"redacted_fields"`
	// RawKeysStored is always false; it exists so audits can assert on it.
	RawKeysStored bool `json:"raw_keys_stored"`
}

func (r RedactionSummary) digestParts() []string {
	return []string{"redaction", r.PartitionClassing, r.TenantClassing, fmt.Sprintf("rawkeys=%t", r.RawKeysStored)}
}

// CategoricalCounts is a bounded, deterministic map from an anonymized class to
// a count. Keys are anonymized (never raw identifiers) and the number of keys is
// bounded by the collector.
type CategoricalCounts map[string]uint64

func (c CategoricalCounts) digestParts(prefix string) []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{prefix}
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, c[k]))
	}
	return parts
}

// ArrivalProfile summarizes call arrivals for an operation.
type ArrivalProfile struct {
	// InterArrival is the distribution of nanoseconds between successive calls.
	InterArrival HistogramData `json:"inter_arrival_nanos"`
	// LogicalCalls is the number of logical (caller-visible) calls observed.
	LogicalCalls uint64 `json:"logical_calls"`
}

// QueueProfile summarizes queue behavior.
type QueueProfile struct {
	// WaitNanos is the distribution of per-item queue wait.
	WaitNanos HistogramData `json:"wait_nanos"`
	// DepthItems is the distribution of observed queue depth at enqueue.
	DepthItems HistogramData `json:"depth_items"`
	// Rejections counts admission/queue rejections.
	Rejections uint64 `json:"rejections"`
}

// BatchProfile summarizes dispatched batches.
type BatchProfile struct {
	// Size is the distribution of dispatched batch sizes.
	Size HistogramData `json:"size"`
	// Weight is the distribution of dispatched batch weights.
	Weight HistogramData `json:"weight"`
	// FlushReasons counts flush reasons by name.
	FlushReasons CategoricalCounts `json:"flush_reasons"`
	// Batches is the total number of dispatched batches.
	Batches uint64 `json:"batches"`
	// BackendCalls is the total number of backend calls made.
	BackendCalls uint64 `json:"backend_calls"`
}

// BackendProfile summarizes backend execution.
type BackendProfile struct {
	// LatencyNanos is the distribution of backend call latency.
	LatencyNanos HistogramData `json:"latency_nanos"`
	// SerializationNanos is the distribution of request serialization cost.
	SerializationNanos HistogramData `json:"serialization_nanos"`
	// MappingNanos is the distribution of result-mapping cost.
	MappingNanos HistogramData `json:"mapping_nanos"`
	// FixedCostNanos is the modeled fixed per-call cost estimate.
	FixedCostNanos float64 `json:"fixed_cost_nanos"`
	// PerItemCostNanos is the modeled marginal per-item cost estimate.
	PerItemCostNanos float64 `json:"per_item_cost_nanos"`
	// ThrottleEvents counts observed backend throttling responses.
	ThrottleEvents uint64 `json:"throttle_events"`
}

// DeadlineProfile summarizes deadline behavior.
type DeadlineProfile struct {
	// SlackNanos is the distribution of caller deadline slack at enqueue.
	SlackNanos HistogramData `json:"slack_nanos"`
	// WithDeadline counts calls carrying a deadline.
	WithDeadline uint64 `json:"with_deadline"`
	// Misses counts calls that missed their deadline.
	Misses uint64 `json:"misses"`
}

// ErrorProfile summarizes errors by anonymized class.
type ErrorProfile struct {
	// ByClass counts errors by anonymized class name.
	ByClass CategoricalCounts `json:"by_class"`
	// Total is the total number of errors observed.
	Total uint64 `json:"total"`
}

// DuplicateProfile summarizes duplicate keys.
type DuplicateProfile struct {
	// Duplicates counts logical calls served by an existing in-flight or memoized
	// item.
	Duplicates uint64 `json:"duplicates"`
	// Unique counts distinct backend items.
	Unique uint64 `json:"unique"`
}

// PayloadProfile summarizes payload sizes.
type PayloadProfile struct {
	// Bytes is the distribution of per-item payload bytes.
	Bytes HistogramData `json:"bytes"`
}

// PartitionProfile summarizes the partition distribution by anonymized class.
type PartitionProfile struct {
	// ByClass counts items by anonymized partition class.
	ByClass CategoricalCounts `json:"by_class"`
	// DistinctClasses is the number of distinct classes observed (bounded).
	DistinctClasses int `json:"distinct_classes"`
}

// FallbackProfile summarizes scalar fallbacks by anonymized reason.
type FallbackProfile struct {
	// ByReason counts fallbacks by anonymized reason.
	ByReason CategoricalCounts `json:"by_reason"`
	// Total is the total number of fallbacks.
	Total uint64 `json:"total"`
}

// ChunkProfile summarizes chunking.
type ChunkProfile struct {
	// Count is the distribution of chunks per batch.
	Count HistogramData `json:"count"`
}

// FairnessProfile summarizes per-class service shares (anonymized).
type FairnessProfile struct {
	// ServiceShare maps an anonymized class to its share of served items.
	ServiceShare CategoricalCounts `json:"service_share"`
	// WaitNanos is the distribution of per-class scheduling wait.
	WaitNanos HistogramData `json:"wait_nanos"`
}

// ExecutionModeProfile summarizes behavior observed under one execution mode.
type ExecutionModeProfile struct {
	Mode         ExecutionMode `json:"mode"`
	Calls        uint64        `json:"calls"`
	BackendCalls uint64        `json:"backend_calls"`
	LatencyNanos HistogramData `json:"latency_nanos"`
}

// OperationProfile is the per-operation workload profile.
type OperationProfile struct {
	Operation       string                 `json:"operation"`
	Adapter         string                 `json:"adapter"`
	OperationDigest Digest                 `json:"operation_digest"`
	ExecutionModes  []ExecutionModeProfile `json:"execution_modes"`
	Arrivals        ArrivalProfile         `json:"arrivals"`
	Queue           QueueProfile           `json:"queue"`
	Batches         BatchProfile           `json:"batches"`
	Backend         BackendProfile         `json:"backend"`
	Deadlines       DeadlineProfile        `json:"deadlines"`
	Errors          ErrorProfile           `json:"errors"`
	Duplicates      DuplicateProfile       `json:"duplicates"`
	Payloads        PayloadProfile         `json:"payloads"`
	Partitions      PartitionProfile       `json:"partitions"`
	Fallbacks       FallbackProfile        `json:"fallbacks"`
	Chunks          ChunkProfile           `json:"chunks"`
	Fairness        FairnessProfile        `json:"fairness"`
	Sampling        SamplingMetadata       `json:"sampling"`
	Digest          Digest                 `json:"digest"`
}

// ProfileBundle is the top-level, versioned, deterministic workload profile. Its
// Digest is content-addressed over the operation profiles and identity fields
// but excludes wall-clock metadata (Created, Window) so structurally identical
// bundles compare equal.
type ProfileBundle struct {
	SchemaVersion    string             `json:"schema_version"`
	ID               string             `json:"id"`
	Toolchain        ToolchainIdentity  `json:"toolchain"`
	RuntimeABI       string             `json:"runtime_abi"`
	ConfigDigest     Digest             `json:"config_digest"`
	CreatedUnixNanos int64              `json:"created_unix_nanos"`
	Window           TimeWindow         `json:"window"`
	Operations       []OperationProfile `json:"operations"`
	Redaction        RedactionSummary   `json:"redaction"`
	Digest           Digest             `json:"digest"`
}
