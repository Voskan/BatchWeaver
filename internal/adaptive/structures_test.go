package adaptive

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaveGraphWavesAndCriticalPath(t *testing.T) {
	g := NewWaveGraph()
	for _, id := range []string{"a", "b", "c", "d"} {
		if err := g.AddNode(Node{ID: id, Kind: NodeOperation, Cost: 1}); err != nil {
			t.Fatal(err)
		}
	}
	_ = g.AddEdge(Edge{From: "a", To: "c", Kind: EdgeData})
	_ = g.AddEdge(Edge{From: "b", To: "c", Kind: EdgeData})
	_ = g.AddEdge(Edge{From: "c", To: "d", Kind: EdgeData})
	if d := g.Validate(); d != nil {
		t.Fatalf("unexpected invalid graph: %v", d)
	}
	waves, diag := g.Waves()
	if diag != nil {
		t.Fatalf("waves error: %v", diag)
	}
	if len(waves) != 3 {
		t.Fatalf("want 3 waves, got %d: %+v", len(waves), waves)
	}
	if len(waves[0].Nodes) != 2 {
		t.Errorf("wave 0 should co-schedule a and b, got %v", waves[0].Nodes)
	}
	cp := g.CriticalPath()
	if len(cp) != 3 || cp[len(cp)-1] != "d" {
		t.Errorf("unexpected critical path %v", cp)
	}
}

func TestWaveGraphCycleDetected(t *testing.T) {
	g := NewWaveGraph()
	_ = g.AddNode(Node{ID: "a", Kind: NodeOperation})
	_ = g.AddNode(Node{ID: "b", Kind: NodeOperation})
	_ = g.AddEdge(Edge{From: "a", To: "b"})
	_ = g.AddEdge(Edge{From: "b", To: "a"})
	if d := g.Validate(); d == nil || d.Code != CodeUnsupportedCycle {
		t.Fatalf("expected cycle diagnostic, got %v", d)
	}
}

func TestWaveFusionGroups(t *testing.T) {
	g := NewWaveGraph()
	_ = g.AddNode(Node{ID: "q1", Kind: NodeOperation, FusionGroup: "gql"})
	_ = g.AddNode(Node{ID: "q2", Kind: NodeOperation, FusionGroup: "gql"})
	waves, _ := g.Waves()
	if len(waves) != 1 || len(waves[0].FusionGroups["gql"]) != 2 {
		t.Errorf("expected one wave with a 2-member fusion group, got %+v", waves)
	}
}

func TestRecursiveBFSOrderAndFrontiers(t *testing.T) {
	contract := RecursiveContract[int, int]{
		Children: func(_ int, v int) []int {
			if v >= 6 {
				return nil
			}
			return []int{v*2 + 1, v*2 + 2}
		},
		Limits:      RecursiveLimits{MaxDepth: 8},
		Cycle:       CycleSkipSeen,
		ErrorPolicy: ErrCollectPerNode,
		ProofValid:  true,
	}
	loader := func(_ context.Context, keys []int) ([]NodeResult[int, int], error) {
		out := make([]NodeResult[int, int], len(keys))
		for i, k := range keys {
			out[i] = NodeResult[int, int]{Key: k, Value: k, Found: true}
		}
		return out, nil
	}
	res, err := Traverse(context.Background(), []int{0}, contract, loader)
	if err != nil {
		t.Fatalf("traverse: %v", err)
	}
	if len(res.FrontierSizes) == 0 || res.FrontierSizes[0] != 1 {
		t.Errorf("first frontier should be 1, got %v", res.FrontierSizes)
	}
	if res.Order[0].Key != 0 {
		t.Errorf("BFS should visit root first, got %d", res.Order[0].Key)
	}
}

func TestRecursiveDepthLimit(t *testing.T) {
	contract := RecursiveContract[int, int]{
		Children:   func(_ int, v int) []int { return []int{v + 1} },
		Limits:     RecursiveLimits{MaxDepth: 3},
		Cycle:      CycleSkipSeen,
		ProofValid: true,
	}
	loader := func(_ context.Context, keys []int) ([]NodeResult[int, int], error) {
		out := make([]NodeResult[int, int], len(keys))
		for i, k := range keys {
			out[i] = NodeResult[int, int]{Key: k, Value: k, Found: true}
		}
		return out, nil
	}
	_, err := Traverse(context.Background(), []int{0}, contract, loader)
	if !errors.Is(err, ErrRecursiveLimit) {
		t.Fatalf("expected depth-limit error, got %v", err)
	}
}

func TestRecursiveProofStale(t *testing.T) {
	contract := RecursiveContract[int, int]{Children: func(_ int, _ int) []int { return nil }, ProofValid: false}
	_, err := Traverse(context.Background(), []int{0}, contract, func(_ context.Context, _ []int) ([]NodeResult[int, int], error) { return nil, nil })
	if !errors.Is(err, ErrRecursiveProofStale) {
		t.Fatalf("expected stale-proof error, got %v", err)
	}
}

