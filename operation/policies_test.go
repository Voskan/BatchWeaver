package operation

import (
	"testing"
	"time"
)

func TestNamedSemanticsAreValid(t *testing.T) {
	t.Parallel()
	cons := map[string]Semantics{
		"read-only":               ReadOnly(),
		"freshness-sensitive":     FreshnessSensitiveRead(),
		"idempotent-write":        IdempotentWrite(),
		"non-idempotent-write":    NonIdempotentWrite(),
		"commutative-aggregation": CommutativeAggregation(),
		"ordered-aggregation":     OrderedAggregation(),
		"atomic-group":            AtomicGroup(),
		"transaction-bound":       TransactionBound(),
		"session-bound":           SessionBound(),
	}
	for name, sem := range cons {
		if err := sem.Validate(); err != nil {
			t.Errorf("%s semantics invalid: %v", name, err)
		}
		if sem.Effect() != sem.Kind().Effect() {
			t.Errorf("%s effect mismatch", name)
		}
	}
}

func TestSemanticsReorderableOnlyForReads(t *testing.T) {
	t.Parallel()
	if err := ReadOnly().WithReorderable().Validate(); err != nil {
		t.Errorf("read-only reorderable rejected: %v", err)
	}
	if err := IdempotentWrite().WithReorderable().Validate(); err == nil {
		t.Errorf("write reorderable should be rejected")
	}
}

func TestResultContractConstructors(t *testing.T) {
	t.Parallel()
	if err := OrderedResults().Validate(); err != nil {
		t.Errorf("ordered invalid: %v", err)
	}
	if err := KeyedResults().Validate(); err != nil {
		t.Errorf("keyed invalid: %v", err)
	}
	if err := SparseResults(MissingNotFound).Validate(); err != nil {
		t.Errorf("sparse invalid: %v", err)
	}
}

func TestSparseResultsPanicsOnContractViolation(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Errorf("SparseResults(MissingContractViolation) did not panic")
		}
	}()
	_ = SparseResults(MissingContractViolation)
}

func TestSchedulerPolicyValidation(t *testing.T) {
	t.Parallel()
	if err := DefaultSchedulerPolicy().Validate(); err != nil {
		t.Errorf("default scheduler invalid: %v", err)
	}
	// min > max
	if _, err := NewSchedulerPolicy(SchedulerParams{
		Mode: SchedulerImmediateWave, MinBatchSize: 10, MaxBatchSize: 5,
		MaxBatchWeight: 1, MaxPayloadBytes: 1, MaxConcurrency: 1,
		QueueItems: 1, QueueBytes: 1, ActivePartitions: 1, WaitersPerKey: 1,
	}); err == nil {
		t.Errorf("min>max accepted")
	}
	// manual with positive wait
	if _, err := NewSchedulerPolicy(SchedulerParams{
		Mode: SchedulerManual, MinBatchSize: 1, MaxBatchSize: 5, MaxBatchWeight: 1,
		MaxPayloadBytes: 1, MaxWait: time.Second, MaxConcurrency: 1,
		QueueItems: 1, QueueBytes: 1, ActivePartitions: 1, WaitersPerKey: 1,
	}); err == nil {
		t.Errorf("manual mode with positive wait accepted")
	}
	// adaptive without wait
	if _, err := NewSchedulerPolicy(SchedulerParams{
		Mode: SchedulerAdaptive, MinBatchSize: 1, MaxBatchSize: 5, MaxBatchWeight: 1,
		MaxPayloadBytes: 1, MaxWait: 0, MaxConcurrency: 1,
		QueueItems: 1, QueueBytes: 1, ActivePartitions: 1, WaitersPerKey: 1,
	}); err == nil {
		t.Errorf("adaptive without wait accepted")
	}
}

func TestDeduplicationPolicyValidation(t *testing.T) {
	t.Parallel()
	if err := DefaultDeduplicationPolicy().Validate(); err != nil {
		t.Errorf("default dedup invalid: %v", err)
	}
	if _, err := NewDeduplicationPolicy(DeduplicationParams{Mode: DeduplicationCanonical}); err == nil {
		t.Errorf("canonical without canonicalizer accepted")
	}
	if _, err := NewDeduplicationPolicy(DeduplicationParams{
		Mode: DeduplicationExact, ScopeMemoization: true, MaxItems: 0, MaxBytes: 0,
	}); err == nil {
		t.Errorf("unbounded memoization accepted")
	}
}

func TestRetryPolicyValidation(t *testing.T) {
	t.Parallel()
	if err := DefaultRetryPolicy().Validate(); err != nil {
		t.Errorf("default retry invalid: %v", err)
	}
	if _, err := NewRetryPolicy(RetryParams{Enabled: true, MaxAttempts: 1}); err == nil {
		t.Errorf("retry with <2 attempts accepted")
	}
	if _, err := NewRetryPolicy(RetryParams{
		Enabled: true, MaxAttempts: 3, InitialBackoff: 10, MaxBackoff: 5,
		Retryable: []RetryClassification{RetryTransport},
	}); err == nil {
		t.Errorf("maxBackoff < initialBackoff accepted")
	}
}

func TestFallbackPolicyValidation(t *testing.T) {
	t.Parallel()
	if err := DefaultFallbackPolicy().Validate(); err != nil {
		t.Errorf("default fallback invalid: %v", err)
	}
	if _, err := NewFallbackPolicy(FallbackParams{Mode: FallbackParallelScalar, MaxScalarConcurrency: 0}); err == nil {
		t.Errorf("parallel scalar without concurrency accepted")
	}
	if _, err := NewFallbackPolicy(FallbackParams{Mode: FallbackReject, MaxScalarConcurrency: 4}); err == nil {
		t.Errorf("reject with concurrency accepted")
	}
}

func TestCustomDimension(t *testing.T) {
	t.Parallel()
	d, err := CustomDimension("shard-key")
	if err != nil {
		t.Fatalf("CustomDimension: %v", err)
	}
	if d.IsWellKnown() {
		t.Errorf("custom dimension reported well-known")
	}
	if _, err := CustomDimension("tenant"); err == nil {
		t.Errorf("reserved custom dimension name accepted")
	}
	if err := DimensionReceiver.Validate(); err != nil {
		t.Errorf("well-known dimension invalid: %v", err)
	}
}

func TestEnumTextRoundTrip(t *testing.T) {
	t.Parallel()
	if v, err := ParseKind("read-only"); err != nil || v != KindReadOnly {
		t.Errorf("ParseKind: %v %v", v, err)
	}
	if _, err := ParseKind("bogus"); err == nil {
		t.Errorf("ParseKind accepted bogus")
	}
	if v, err := ParseScope("process"); err != nil || v != ScopeProcess {
		t.Errorf("ParseScope: %v %v", v, err)
	}
}
