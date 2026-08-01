package runtime

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Voskan/BatchWeaver/operation"
)

func TestAdaptiveSettingsClampToHardBounds(t *testing.T) {
	e := mustEngine(t)
	defer func() { _ = e.Close(context.Background()) }()
	op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerDeadlineAware, 500*time.Microsecond, 0, 256))),
		Binding[int, int]{Provider: doubleProvider(&recorder{})})

	// Above the configured limits: clamp down to the configured maxima.
	op.ApplyAdaptiveSettings(AdaptiveSettings{MaxWait: 5 * time.Millisecond, MaxBatchSize: 10000, MaxConcurrency: 1000})
	eff := op.EffectiveSettings()
	if eff.MaxWait != 500*time.Microsecond {
		t.Errorf("max_wait not clamped: %s", eff.MaxWait)
	}
	if eff.MaxBatchSize != 256 {
		t.Errorf("max_batch_size not clamped: %d", eff.MaxBatchSize)
	}
	if eff.MaxConcurrency != 8 {
		t.Errorf("concurrency not clamped: %d", eff.MaxConcurrency)
	}

	// Below the configured limits: applied as-is (tighter).
	op.ApplyAdaptiveSettings(AdaptiveSettings{MaxWait: 100 * time.Microsecond, MaxBatchSize: 4, MaxConcurrency: 2})
	eff = op.EffectiveSettings()
	if eff.MaxWait != 100*time.Microsecond || eff.MaxBatchSize != 4 || eff.MaxConcurrency != 2 {
		t.Errorf("adaptive tightening not honored: %+v", eff)
	}

	// Clear restores configured defaults.
	op.ClearAdaptiveSettings()
	eff = op.EffectiveSettings()
	if eff.MaxBatchSize != 256 || eff.MaxWait != 500*time.Microsecond {
		t.Errorf("clear did not restore defaults: %+v", eff)
	}
}

func TestAdaptiveMaxBatchSizeCapsDispatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := mustEngine(t)
		defer func() { _ = e.Close(context.Background()) }()
		rec := &recorder{}
		// Immediate-wave with a large size cap; adaptive will lower it to 2.
		op := mustBind(t, e, readDecl(t, "t.get", operation.WithSchedulerPolicy(schedPolicy(t, operation.SchedulerManual, 0, 0, 256))),
			Binding[int, int]{Provider: doubleProvider(rec)})
		op.ApplyAdaptiveSettings(AdaptiveSettings{MaxBatchSize: 2})

		ctx, scope, err := e.NewScope(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		for _, k := range []int{1, 2, 3, 4} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := op.Do(ctx, k); err != nil {
					t.Errorf("Do(%d): %v", k, err)
				}
			}()
		}
		synctest.Wait()
		if err := scope.Flush(ctx); err != nil {
			t.Fatalf("Flush: %v", err)
		}
		wg.Wait()

		rec.mu.Lock()
		sizes := append([]int(nil), rec.sizes...)
		rec.mu.Unlock()
		for _, s := range sizes {
			if s > 2 {
				t.Errorf("batch size %d exceeds adaptive cap of 2 (sizes=%v)", s, sizes)
			}
		}
	})
}
