package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	batchweaver "github.com/Voskan/BatchWeaver"
	"github.com/Voskan/BatchWeaver/operation"
)

// specDecl adapts an operation.Spec to the Declaration interface.
type specDecl struct{ s operation.Spec }

func (d specDecl) Spec() operation.Spec { return d.s }

// schedPolicy builds a scheduler policy for tests.
func schedPolicy(t *testing.T, mode operation.SchedulerMode, maxWait, margin time.Duration, maxSize int) operation.SchedulerPolicy {
	t.Helper()
	p, err := operation.NewSchedulerPolicy(operation.SchedulerParams{
		Mode: mode, MinBatchSize: 1, MaxBatchSize: maxSize, MaxBatchWeight: 1 << 20,
		MaxPayloadBytes: 1 << 30, MaxWait: maxWait, DeadlineMargin: margin, MaxConcurrency: 8,
		QueueItems: 1000, QueueBytes: 1 << 30, ActivePartitions: 1000, WaitersPerKey: 1000,
	})
	if err != nil {
		t.Fatalf("scheduler policy: %v", err)
	}
	return p
}

// readDecl builds a read-only operation declaration for tests.
func readDecl(t *testing.T, id string, opts ...operation.SpecOption) Declaration {
	t.Helper()
	base := []operation.SpecOption{operation.WithOrderedResults(), operation.WithRequestScope()}
	s := operation.MustNewSpec(operation.MustParseID(id), operation.ReadOnly(), append(base, opts...)...)
	return specDecl{s}
}

// recorder captures provider invocations.
type recorder struct {
	mu    sync.Mutex
	calls int
	items int
	sizes []int
}

func (r *recorder) record(n int) {
	r.mu.Lock()
	r.calls++
	r.items += n
	r.sizes = append(r.sizes, n)
	r.mu.Unlock()
}

func (r *recorder) snapshot() (calls, items int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls, r.items
}

// doubleProvider returns each key times ten and records the batch.
func doubleProvider(r *recorder) ProviderFunc[int, int] {
	return func(ctx context.Context, req batchweaver.BatchRequest[int]) (batchweaver.BatchResponse[int], error) {
		r.record(req.Len())
		outs := make([]batchweaver.Outcome[int], 0, req.Len())
		for _, it := range req.Items() {
			outs = append(outs, batchweaver.Success(it.ID, it.Key*10))
		}
		return batchweaver.NewBatchResponse(outs)
	}
}

func mustEngine(t *testing.T, opts ...EngineOption) *Engine {
	t.Helper()
	e, err := NewEngine(opts...)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

func mustBind(t *testing.T, e *Engine, decl Declaration, b Binding[int, int]) *BoundOperation[int, int] {
	t.Helper()
	if b.Keys == nil {
		b.Keys = ComparableKeys[int]()
	}
	op, err := Bind(e, decl, b)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	return op
}

func TestCoalescing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256))),
			Binding[int, int]{Provider: doubleProvider(rec)})

		ctx, scope, err := e.NewScope(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		got := make([]int, 3)
		var wg sync.WaitGroup
		for i, k := range []int{1, 2, 3} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := op.Do(ctx, k)
				if err != nil {
					t.Errorf("Do(%d): %v", k, err)
				}
				got[i] = v
			}()
		}
		synctest.Wait()
		if err := scope.Flush(ctx); err != nil {
			t.Fatalf("Flush: %v", err)
		}
		wg.Wait()

		if calls, items := rec.snapshot(); calls != 1 || items != 3 {
			t.Errorf("provider calls=%d items=%d, want 1 and 3", calls, items)
		}
		if got[0] != 10 || got[1] != 20 || got[2] != 30 {
			t.Errorf("results = %v", got)
		}
	})
}

func TestDeduplication(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		dedup, _ := operation.NewDeduplicationPolicy(operation.DeduplicationParams{Mode: operation.DeduplicationExact, InFlight: true})
		op := mustBind(t, e, readDecl(t, "t.get",
			operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256)),
			operation.WithDeduplicationPolicy(dedup)),
			Binding[int, int]{Provider: doubleProvider(rec)})

		ctx, scope, _ := e.NewScope(context.Background())
		var wg sync.WaitGroup
		results := make([]int, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := op.Do(ctx, 42)
				if err != nil {
					t.Errorf("Do: %v", err)
				}
				results[i] = v
			}()
		}
		synctest.Wait()
		_ = scope.Flush(ctx)
		wg.Wait()
		if calls, items := rec.snapshot(); calls != 1 || items != 1 {
			t.Errorf("dedup: provider calls=%d items=%d, want 1 and 1", calls, items)
		}
		if results[0] != 420 || results[1] != 420 {
			t.Errorf("both callers should get 420, got %v", results)
		}
	})
}

