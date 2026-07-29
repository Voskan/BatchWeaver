package proof

import (
	"bytes"
	"context"
	"testing"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/operation"
)

// readOnlySpec builds a read-only ordered spec with a receiver partition.
func readOnlySpec(id string) operation.Spec {
	return operation.MustNewSpec(
		operation.MustParseID(id),
		operation.ReadOnly(),
		operation.WithOrderedResults(),
		operation.WithPartitionContract(operation.NewPartitionContract(
			operation.ScopeRequest,
			[]operation.PartitionDimension{operation.DimensionReceiver}, nil)),
	)
}

// buildSnapshot assembles a minimal snapshot around a single candidate.
func buildSnapshot(op analysis.Operation, site analysis.CallSite, eff analysis.EffectSummary, state string) *analysis.Snapshot {
	cand := analysis.Candidate{
		ID: "bwcand_test", Operation: op.ID, State: state, CallSites: []string{site.ID},
	}
	return &analysis.Snapshot{
		SchemaVersion: analysis.SchemaVersion, Workspace: ".", BuildDigest: "sha256:analysis",
		Operations: []analysis.Operation{op}, CallSites: []analysis.CallSite{site},
		Effects: []analysis.EffectSummary{eff}, Candidates: []analysis.Candidate{cand},
	}
}

func cleanLoopSite() analysis.CallSite {
	return analysis.CallSite{
		ID: "bwcall_test", Operation: "users.get", Location: "svc/svc.go:10:2",
		EnclosingFunctionID: "F1", Dispatch: analysis.DispatchDirect, Targets: 1,
		LoopDepth: 1, ContextArg: "parameter ctx", Receiver: "field repo",
		KeyDependency: analysis.KeyStructural,
	}
}

func cleanEffect() analysis.EffectSummary {
	return analysis.EffectSummary{Function: "F1", Effects: nil, Complete: true}
}

func TestProveEligibleLoop(t *testing.T) {
	t.Parallel()
	op := analysis.Operation{ID: "users.get", Compatibility: "valid", Kind: "read-only"}
	snap := buildSnapshot(op, cleanLoopSite(), cleanEffect(), analysis.StatePotentialLoop)
	rep, err := Prove(context.Background(), Input{
		Snapshot: snap, Reproducible: true,
		Specs:          map[string]operation.Spec{"users.get": readOnlySpec("users.get")},
		ContractDigest: "sha256:contract",
	})
	if err != nil {
		t.Fatal(err)
	}
	c := rep.CandidateProofs[0]
	if c.Decision != DecisionProvenEligible {
		t.Fatalf("decision = %s, want proven_eligible; obligations: %+v", c.Decision, c.Obligations)
	}
	if !candidateHasStrategy(c, StrategyStaticLoopPrefetch, DecisionProvenEligible) {
		t.Errorf("expected static-loop-prefetch eligible, got %+v", c.AllowedStrategies)
	}
	if c.ProofID == "" {
		t.Error("missing proof ID")
	}
}

func TestProveIneligibleWrite(t *testing.T) {
	t.Parallel()
	op := analysis.Operation{ID: "events.append", Compatibility: "valid", Kind: "non-idempotent-write"}
	site := cleanLoopSite()
	site.Operation = "events.append"
	snap := buildSnapshot(op, site, cleanEffect(), analysis.StatePotentialLoop)
	rep, _ := Prove(context.Background(), Input{Snapshot: snap, Reproducible: true, ContractDigest: "d"})
	c := rep.CandidateProofs[0]
	if c.Decision != DecisionProvenIneligible {
		t.Fatalf("decision = %s, want proven_ineligible", c.Decision)
	}
	if c.ReasonCode != ReasonWriteCategory {
		t.Errorf("reason = %s, want %s", c.ReasonCode, ReasonWriteCategory)
	}
}

func TestProveUnknownAmbiguousTarget(t *testing.T) {
	t.Parallel()
	op := analysis.Operation{ID: "users.get", Compatibility: "valid", Kind: "read-only"}
	site := cleanLoopSite()
	site.Dispatch = analysis.DispatchInterface
	site.Targets = 0
	snap := buildSnapshot(op, site, cleanEffect(), analysis.StateAmbiguousTarget)
	rep, _ := Prove(context.Background(), Input{Snapshot: snap, Reproducible: true,
		Specs: map[string]operation.Spec{"users.get": readOnlySpec("users.get")}})
	c := rep.CandidateProofs[0]
	if c.Decision != DecisionUnknown {
		t.Fatalf("decision = %s, want unknown", c.Decision)
	}
	if c.ReasonCode != ReasonAmbiguousTarget {
		t.Errorf("reason = %s, want %s", c.ReasonCode, ReasonAmbiguousTarget)
	}
}