func TestRecursiveCycleError(t *testing.T) {
	// A -> B -> A cycle under CycleError.
	contract := RecursiveContract[int, int]{
		Children:   func(_ int, v int) []int { return []int{1 - v} }, // 0->1, 1->0
		Limits:     RecursiveLimits{MaxDepth: 10},
		Cycle:      CycleError,
		ProofValid: true,
	}
	loader := func(_ context.Context, keys []int) ([]NodeResult[int, int], error) {
		out := make([]NodeResult[int, int], len(keys))
		for i, k := range keys {
			out[i] = NodeResult[int, int]{Key: k, Value: k, Found: true}
		}
		return out, nil
	}
	_, err := Traverse(context.Background(), []int{0}, contract, loader)
	if !errors.Is(err, ErrRecursiveCycle) {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestFairSchedulerReservedAndStarvation(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0))
	s := NewFairScheduler(FairnessConfig{
		Algorithm:           FairDeficitRoundRobin,
		StarvationThreshold: 10 * time.Millisecond,
		Classes: []ClassPolicy{
			{Class: "a", Weight: 1},
			{Class: "b", Weight: 1},
		},
	}, clock)
	for i := 0; i < 10; i++ {
		s.Admit("a", 0)
		s.Admit("b", 0)
	}
	served := map[string]int{}
	for i := 0; i < 12; i++ {
		if c, ok := s.Next(); ok {
			served[c]++
			s.Serve(c)
		}
	}
	if served["a"] == 0 || served["b"] == 0 {
		t.Errorf("DRR should serve both classes, got %v", served)
	}
	clock.Advance(50 * time.Millisecond)
	starved, diags := s.DetectStarvation()
	if len(starved) == 0 || len(diags) == 0 {
		t.Error("expected starvation detection after threshold with backlog")
	}
}

func TestFairSchedulerQuotaRejection(t *testing.T) {
	s := NewFairScheduler(FairnessConfig{Classes: []ClassPolicy{{Class: "a", Weight: 1, Quota: Quota{MaxQueued: 2}}}}, NewFakeClock(time.Unix(0, 0)))
	s.Admit("a", 0)
	s.Admit("a", 0)
	r := s.Admit("a", 0)
	if r.Admitted || r.Diag == nil || r.Diag.Code != CodeQuotaExceeded {
		t.Errorf("expected quota rejection, got %+v", r)
	}
}

func TestOverloadStatesAndAdmission(t *testing.T) {
	det := NewOverloadDetector(OverloadConfig{Policy: AdmitShedLowPriority})
	if st := det.State(OverloadSignals{QueueDepthRatio: 0.5}); st != LoadNormal {
		t.Errorf("state = %s, want normal", st)
	}
	if st := det.State(OverloadSignals{QueueDepthRatio: 0.85}); st != LoadElevated {
		t.Errorf("state = %s, want elevated", st)
	}
	if st := det.State(OverloadSignals{QueueDepthRatio: 0.99}); st != LoadCritical {
		t.Errorf("state = %s, want critical", st)
	}
	d := det.Admit(AdmissionRequest{LowPriority: true}, OverloadSignals{QueueDepthRatio: 0.99})
	if d.Action != AdmitShedLowPriority || d.Diag == nil {
		t.Errorf("low-priority request under critical load should be shed, got %+v", d)
	}
	d = det.Admit(AdmissionRequest{Critical: true}, OverloadSignals{QueueDepthRatio: 0.99})
	if d.Action != AdmitAccept {
		t.Errorf("critical request must be admitted, got %+v", d)
	}
}

func TestRollbackMonitor(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0))
	m := NewRollbackMonitor(clock)
	d := &Decision{Operation: "x", Previous: Settings{MaxBatchSize: 256}, New: Settings{MaxBatchSize: 64}, Applied: true,
		Guardrails: SLOGuardrails{TimeoutRate: 0.001, RollbackWindow: time.Minute}}
	d.finalize()
	m.Arm(d, ObservedMetrics{})
	if !m.Armed("x") {
		t.Fatal("expected armed change")
	}
	// Breach the timeout guardrail.
	rb, ok := m.Evaluate("x", ObservedMetrics{TimeoutRate: 0.5})
	if !ok || rb == nil || rb.New != d.Previous {
		t.Fatalf("expected rollback restoring previous settings, got %+v ok=%t", rb, ok)
	}
	if len(rb.Diagnostics) == 0 || rb.Diagnostics[0].Code != CodePolicyRolledBack {
		t.Errorf("expected rollback diagnostic, got %+v", rb.Diagnostics)
	}
}

func TestRollbackWindowCommit(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0))
	m := NewRollbackMonitor(clock)
	d := &Decision{Operation: "x", Applied: true, Guardrails: SLOGuardrails{RollbackWindow: time.Minute}}
	d.finalize()
	m.Arm(d, ObservedMetrics{})
	clock.Advance(2 * time.Minute)
	if rb, ok := m.Evaluate("x", ObservedMetrics{TimeoutRate: 1}); ok || rb != nil {
		t.Error("after window elapses the change should commit, not roll back")
	}
	if m.Armed("x") {
		t.Error("change should be disarmed after window")
	}
}
