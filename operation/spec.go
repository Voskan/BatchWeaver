package operation

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Spec is the central, immutable-by-convention description of an operation: its
// identity, semantics, optional Go symbols, and the contracts and policies that
// govern batching. Construct it with NewSpec or MustNewSpec; the value carries
// its own defaults for any contract or policy not supplied.
type Spec struct {
	id                  ID
	semantics           Semantics
	scalarSymbol        Symbol
	batchSymbol         Symbol
	keyContract         KeyContract
	resultContract      ResultContract
	partitionContract   PartitionContract
	schedulerPolicy     SchedulerPolicy
	deduplicationPolicy DeduplicationPolicy
	retryPolicy         RetryPolicy
	fallbackPolicy      FallbackPolicy
	source              Source
	extensions          []Extension
}

// SpecOption configures a Spec during construction.
type SpecOption func(*Spec)

// WithScalarSymbol sets the configuration-based scalar symbol.
func WithScalarSymbol(s Symbol) SpecOption { return func(sp *Spec) { sp.scalarSymbol = s } }

// WithBatchSymbol sets the configuration-based batch symbol.
func WithBatchSymbol(s Symbol) SpecOption { return func(sp *Spec) { sp.batchSymbol = s } }

// WithKeyContract sets the key contract.
func WithKeyContract(c KeyContract) SpecOption { return func(sp *Spec) { sp.keyContract = c } }

// WithResultContract sets the result contract.
func WithResultContract(c ResultContract) SpecOption { return func(sp *Spec) { sp.resultContract = c } }

// WithOrderedResults sets an ordered result contract.
func WithOrderedResults() SpecOption { return WithResultContract(OrderedResults()) }

// WithKeyedResults sets a keyed result contract.
func WithKeyedResults() SpecOption { return WithResultContract(KeyedResults()) }

// WithPartitionContract sets the partition contract.
func WithPartitionContract(c PartitionContract) SpecOption {
	return func(sp *Spec) { sp.partitionContract = c }
}

// WithRequestScope sets a request-scoped partition contract with no required
// dimensions. It is a convenience for the common case.
func WithRequestScope() SpecOption {
	return WithPartitionContract(NewPartitionContract(ScopeRequest, nil, nil))
}

// WithSchedulerPolicy sets the scheduler policy.
func WithSchedulerPolicy(p SchedulerPolicy) SpecOption {
	return func(sp *Spec) { sp.schedulerPolicy = p }
}

// WithDeduplicationPolicy sets the deduplication policy.
func WithDeduplicationPolicy(p DeduplicationPolicy) SpecOption {
	return func(sp *Spec) { sp.deduplicationPolicy = p }
}

// WithRetryPolicy sets the retry policy.
func WithRetryPolicy(p RetryPolicy) SpecOption { return func(sp *Spec) { sp.retryPolicy = p } }

// WithFallbackPolicy sets the fallback policy.
func WithFallbackPolicy(p FallbackPolicy) SpecOption {
	return func(sp *Spec) { sp.fallbackPolicy = p }
}

// WithSource sets diagnostic source metadata. It never affects the digest.
func WithSource(src Source) SpecOption { return func(sp *Spec) { sp.source = src } }

// WithExtensions sets namespaced extension data, copying it so the spec does not
// alias caller storage.
func WithExtensions(exts []Extension) SpecOption {
	return func(sp *Spec) {
		sp.extensions = make([]Extension, 0, len(exts))
		for _, e := range exts {
			sp.extensions = append(sp.extensions, e.clone())
		}
	}
}

// NewSpec builds a Spec from an ID, semantics, and options, applying defaults
// for any contract or policy not supplied, then validates it. On validation
// failure it returns an error that carries the full diagnostic collection; use
// errors.As with *ValidationError to retrieve it.
func NewSpec(id ID, semantics Semantics, options ...SpecOption) (Spec, error) {
	s := Spec{
		id:                  id,
		semantics:           semantics,
		keyContract:         DefaultKeyContract(),
		resultContract:      OrderedResults(),
		partitionContract:   DefaultPartitionContract(),
		schedulerPolicy:     DefaultSchedulerPolicy(),
		deduplicationPolicy: DefaultDeduplicationPolicy(),
		retryPolicy:         DefaultRetryPolicy(),
		fallbackPolicy:      DefaultFallbackPolicy(),
	}
	for _, opt := range options {
		opt(&s)
	}
	if diags := s.Validate(); diags.HasErrors() {
		return Spec{}, &ValidationError{ID: id, Diagnostics: diags}
	}
	return s, nil
}

