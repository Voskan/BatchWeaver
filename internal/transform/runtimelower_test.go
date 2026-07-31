package transform

import (
	"context"
	"strings"
	"testing"
)

func rtPlan(t *testing.T, strategies ...StrategyID) *Plan {
	t.Helper()
	dir := fixtureDir(t, "transform", "rtfixture")
	plan, err := BuildPlan(context.Background(), Request{
		Patterns: []string{"./..."}, Dir: dir, ToolVersion: "test", Strategies: strategies,
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestRuntimeLoweringTypeChecks(t *testing.T) {
	plan := rtPlan(t, StrategyRuntimeCallCoalescing)
	if plan.Validation.TypeCheck != ValidationPassed {
		t.Fatalf("type check = %s, detail=%s", plan.Validation.TypeCheck, plan.Validation.Detail)
	}
	if len(plan.Transformations) == 0 {
		t.Fatalf("expected runtime lowerings; skipped=%+v", plan.Skipped)
	}
	var created, modified int
	for _, f := range plan.Files {
		if f.Created {
			created++
			if !strings.Contains(string(f.Transformed()), "bridge.Operation[") {
				t.Errorf("bridge file missing typed Operation:\n%s", f.Transformed())
			}
			if !strings.Contains(string(f.Transformed()), "DO NOT EDIT") {
				t.Error("bridge file missing generated header")
			}
		} else {
			modified++
			if !strings.Contains(string(f.Transformed()), ".Call(ctx,") {
				t.Errorf("modified file missing bridge call:\n%s", f.Transformed())
			}
		}
	}
	if created == 0 || modified == 0 {
		t.Errorf("created=%d modified=%d, want both > 0", created, modified)
	}
}

func TestRuntimeLoweringDeterministic(t *testing.T) {
	a := rtPlan(t, StrategyRuntimeCallCoalescing)
	b := rtPlan(t, StrategyRuntimeCallCoalescing)
	if a.ID != b.ID || a.Digest != b.Digest {
		t.Errorf("runtime plan not deterministic: %s vs %s", a.ID, b.ID)
	}
}

func TestRuntimeLoweringBridgeABIRecorded(t *testing.T) {
	plan := rtPlan(t, StrategyRuntimeCallCoalescing)
	for _, tr := range plan.Transformations {
		if tr.RuntimeABI != RuntimeABIVersion {
			t.Errorf("transformation %s ABI = %q, want %q", tr.ID, tr.RuntimeABI, RuntimeABIVersion)
		}
		if tr.Bridge == "" {
			t.Errorf("transformation %s missing bridge file", tr.ID)
		}
	}
}

func TestBridgeVarAndFileNames(t *testing.T) {
	t.Parallel()
	if got := bridgeVarName("users.get"); got != "bwopUsersGet" {
		t.Errorf("bridgeVarName = %q", got)
	}
	if got := bridgeFileName("users.get"); got != "zz_batchweaver_users_get_gen.go" {
		t.Errorf("bridgeFileName = %q", got)
	}
}
