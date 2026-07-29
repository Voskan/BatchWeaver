package analysis

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
)

// fixtureDir returns the absolute path to the standalone analysis fixture module.
func fixtureDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "testdata", "analysis", "fixture"))
	if err != nil {
		t.Fatalf("fixture path: %v", err)
	}
	return abs
}

func analyzeFixture(t *testing.T) *Snapshot {
	t.Helper()
	snap, err := Analyze(context.Background(), Request{
		Patterns:     []string{"./..."},
		Dir:          fixtureDir(t),
		Reproducible: true,
		ToolVersion:  "test",
	})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	return snap
}

func TestAnalyzeDiscoversOperation(t *testing.T) {
	t.Parallel()
	snap := analyzeFixture(t)
	if snap.LoadFailed() {
		t.Fatalf("fixture failed to load: %+v", snap.Diagnostics)
	}
	if len(snap.Operations) != 1 {
		t.Fatalf("operations = %d, want 1: %+v", len(snap.Operations), snap.Operations)
	}
	op := snap.Operations[0]
	if op.ID != "users.get" {
		t.Errorf("operation id = %q", op.ID)
	}
	if op.Compatibility != "valid" {
		t.Errorf("compatibility = %q, want valid", op.Compatibility)
	}
	if op.ScalarSymbol == "" || op.BatchSymbol == "" {
		t.Errorf("symbols not resolved: %+v", op)
	}
	found := false
	for _, s := range op.Sources {
		if s.Kind == "configuration" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected configuration declaration source")
	}
}

func TestAnalyzeFindsCallSites(t *testing.T) {
	t.Parallel()
	snap := analyzeFixture(t)
	if len(snap.CallSites) < 2 {
		t.Fatalf("call sites = %d, want >= 2 (loop + goroutine): %+v", len(snap.CallSites), snap.CallSites)
	}
	loop, fanout := false, false
	for _, s := range snap.CallSites {
		if s.Operation != "users.get" {
			t.Errorf("unexpected operation on call site: %q", s.Operation)
		}
		if s.LoopDepth > 0 {
			loop = true
		}
		if s.InGoroutine {
			fanout = true
		}
	}
	if !loop {
		t.Errorf("no loop call site found: %+v", snap.CallSites)
	}
	if !fanout {
		t.Errorf("no goroutine call site found: %+v", snap.CallSites)
	}
}

func TestAnalyzeCandidateStates(t *testing.T) {
	t.Parallel()
	snap := analyzeFixture(t)
	states := map[string]bool{}
	for _, c := range snap.Candidates {
		states[c.State] = true
		if len(c.Evidence) == 0 {
			t.Errorf("candidate %s has no evidence", c.ID)
		}
	}
	if !states[StatePotentialLoop] {
		t.Errorf("expected a potential_loop candidate: %+v", snap.Candidates)
	}
	if !states[StatePotentialFanout] {
		t.Errorf("expected a potential_fanout candidate: %+v", snap.Candidates)
	}
}

func TestAnalyzeDeterministicJSON(t *testing.T) {
	t.Parallel()
	var a, b bytes.Buffer
	if err := RenderJSON(&a, analyzeFixture(t)); err != nil {
		t.Fatal(err)
	}
	if err := RenderJSON(&b, analyzeFixture(t)); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Errorf("reproducible JSON is not byte-identical across runs")
	}
	if a.Len() == 0 || a.Bytes()[a.Len()-1] != '\n' {
		t.Errorf("JSON must end with a newline")
	}
}

func TestSnapshotHasSchemaVersion(t *testing.T) {
	t.Parallel()
	snap := analyzeFixture(t)
	if snap.SchemaVersion != SchemaVersion {
		t.Errorf("schema version = %q, want %q", snap.SchemaVersion, SchemaVersion)
	}
	if snap.Timestamp != "" {
		t.Errorf("reproducible snapshot must omit timestamp")
	}
}

func TestStdlibEffect(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"net/http": "network", "os": "filesystem", "time": "time",
		"sync": "synchronization", "reflect": "reflection", "os/exec": "process",
	}
	for path, want := range cases {
		if got, ok := stdlibEffect(path); !ok || got != want {
			t.Errorf("stdlibEffect(%q) = %q, %v; want %q", path, got, ok, want)
		}
	}
	if _, ok := stdlibEffect("strings"); ok {
		t.Errorf("strings should have no modeled effect")
	}
}

func TestPortablePath(t *testing.T) {
	t.Parallel()
	pc := pathContext{root: "/repo"}
	if got := pc.portable("/repo/internal/x.go"); got != "internal/x.go" {
		t.Errorf("portable = %q, want internal/x.go", got)
	}
	if got := pc.portable(""); got != "" {
		t.Errorf("portable of empty = %q", got)
	}
}