func TestProveDeferredDisabled(t *testing.T) {
	t.Parallel()
	op := analysis.Operation{ID: "users.get", Compatibility: "valid", Kind: "read-only", Disabled: true}
	snap := buildSnapshot(op, cleanLoopSite(), cleanEffect(), analysis.StateDisabledOperation)
	rep, _ := Prove(context.Background(), Input{Snapshot: snap, Reproducible: true})
	if got := rep.CandidateProofs[0].Decision; got != DecisionDeferred {
		t.Fatalf("decision = %s, want deferred", got)
	}
}

func TestProveIneligibleInvalidDeclaration(t *testing.T) {
	t.Parallel()
	op := analysis.Operation{ID: "users.get", Compatibility: "invalid"}
	snap := buildSnapshot(op, cleanLoopSite(), cleanEffect(), analysis.StateInvalidDeclaration)
	rep, _ := Prove(context.Background(), Input{Snapshot: snap, Reproducible: true})
	c := rep.CandidateProofs[0]
	if c.Decision != DecisionProvenIneligible || c.ReasonCode != ReasonInvalidDeclaration {
		t.Fatalf("decision=%s reason=%s, want proven_ineligible/%s", c.Decision, c.ReasonCode, ReasonInvalidDeclaration)
	}
}

func TestProveUnknownWithoutContract(t *testing.T) {
	t.Parallel()
	// A structurally perfect loop without a result contract cannot be proven.
	op := analysis.Operation{ID: "users.get", Compatibility: "valid", Kind: "read-only"}
	snap := buildSnapshot(op, cleanLoopSite(), cleanEffect(), analysis.StatePotentialLoop)
	rep, _ := Prove(context.Background(), Input{Snapshot: snap, Reproducible: true})
	if got := rep.CandidateProofs[0].Decision; got != DecisionUnknown {
		t.Fatalf("decision = %s, want unknown without a contract", got)
	}
}

func TestProveBarrierBlocksStatic(t *testing.T) {
	t.Parallel()
	op := analysis.Operation{ID: "users.get", Compatibility: "valid", Kind: "read-only"}
	eff := analysis.EffectSummary{Function: "F1", Effects: []string{"global-write"}, Complete: true}
	snap := buildSnapshot(op, cleanLoopSite(), eff, analysis.StatePotentialLoop)
	rep, _ := Prove(context.Background(), Input{Snapshot: snap, Reproducible: true,
		Specs: map[string]operation.Spec{"users.get": readOnlySpec("users.get")}})
	c := rep.CandidateProofs[0]
	if !candidateHasStrategy(c, StrategyStaticLoopPrefetch, DecisionUnknown) {
		t.Errorf("static prefetch should be unknown under a barrier; got %+v", c.AllowedStrategies)
	}
	// Runtime coalescing does not require the order obligation, so it remains
	// eligible and lifts the candidate decision.
	if c.Decision != DecisionProvenEligible {
		t.Errorf("decision = %s, want proven_eligible via runtime coalescing", c.Decision)
	}
}

func TestProveDeterministicAndInvalidation(t *testing.T) {
	t.Parallel()
	op := analysis.Operation{ID: "users.get", Compatibility: "valid", Kind: "read-only"}
	specs := map[string]operation.Spec{"users.get": readOnlySpec("users.get")}
	mk := func(contract string) *Report {
		snap := buildSnapshot(op, cleanLoopSite(), cleanEffect(), analysis.StatePotentialLoop)
		r, _ := Prove(context.Background(), Input{Snapshot: snap, Reproducible: true, Specs: specs, ContractDigest: contract})
		return r
	}
	a, b := mk("sha256:v1"), mk("sha256:v1")
	var ba, bb bytes.Buffer
	_ = RenderJSON(&ba, a)
	_ = RenderJSON(&bb, b)
	if ba.String() != bb.String() {
		t.Fatal("identical inputs produced different proof reports")
	}
	c := mk("sha256:v2")
	if a.CandidateProofs[0].ProofID == c.CandidateProofs[0].ProofID {
		t.Error("changing the contract digest must change the proof ID")
	}
}

func TestProveCancellation(t *testing.T) {
	t.Parallel()
	op := analysis.Operation{ID: "users.get", Compatibility: "valid", Kind: "read-only"}
	snap := buildSnapshot(op, cleanLoopSite(), cleanEffect(), analysis.StatePotentialLoop)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Prove(ctx, Input{Snapshot: snap}); err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestProveNilSnapshot(t *testing.T) {
	t.Parallel()
	if _, err := Prove(context.Background(), Input{}); err == nil {
		t.Fatal("expected error for nil snapshot")
	}
}

func candidateHasStrategy(c CandidateProof, id string, status Decision) bool {
	for _, s := range c.AllowedStrategies {
		if s.Strategy == id && s.Status == status {
			return true
		}
	}
	return false
}
