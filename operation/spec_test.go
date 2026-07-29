package operation

import (
	"errors"
	"testing"

	"github.com/Voskan/BatchWeaver/diagnostics"
)

func TestMustNewSpecReadOnly(t *testing.T) {
	t.Parallel()
	spec := MustNewSpec(
		MustParseID("users.get"),
		ReadOnly(),
		WithOrderedResults(),
		WithRequestScope(),
		WithScalarSymbol(MustParseSymbol("github.com/example/app/users.(*Repository).GetUser")),
		WithBatchSymbol(MustParseSymbol("github.com/example/app/users.(*Repository).GetUsersBatch")),
	)
	if spec.ID() != "users.get" {
		t.Errorf("ID = %q", spec.ID())
	}
	if spec.Semantics().Kind() != KindReadOnly {
		t.Errorf("kind = %s", spec.Semantics().Kind())
	}
	if diags := spec.Validate(); diags.HasErrors() {
		t.Errorf("valid spec has errors: %s", diags.Sorted().Diagnostics())
	}
}

func TestNewSpecInvalidReturnsCollection(t *testing.T) {
	t.Parallel()
	// Retry enabled on a non-idempotent write is invalid.
	retry, err := NewRetryPolicy(RetryParams{
		Enabled: true, MaxAttempts: 3, InitialBackoff: 1, MaxBackoff: 2,
		Retryable: []RetryClassification{RetryTransport},
	})
	if err != nil {
		t.Fatalf("build retry: %v", err)
	}
	_, err = NewSpec(MustParseID("orders.create"), NonIdempotentWrite(), WithRetryPolicy(retry))
	if err == nil {
		t.Fatal("expected error for retry on non-idempotent write")
	}
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error is not *ValidationError: %v", err)
	}
	if !verr.Diagnostics.HasErrors() {
		t.Errorf("ValidationError carries no error diagnostics")
	}
	if verr.Diagnostics.FilterByCodePrefix("BWOP").Len() == 0 {
		t.Errorf("expected BWOP diagnostics")
	}
}

func TestSpecTransactionBoundRequiresPartition(t *testing.T) {
	t.Parallel()
	_, err := NewSpec(MustParseID("ledger.commit"), TransactionBound())
	if err == nil {
		t.Fatal("transaction-bound spec without transaction partition should be invalid")
	}
	var verr *ValidationError
	errors.As(err, &verr)
	if !containsCode(verr.Diagnostics, "BWOP011") {
		t.Errorf("expected BWOP011 partition/semantics diagnostic; got %v", verr.Diagnostics.Sorted().Diagnostics())
	}
}

func TestSpecTransactionBoundValidWithPartition(t *testing.T) {
	t.Parallel()
	part := NewPartitionContract(ScopeTransaction, []PartitionDimension{DimensionTransaction}, nil)
	_, err := NewSpec(MustParseID("ledger.commit"), TransactionBound(), WithPartitionContract(part))
	if err != nil {
		t.Errorf("valid transaction-bound spec rejected: %v", err)
	}
}

func TestSpecDeduplicationOnWriteRejected(t *testing.T) {
	t.Parallel()
	dedup, err := NewDeduplicationPolicy(DeduplicationParams{
		Mode: DeduplicationExact, ScopeMemoization: true, MaxItems: 10, MaxBytes: 1024,
	})
	if err != nil {
		t.Fatalf("build dedup: %v", err)
	}
	_, err = NewSpec(MustParseID("orders.create"), IdempotentWrite(), WithDeduplicationPolicy(dedup))
	if err == nil {
		t.Fatal("deduplication on a write without semantic permission should be invalid")
	}
	var verr *ValidationError
	errors.As(err, &verr)
	if !containsCode(verr.Diagnostics, "BWOP012") {
		t.Errorf("expected BWOP012; got %v", verr.Diagnostics.Sorted().Diagnostics())
	}
}

func TestSpecProcessScopeRequiresIsolation(t *testing.T) {
	t.Parallel()
	part := NewPartitionContract(ScopeProcess, nil, nil)
	_, err := NewSpec(MustParseID("cache.get"), ReadOnly(), WithPartitionContract(part))
	if err == nil {
		t.Fatal("process scope without isolation should be invalid")
	}
}

func TestSpecCanonicalDeterministic(t *testing.T) {
	t.Parallel()
	build := func() Spec {
		return MustNewSpec(MustParseID("users.get"), ReadOnly(), WithOrderedResults(), WithRequestScope())
	}
	a, b := build().Canonical(), build().Canonical()
	if a != b {
		t.Errorf("canonical not deterministic:\n%s\n---\n%s", a, b)
	}
}

func TestSpecCanonicalExcludesSource(t *testing.T) {
	t.Parallel()
	base := MustNewSpec(MustParseID("users.get"), ReadOnly())
	withSrc := MustNewSpec(MustParseID("users.get"), ReadOnly(),
		WithSource(Source{Range: diagnostics.Range{Start: diagnostics.Position{File: "x.yaml", Line: 3, Column: 1}}}))
	if base.Canonical() != withSrc.Canonical() {
		t.Errorf("source metadata leaked into canonical representation")
	}
}

func containsCode(c diagnostics.Collection, code diagnostics.Code) bool {
	for _, d := range c.Diagnostics() {
		if d.Code == code {
			return true
		}
	}
	return false
}