func TestPartitionIsolation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256))),
			Binding[int, int]{
				Provider: doubleProvider(rec),
				Partitioner: PartitionerFunc[int](func(ctx context.Context, key int) (Partition, error) {
					tenant, _ := ctx.Value(tenantKeyT{}).(string)
					return PartitionFromStrings(tenant), nil
				}),
			})
		ctx, scope, _ := e.NewScope(context.Background())
		var wg sync.WaitGroup
		for _, tenant := range []string{"a", "b"} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := op.Do(context.WithValue(ctx, tenantKeyT{}, tenant), 1)
				if err != nil {
					t.Errorf("Do: %v", err)
				}
			}()
		}
		synctest.Wait()
		_ = scope.Flush(ctx)
		wg.Wait()
		if calls, _ := rec.snapshot(); calls != 2 {
			t.Errorf("two tenants should produce two batches, got %d", calls)
		}
	})
}

type tenantKeyT struct{}

func TestCancelBeforeDispatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256))),
			Binding[int, int]{Provider: doubleProvider(rec)})
		ctx, scope, _ := e.NewScope(context.Background())

		cctx, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := op.Do(cctx, 1)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("canceled caller err = %v, want context.Canceled", err)
			}
		}()
		go func() {
			defer wg.Done()
			_, err := op.Do(ctx, 2)
			if err != nil {
				t.Errorf("other caller: %v", err)
			}
		}()
		synctest.Wait()
		cancel()
		synctest.Wait()
		_ = scope.Flush(ctx)
		wg.Wait()
		if calls, items := rec.snapshot(); calls != 1 || items != 1 {
			t.Errorf("canceled item should be removed: calls=%d items=%d, want 1 and 1", calls, items)
		}
	})
}

func TestDeadlineEarlyFlush(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		op := mustBind(t, e, readDecl(t, "t.get",
			operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerDeadlineAware, time.Second, 10*time.Millisecond, 256))),
			Binding[int, int]{Provider: doubleProvider(rec)})
		ctx, scope, _ := e.NewScope(context.Background())
		defer func() { _ = scope.Close(context.Background()) }()

		cctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		v, err := op.Do(cctx, 5)
		if err != nil {
			t.Fatalf("deadline-aware Do should flush before deadline, got %v", err)
		}
		if v != 50 {
			t.Errorf("value = %d, want 50", v)
		}
		if calls, _ := rec.snapshot(); calls != 1 {
			t.Errorf("provider calls = %d, want 1", calls)
		}
	})
}

func TestOverflowReject(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		limit := QueueLimits{MaxItems: 1}
		op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256))),
			Binding[int, int]{Provider: doubleProvider(rec), Limits: limit, Overflow: ptr(OverflowReject)})
		ctx, scope, _ := e.NewScope(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = op.Do(ctx, 1) }()
		synctest.Wait()
		_, err := op.Do(ctx, 2)
		if !errors.Is(err, ErrQueueFull) {
			t.Errorf("second submission err = %v, want ErrQueueFull", err)
		}
		_ = scope.Flush(ctx)
		wg.Wait()
	})
}

func TestOverflowFallback(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		fb := ScalarFallbackFunc[int, int](func(ctx context.Context, k int) (int, error) { return k * 100, nil })
		op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256))),
			Binding[int, int]{Provider: doubleProvider(rec), Limits: QueueLimits{MaxItems: 1}, Overflow: ptr(OverflowFallback), Fallback: fb})
		ctx, scope, _ := e.NewScope(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = op.Do(ctx, 1) }()
		synctest.Wait()
		v, err := op.Do(ctx, 7)
		if err != nil || v != 700 {
			t.Errorf("fallback = %d, %v; want 700, nil", v, err)
		}
		_ = scope.Flush(ctx)
		wg.Wait()
	})
}

func TestMemoization(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		memo, _ := operation.NewDeduplicationPolicy(operation.DeduplicationParams{Mode: operation.DeduplicationExact, ScopeMemoization: true, MaxItems: 100, MaxBytes: 1 << 20})
		op := mustBind(t, e, readDecl(t, "t.get",
			operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerImmediateWave, 0, 0, 256)),
			operation.WithDeduplicationPolicy(memo)),
			Binding[int, int]{Provider: doubleProvider(rec)})
		ctx, scope, _ := e.NewScope(context.Background())
		defer func() { _ = scope.Close(context.Background()) }()

		if v, err := op.Do(ctx, 1); err != nil || v != 10 {
			t.Fatalf("first Do = %d, %v", v, err)
		}
		if v, err := op.Do(ctx, 1); err != nil || v != 10 {
			t.Fatalf("second Do = %d, %v", v, err)
		}
		if calls, _ := rec.snapshot(); calls != 1 {
			t.Errorf("memoization: provider calls = %d, want 1", calls)
		}
	})
}

func TestProviderPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerImmediateWave, 0, 0, 256))),
			Binding[int, int]{Provider: ProviderFunc[int, int](func(context.Context, batchweaver.BatchRequest[int]) (batchweaver.BatchResponse[int], error) {
				panic("boom")
			})})
		ctx, scope, _ := e.NewScope(context.Background())
		defer func() { _ = scope.Close(context.Background()) }()
		_, err := op.Do(ctx, 1)
		if !errors.Is(err, ErrProviderPanic) {
			t.Errorf("err = %v, want ErrProviderPanic", err)
		}
	})
}

