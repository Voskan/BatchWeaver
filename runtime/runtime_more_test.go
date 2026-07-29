package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"

	batchweaver "github.com/Voskan/BatchWeaver"
	"github.com/Voskan/BatchWeaver/operation"
)

func TestCancelAfterDispatchIndependent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		gate := make(chan struct{})
		provider := ProviderFunc[int, int](func(ctx context.Context, req batchweaver.BatchRequest[int]) (batchweaver.BatchResponse[int], error) {
			<-gate
			outs := make([]batchweaver.Outcome[int], 0, req.Len())
			for _, it := range req.Items() {
				outs = append(outs, batchweaver.Success(it.ID, it.Key*10))
			}
			return batchweaver.NewBatchResponse(outs)
		})
		op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256))),
			Binding[int, int]{Provider: provider})
		ctx, scope, _ := e.NewScope(context.Background())

		cctx, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := op.Do(cctx, 1); !errors.Is(err, context.Canceled) {
				t.Errorf("canceled caller err = %v, want context.Canceled", err)
			}
		}()
		var bValue int
		var bErr error
		go func() {
			defer wg.Done()
			bValue, bErr = op.Do(ctx, 2)
		}()

		synctest.Wait()      // both queued, blocked on waiters
		_ = scope.Flush(ctx) // dispatch one batch; provider blocks on gate
		synctest.Wait()      // provider blocked on gate
		cancel()             // cancel caller 1 after dispatch
		synctest.Wait()      // caller 1 returns
		close(gate)          // provider completes
		wg.Wait()

		if bErr != nil || bValue != 20 {
			t.Errorf("surviving caller got %d, %v; want 20, nil", bValue, bErr)
		}
	})
}

func TestStressManyCallers(t *testing.T) {
	e := mustEngine(t)
	defer func() { _ = e.Close(context.Background()) }()
	rec := &recorder{}
	op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerImmediateWave, 0, 0, 64))),
		Binding[int, int]{Provider: doubleProvider(rec)})

	const n = 400
	result, err := Run(e, context.Background(), func(ctx context.Context) ([]int, error) {
		out := make([]int, n)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := op.Do(ctx, i%50) // high duplicate rate
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}
				out[i] = v
			}()
		}
		wg.Wait()
		return out, firstErr
	})
	if err != nil {
		t.Fatalf("stress run: %v", err)
	}
	for i := 0; i < n; i++ {
		if result[i] != (i%50)*10 {
			t.Fatalf("result[%d] = %d, want %d", i, result[i], (i%50)*10)
		}
	}
}

func FuzzPartitionEncoding(f *testing.F) {
	f.Add("a", "b")
	f.Add("", "")
	f.Add("ab", "")
	f.Fuzz(func(t *testing.T, x, y string) {
		p1 := PartitionFromStrings(x, y)
		if p1 != PartitionFromStrings(x, y) {
			t.Errorf("equal inputs produced different partitions")
		}
		// Length-delimited encoding: a two-component partition never collides with
		// the one-component concatenation.
		if p1 == PartitionFromStrings(x+y) {
			t.Errorf("component grouping collision for %q,%q", x, y)
		}
		if p1.IsZero() {
			t.Errorf("valid partition reported zero")
		}
	})
}

func TestSnapshotReflectsActivity(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256))),
			Binding[int, int]{Provider: doubleProvider(rec)})
		ctx, scope, _ := e.NewScope(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = op.Do(ctx, 1) }()
		synctest.Wait()
		snap := e.Snapshot()
		if snap.QueuedItems != 1 || snap.BoundOperations != 1 {
			t.Errorf("snapshot queuedItems=%d boundOps=%d, want 1 and 1", snap.QueuedItems, snap.BoundOperations)
		}
		_ = scope.Flush(ctx)
		wg.Wait()
		final := e.Snapshot()
		if final.TotalProviderCalls != 1 || final.TotalRequests != 1 {
			t.Errorf("final providerCalls=%d requests=%d, want 1 and 1", final.TotalProviderCalls, final.TotalRequests)
		}
	})
}

// Example demonstrates binding an operation and coalescing calls within a scope.
func Example() {
	engine, err := NewEngine()
	if err != nil {
		panic(err)
	}
	defer func() { _ = engine.Close(context.Background()) }()

	spec := operation.MustNewSpec(
		operation.MustParseID("users.get"),
		operation.ReadOnly(),
		operation.WithOrderedResults(),
		operation.WithRequestScope(),
	)
	getUser, err := Bind(engine, specDeclFor(spec), Binding[int, string]{
		Keys: ComparableKeys[int](),
		Provider: ProviderFunc[int, string](func(ctx context.Context, req batchweaver.BatchRequest[int]) (batchweaver.BatchResponse[string], error) {
			outs := make([]batchweaver.Outcome[string], 0, req.Len())
			for _, it := range req.Items() {
				outs = append(outs, batchweaver.Success(it.ID, fmt.Sprintf("user-%d", it.Key)))
			}
			return batchweaver.NewBatchResponse(outs)
		}),
	})
	if err != nil {
		panic(err)
	}

	name, err := Run(engine, context.Background(), func(ctx context.Context) (string, error) {
		return getUser.Do(ctx, 42)
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(name)
	// Output: user-42
}

// specDeclFor adapts a spec to Declaration for the example.
func specDeclFor(s operation.Spec) Declaration { return specDecl{s} }