// MustNewSpec is like NewSpec but panics on validation failure. It is intended
// for package-level declarations where an invalid spec is a programmer error.
// The panic message includes the operation ID and is deterministic.
func MustNewSpec(id ID, semantics Semantics, options ...SpecOption) Spec {
	s, err := NewSpec(id, semantics, options...)
	if err != nil {
		panic(fmt.Sprintf("operation.MustNewSpec(%q): %v", id, err))
	}
	return s
}

// ID returns the operation identifier.
func (s Spec) ID() ID { return s.id }

// Semantics returns the operation semantics.
func (s Spec) Semantics() Semantics { return s.semantics }

// ScalarSymbol returns the configuration-based scalar symbol, if any.
func (s Spec) ScalarSymbol() Symbol { return s.scalarSymbol }

// BatchSymbol returns the configuration-based batch symbol, if any.
func (s Spec) BatchSymbol() Symbol { return s.batchSymbol }

// KeyContract returns the key contract.
func (s Spec) KeyContract() KeyContract { return s.keyContract }

// ResultContract returns the result contract.
func (s Spec) ResultContract() ResultContract { return s.resultContract }

// PartitionContract returns the partition contract.
func (s Spec) PartitionContract() PartitionContract { return s.partitionContract }

// SchedulerPolicy returns the scheduler policy.
func (s Spec) SchedulerPolicy() SchedulerPolicy { return s.schedulerPolicy }

// DeduplicationPolicy returns the deduplication policy.
func (s Spec) DeduplicationPolicy() DeduplicationPolicy { return s.deduplicationPolicy }

// RetryPolicy returns the retry policy.
func (s Spec) RetryPolicy() RetryPolicy { return s.retryPolicy }

// FallbackPolicy returns the fallback policy.
func (s Spec) FallbackPolicy() FallbackPolicy { return s.fallbackPolicy }

// Source returns the diagnostic source metadata.
func (s Spec) Source() Source { return s.source }

// Extensions returns a copy of the namespaced extension data.
func (s Spec) Extensions() []Extension {
	out := make([]Extension, 0, len(s.extensions))
	for _, e := range s.extensions {
		out = append(out, e.clone())
	}
	return out
}

// ConfigBased reports whether the spec references its implementations by symbol
// rather than through a typed declaration.
func (s Spec) ConfigBased() bool {
	return !s.scalarSymbol.IsZero() || !s.batchSymbol.IsZero()
}