func TestContractViolations(t *testing.T) {
	cases := map[string]ProviderFunc[int, int]{
		"missing": func(ctx context.Context, req batchweaver.BatchRequest[int]) (batchweaver.BatchResponse[int], error) {
			return batchweaver.NewBatchResponse([]batchweaver.Outcome[int]{}) // omit result
		},
		"unexpected": func(ctx context.Context, req batchweaver.BatchRequest[int]) (batchweaver.BatchResponse[int], error) {
			return batchweaver.NewBatchResponse([]batchweaver.Outcome[int]{batchweaver.Success(999999, 1)})
		},
	}
	for name, prov := range cases {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				e := mustEngine(t)
				defer func() { _ = e.Close(context.Background()) }()
				op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerImmediateWave, 0, 0, 256))),
					Binding[int, int]{Provider: prov})
				ctx, scope, _ := e.NewScope(context.Background())
				defer func() { _ = scope.Close(context.Background()) }()
				_, err := op.Do(ctx, 1)
				if !errors.Is(err, ErrBatchContractViolation) {
					t.Errorf("%s: err = %v, want ErrBatchContractViolation", name, err)
				}
			})
		})
	}
}

func TestCollisionSafety(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		dedup, _ := operation.NewDeduplicationPolicy(operation.DeduplicationParams{Mode: operation.DeduplicationExact, InFlight: true})
		op := mustBind(t, e, readDecl(t, "t.get",
			operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256)),
			operation.WithDeduplicationPolicy(dedup)),
			Binding[int, int]{Provider: doubleProvider(rec), Keys: constantHashKeys{}})
		ctx, scope, _ := e.NewScope(context.Background())
		var wg sync.WaitGroup
		for _, k := range []int{1, 2} {
			wg.Add(1)
			go func() { defer wg.Done(); _, _ = op.Do(ctx, k) }()
		}
		synctest.Wait()
		_ = scope.Flush(ctx)
		wg.Wait()
		if _, items := rec.snapshot(); items != 2 {
			t.Errorf("colliding but unequal keys must stay separate: items = %d, want 2", items)
		}
	})
}

// constantHashKeys hashes every key to zero, forcing bucket collisions.
type constantHashKeys struct{}

func (constantHashKeys) Clone(k int) int       { return k }
func (constantHashKeys) Hash(int) uint64       { return 0 }
func (constantHashKeys) Equal(a, b int) bool   { return a == b }
func (constantHashKeys) EstimateBytes(int) int { return 8 }

func TestScopeRequired(t *testing.T) {
	e := mustEngine(t)
	defer func() { _ = e.Close(context.Background()) }()
	op := mustBind(t, e, readDecl(t, "t.get"), Binding[int, int]{Provider: doubleProvider(&recorder{})})
	_, err := op.Do(context.Background(), 1)
	if !errors.Is(err, ErrScopeRequired) {
		t.Errorf("err = %v, want ErrScopeRequired", err)
	}
}

func TestEngineAndScopeCloseIdempotent(t *testing.T) {
	e := mustEngine(t)
	_ = mustBind(t, e, readDecl(t, "t.get"), Binding[int, int]{Provider: doubleProvider(&recorder{})})
	_, scope, _ := e.NewScope(context.Background())
	if err := scope.Close(context.Background()); err != nil {
		t.Errorf("scope close: %v", err)
	}
	if err := scope.Close(context.Background()); err != nil {
		t.Errorf("second scope close: %v", err)
	}
	if err := e.Close(context.Background()); err != nil {
		t.Errorf("engine close: %v", err)
	}
	if err := e.Close(context.Background()); err != nil {
		t.Errorf("second engine close: %v", err)
	}
	if e.State() != EngineStateClosed {
		t.Errorf("state = %v, want closed", e.State())
	}
}

func TestBindRejectsMemoizationOnWrite(t *testing.T) {
	e := mustEngine(t)
	defer func() { _ = e.Close(context.Background()) }()
	// A non-idempotent write cannot enable memoization; binding validation must
	// reject it. (Operation-spec validation already rejects in-flight dedup on
	// writes, so that unsafe combination cannot even be constructed.)
	spec := operation.MustNewSpec(operation.MustParseID("orders.create"), operation.NonIdempotentWrite())
	_, err := Bind(e, specDecl{spec}, Binding[int, int]{
		Provider: doubleProvider(&recorder{}), Keys: ComparableKeys[int](), EnableMemoization: true,
	})
	if !errors.Is(err, ErrInvalidBinding) {
		t.Errorf("err = %v, want ErrInvalidBinding", err)
	}
}

func TestDuplicateBindingRejected(t *testing.T) {
	e := mustEngine(t)
	defer func() { _ = e.Close(context.Background()) }()
	_ = mustBind(t, e, readDecl(t, "t.get"), Binding[int, int]{Provider: doubleProvider(&recorder{})})
	_, err := Bind(e, readDecl(t, "t.get"), Binding[int, int]{Provider: doubleProvider(&recorder{}), Keys: ComparableKeys[int]()})
	if !errors.Is(err, ErrOperationAlreadyBound) {
		t.Errorf("err = %v, want ErrOperationAlreadyBound", err)
	}
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T { return &v }
