package transform

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureDir(t *testing.T, parts ...string) string {
	t.Helper()
	elems := append([]string{"..", "..", "testdata"}, parts...)
	abs, err := filepath.Abs(filepath.Join(elems...))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestBuildPlanProofFixture(t *testing.T) {
	dir := fixtureDir(t, "proof", "fixture")
	plan, err := BuildPlan(context.Background(), Request{
		Patterns: []string{"./..."}, Dir: dir, ToolVersion: "test",
	})
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if plan.Validation.TypeCheck != ValidationPassed {
		t.Fatalf("type check = %s, detail=%s", plan.Validation.TypeCheck, plan.Validation.Detail)
	}
	if len(plan.Transformations) != 1 {
		t.Fatalf("transformations = %d, want 1; skipped=%+v", len(plan.Transformations), plan.Skipped)
	}
	tr := plan.Transformations[0]
	if tr.Operation != "users.get" {
		t.Errorf("operation = %s, want users.get", tr.Operation)
	}
	if tr.Strategy != StrategyStaticLoopPrefetch {
		t.Errorf("strategy = %s", tr.Strategy)
	}
	if len(plan.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(plan.Files))
	}
	gen := string(plan.Files[0].Transformed())
	for _, want := range []string{"bwKeys", "GetUsersBatch", "bwValues", "bwIndex"} {
		if !strings.Contains(gen, want) {
			t.Errorf("generated code missing %q:\n%s", want, gen)
		}
	}
	// The write, chain, price, and interface loops must not transform.
	if len(plan.Skipped) == 0 {
		t.Error("expected skipped candidates")
	}
	t.Logf("generated:\n%s", gen)
}

func TestBuildPlanDeterministic(t *testing.T) {
	dir := fixtureDir(t, "proof", "fixture")
	a, err := BuildPlan(context.Background(), Request{Patterns: []string{"./..."}, Dir: dir, ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildPlan(context.Background(), Request{Patterns: []string{"./..."}, Dir: dir, ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Errorf("plan IDs differ: %s vs %s", a.ID, b.ID)
	}
	if a.Digest != b.Digest {
		t.Errorf("plan digests differ")
	}
	if len(a.Files) == 1 && len(b.Files) == 1 {
		if a.Files[0].TransformedDigest != b.Files[0].TransformedDigest {
			t.Errorf("transformed bytes not deterministic")
		}
	}
}
