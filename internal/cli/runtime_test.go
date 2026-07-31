package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inRTFixture changes into the runtime-lowering fixture module, which depends on
// BatchWeaver through a local replace directive.
func inRTFixture(t *testing.T) {
	t.Helper()
	fixture, err := filepath.Abs(filepath.Join("..", "..", "testdata", "transform", "rtfixture"))
	if err != nil {
		t.Fatal(err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(fixture); err != nil {
		t.Fatalf("chdir: %v", err)
	}
}

func TestTransformPlanRuntimeStrategy(t *testing.T) {
	inRTFixture(t)
	t.Cleanup(func() { _, _, _ = run(t, "transform", "clean") })
	code, stdout, stderr := run(t, "transform", "plan", "--strategy", "runtime-call-coalescing", "--format", "json", "./...")
	if code != ExitOK {
		t.Fatalf("exit = %d stderr=%s", code, stderr)
	}
	var plan struct {
		Transformations []struct {
			Operation string `json:"operation"`
			Strategy  string `json:"strategy"`
			Bridge    string `json:"bridge"`
		} `json:"transformations"`
		Files []struct {
			Path    string `json:"path"`
			Created bool   `json:"created"`
		} `json:"files"`
		Validation struct {
			TypeCheck string `json:"type_check"`
		} `json:"validation"`
	}
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatalf("plan JSON: %v", err)
	}
	if plan.Validation.TypeCheck != "passed" {
		t.Fatalf("type check = %q", plan.Validation.TypeCheck)
	}
	if len(plan.Transformations) == 0 {
		t.Fatal("expected runtime transformations")
	}
	for _, tr := range plan.Transformations {
		if tr.Strategy != "runtime-call-coalescing" || tr.Bridge == "" {
			t.Errorf("unexpected transformation %+v", tr)
		}
	}
	created := false
	for _, f := range plan.Files {
		if f.Created && strings.HasSuffix(f.Path, "_gen.go") {
			created = true
		}
	}
	if !created {
		t.Error("expected a created bridge file")
	}
}

func TestTransformPlanBadStrategy(t *testing.T) {
	t.Parallel()
	code, _, stderr := run(t, "transform", "plan", "--strategy", "no-such-strategy")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "unknown strategy") {
		t.Errorf("stderr: %s", stderr)
	}
}

func TestRuntimeInspect(t *testing.T) {
	inRTFixture(t)
	t.Cleanup(func() { _, _, _ = run(t, "transform", "clean") })
	code, out, stderr := run(t, "runtime", "inspect", "./...")
	if code != ExitOK {
		t.Fatalf("exit = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(out, "users.get") {
		t.Errorf("runtime inspect missing operation:\n%s", out)
	}
}

func TestBarrierInspect(t *testing.T) {
	inRTFixture(t)
	t.Cleanup(func() { _, _, _ = run(t, "transform", "clean") })
	code, out, _ := run(t, "barrier", "inspect", "./...")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "barrier") {
		t.Errorf("barrier inspect output:\n%s", out)
	}
}

func TestToolExecDoctorAndExplain(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "tool-exec", "doctor")
	if code != ExitOK || !strings.Contains(out, "doctor") {
		t.Errorf("doctor exit=%d out=%s", code, out)
	}
	code, out, _ = run(t, "tool-exec", "explain")
	if code != ExitOK || !strings.Contains(out, "overlay") {
		t.Errorf("explain exit=%d out=%s", code, out)
	}
}

func TestToolexecNoArgs(t *testing.T) {
	t.Parallel()
	code, _, _ := run(t, "toolexec")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestToolexecDelegatesExitCode(t *testing.T) {
	t.Parallel()
	// Delegating to a tool that exits non-zero must surface as ExitGoCommand,
	// never an internal error.
	false2, err := exec2LookPath()
	if err != nil {
		t.Skip("no false(1) available")
	}
	code, _, _ := run(t, "toolexec", false2)
	if code != ExitGoCommand {
		t.Errorf("exit = %d, want %d", code, ExitGoCommand)
	}
}

// exec2LookPath returns a path to a program that exits non-zero.
func exec2LookPath() (string, error) {
	for _, p := range []string{"/usr/bin/false", "/bin/false"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}
