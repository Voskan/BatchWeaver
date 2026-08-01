package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileCollectAndInspect(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "svc.bwp")
	code, out, stderr := run(t, "profile", "collect", "--output", path, "--count", "2000", "--rate", "8000")
	if code != ExitOK {
		t.Fatalf("collect exit = %d stderr=%s", code, stderr)
	}
	for _, want := range []string{"Logical calls", "No raw operation keys were stored", "Profile bundle"} {
		if !strings.Contains(out, want) {
			t.Errorf("collect output missing %q:\n%s", want, out)
		}
	}
	code, out, _ = run(t, "profile", "inspect", "--profile", path)
	if code != ExitOK {
		t.Fatalf("inspect exit = %d", code)
	}
	if !strings.Contains(out, "users.get") {
		t.Errorf("inspect missing operation:\n%s", out)
	}
}

func TestProfileValidateAndRedact(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "svc.bwp")
	if code, _, _ := run(t, "profile", "collect", "--output", path); code != ExitOK {
		t.Fatalf("collect failed")
	}
	if code, _, _ := run(t, "profile", "validate", "--profile", path); code != ExitOK {
		t.Fatalf("validate should pass for a fresh profile")
	}
	code, out, _ := run(t, "profile", "redact", "--profile", path)
	if code != ExitOK || !strings.Contains(out, "No raw operation keys") {
		t.Errorf("redact output unexpected: %s", out)
	}
}

func TestTuneAnalyzeJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "svc.bwp")
	if code, _, _ := run(t, "profile", "collect", "--output", path); code != ExitOK {
		t.Fatalf("collect failed")
	}
	code, out, stderr := run(t, "tune", "analyze", "--profile", path, "--format", "json")
	if code != ExitOK {
		t.Fatalf("analyze exit = %d stderr=%s", code, stderr)
	}
	for _, want := range []string{`"objective"`, `"controller_version"`, "users.get"} {
		if !strings.Contains(out, want) {
			t.Errorf("json report missing %q", want)
		}
	}
}

func TestTuneReplayCompares(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "tune", "replay")
	if code != ExitOK {
		t.Fatalf("replay exit = %d", code)
	}
	for _, want := range []string{"current", "latency", "throughput", "backend_calls"} {
		if !strings.Contains(out, want) {
			t.Errorf("replay output missing %q:\n%s", want, out)
		}
	}
}

func TestFairnessReport(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "fairness", "report")
	if code != ExitOK {
		t.Fatalf("fairness exit = %d", code)
	}
	if !strings.Contains(out, "share=") || !strings.Contains(out, "anonymized") {
		t.Errorf("fairness output unexpected:\n%s", out)
	}
}

func TestOverloadInspectCritical(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "overload", "inspect", "--queue", "0.99")
	if code != ExitOK {
		t.Fatalf("overload exit = %d", code)
	}
	if !strings.Contains(out, "critical") {
		t.Errorf("expected critical state:\n%s", out)
	}
}

func TestWaveGraphDefault(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "wave", "graph")
	if code != ExitOK {
		t.Fatalf("wave exit = %d", code)
	}
	if !strings.Contains(out, "Dispatch waves") || !strings.Contains(out, "Critical path") {
		t.Errorf("wave output unexpected:\n%s", out)
	}
}

func TestRecursiveInspect(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "recursive", "inspect")
	if code != ExitOK {
		t.Fatalf("recursive exit = %d", code)
	}
	if !strings.Contains(out, "frontier sizes") {
		t.Errorf("recursive output unexpected:\n%s", out)
	}
}
