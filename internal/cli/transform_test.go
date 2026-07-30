package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTransformPlanFixture(t *testing.T) {
	inProofFixture(t)
	t.Cleanup(func() { _, _, _ = run(t, "transform", "clean") })
	code, stdout, stderr := run(t, "transform", "plan", "--format", "json", "./...")
	if code != ExitOK {
		t.Fatalf("exit = %d stderr=%s", code, stderr)
	}
	var plan struct {
		ID              string `json:"id"`
		Transformations []struct {
			Operation string `json:"operation"`
			Strategy  string `json:"strategy"`
		} `json:"transformations"`
		Validation struct {
			TypeCheck string `json:"type_check"`
		} `json:"validation"`
	}
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("plan JSON: %v", err)
	}
	if plan.Validation.TypeCheck != "passed" {
		t.Errorf("type check = %q", plan.Validation.TypeCheck)
	}
	if len(plan.Transformations) != 1 || plan.Transformations[0].Operation != "users.get" {
		t.Fatalf("transformations = %+v", plan.Transformations)
	}
	if !strings.HasPrefix(plan.ID, "bwplan_") {
		t.Errorf("plan ID = %q", plan.ID)
	}
}

func TestTransformDiffAndInspect(t *testing.T) {
	inProofFixture(t)
	t.Cleanup(func() { _, _, _ = run(t, "transform", "clean") })
	_, planOut, _ := run(t, "transform", "plan", "--format", "json", "./...")
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(planOut), &p); err != nil {
		t.Fatal(err)
	}
	code, diff, stderr := run(t, "transform", "diff", p.ID)
	if code != ExitOK {
		t.Fatalf("diff exit = %d stderr=%s", code, stderr)
	}
	for _, want := range []string{"--- a/svc/svc.go", "+++ b/svc/svc.go", "bwKeys", "GetUsersBatch"} {
		if !strings.Contains(diff, want) {
			t.Errorf("diff missing %q:\n%s", want, diff)
		}
	}
	code, insp, _ := run(t, "transform", "inspect", p.ID)
	if code != ExitOK {
		t.Fatalf("inspect exit = %d", code)
	}
	if !strings.Contains(insp, "static-loop-prefetch") || !strings.Contains(insp, "No source files were changed.") {
		t.Errorf("inspect output:\n%s", insp)
	}
}

func TestTransformVerify(t *testing.T) {
	inProofFixture(t)
	t.Cleanup(func() { _, _, _ = run(t, "transform", "clean") })
	_, planOut, _ := run(t, "transform", "plan", "--format", "json", "./...")
	var p struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(planOut), &p)
	code, out, _ := run(t, "transform", "verify", p.ID)
	if code != ExitOK {
		t.Fatalf("verify exit = %d out=%s", code, out)
	}
	if !strings.Contains(out, "VERIFY PASS") {
		t.Errorf("verify output: %s", out)
	}
}

func TestTransformPlanDeterministicID(t *testing.T) {
	inProofFixture(t)
	t.Cleanup(func() { _, _, _ = run(t, "transform", "clean") })
	_, a, _ := run(t, "transform", "plan", "--format", "json", "./...")
	_, b, _ := run(t, "transform", "plan", "--format", "json", "./...")
	if a != b {
		t.Error("transform plan JSON is not deterministic")
	}
}

func TestTransformUnknownSubcommand(t *testing.T) {
	t.Parallel()
	code, _, _ := run(t, "transform", "frobnicate")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
}