// Canonical returns a deterministic textual encoding of the spec's semantic
// content, suitable for hashing and comparison. It excludes source positions
// and file paths so that the same semantic spec always yields the same value.
func (s Spec) Canonical() string {
	var b strings.Builder
	line := func(k, v string) { b.WriteString(k); b.WriteByte('='); b.WriteString(v); b.WriteByte('\n') }

	line("id", string(s.id))
	line("kind", s.semantics.kind.String())
	line("effect", s.semantics.effect.String())
	line("idempotency", s.semantics.idempotency.String())
	line("ordering", s.semantics.ordering.String())
	line("atomicity", s.semantics.atomicity.String())
	line("retry_allowed", strconv.FormatBool(s.semantics.retryAllowed))
	line("dedup_allowed", strconv.FormatBool(s.semantics.deduplicationAllowed))
	line("cross_scope_allowed", strconv.FormatBool(s.semantics.crossScopeAllowed))
	line("freshness_dependent", strconv.FormatBool(s.semantics.freshnessDependent))
	line("scalar", symbolOrDash(s.scalarSymbol))
	line("batch", symbolOrDash(s.batchSymbol))
	line("key_comparable", strconv.FormatBool(s.keyContract.comparable))
	line("key_canonicalizer", symbolOrDash(s.keyContract.canonicalKeySymbol))

	r := s.resultContract
	line("result_mode", r.mode.String())
	line("result_missing", r.missing.String())
	line("result_error", r.errorMode.String())
	line("result_duplicate", r.duplicate.String())
	line("result_unexpected", r.unexpected.String())
	line("result_order_significant", strconv.FormatBool(r.orderSignificant))
	line("result_unknown_representable", strconv.FormatBool(r.unknownRepresentable))

	p := s.partitionContract
	line("partition_scope", p.scope.String())
	line("partition_required", joinDimensions(p.required))
	line("partition_optional", joinDimensions(p.optional))
	line("partition_cross_root_scope", strconv.FormatBool(p.crossRootScope))
	line("partition_receiver_policy", p.receiverPolicy.String())
	line("partition_missing_dimension", p.missingDimension.String())
	line("partition_max_active", strconv.Itoa(p.maxActivePartitions))
	line("partition_public", strconv.FormatBool(p.publicContextIndep))

	sc := s.schedulerPolicy.params
	line("sched_mode", sc.Mode.String())
	line("sched_min", strconv.Itoa(sc.MinBatchSize))
	line("sched_max", strconv.Itoa(sc.MaxBatchSize))
	line("sched_weight", strconv.Itoa(sc.MaxBatchWeight))
	line("sched_payload", strconv.FormatInt(sc.MaxPayloadBytes, 10))
	line("sched_wait", sc.MaxWait.String())
	line("sched_deadline_margin", sc.DeadlineMargin.String())
	line("sched_concurrency", strconv.Itoa(sc.MaxConcurrency))
	line("sched_queue_items", strconv.Itoa(sc.QueueItems))
	line("sched_queue_bytes", strconv.FormatInt(sc.QueueBytes, 10))
	line("sched_active_partitions", strconv.Itoa(sc.ActivePartitions))
	line("sched_waiters_per_key", strconv.Itoa(sc.WaitersPerKey))
	line("sched_fairness", sc.Fairness.String())
	line("sched_priority", strconv.FormatBool(sc.PrioritySupport))
	line("sched_overflow", sc.Overflow.String())

	d := s.deduplicationPolicy.params
	line("dedup_mode", d.Mode.String())
	line("dedup_inflight", strconv.FormatBool(d.InFlight))
	line("dedup_scope_memo", strconv.FormatBool(d.ScopeMemoization))
	line("dedup_negative_memo", strconv.FormatBool(d.NegativeMemoization))
	line("dedup_error_memo", strconv.FormatBool(d.ErrorMemoization))
	line("dedup_max_items", strconv.Itoa(d.MaxItems))
	line("dedup_max_bytes", strconv.FormatInt(d.MaxBytes, 10))
	line("dedup_canonicalizer", symbolOrDash(d.Canonicalizer))

	rt := s.retryPolicy.params
	line("retry_enabled", strconv.FormatBool(rt.Enabled))
	line("retry_max_attempts", strconv.Itoa(rt.MaxAttempts))
	line("retry_initial_backoff", rt.InitialBackoff.String())
	line("retry_max_backoff", rt.MaxBackoff.String())
	line("retry_jitter", rt.Jitter.String())
	line("retry_classes", joinClassifications(rt.Retryable))
	line("retry_respect_after", strconv.FormatBool(rt.RespectRetryAfter))
	line("retry_partial", strconv.FormatBool(rt.PartialItemRetry))
	line("retry_unknown_outcome", rt.UnknownOutcome.String())
	line("retry_idempotency_key", symbolOrDash(rt.IdempotencyKey))

	fb := s.fallbackPolicy.params
	line("fallback_mode", fb.Mode.String())
	line("fallback_max_concurrency", strconv.Itoa(fb.MaxScalarConcurrency))
	line("fallback_on_overflow", strconv.FormatBool(fb.OnQueueOverflow))
	line("fallback_on_unavailable", strconv.FormatBool(fb.OnProviderUnavailable))
	line("fallback_on_unsupported_partition", strconv.FormatBool(fb.OnUnsupportedPartition))

	exts := append([]Extension(nil), s.extensions...)
	sort.Slice(exts, func(i, j int) bool { return exts[i].Namespace < exts[j].Namespace })
	for _, e := range exts {
		line("ext:"+e.Namespace, base64.StdEncoding.EncodeToString(e.Data))
	}
	return b.String()
}

// symbolOrDash returns the symbol string, or "-" when the symbol is zero.
func symbolOrDash(s Symbol) string {
	if s.IsZero() {
		return "-"
	}
	return s.String()
}

// joinDimensions renders dimensions in sorted order for canonical encoding.
func joinDimensions(dims []PartitionDimension) string {
	out := make([]string, len(dims))
	for i, d := range dims {
		out[i] = string(d)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// joinClassifications renders retry classifications in sorted order.
func joinClassifications(cs []RetryClassification) string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.String()
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}
