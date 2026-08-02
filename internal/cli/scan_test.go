package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanBadFormat(t *testing.T) {
	t.Parallel()
	code, _, stderr := run(t, "scan", "--format", "xml")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "unknown format") {
		t.Errorf("stderr: %s", stderr)
	}
}

func TestScanBadFailOn(t *testing.T) {
	t.Parallel()
	code, _, _ := run(t, "scan", "--fail-on", "sometimes")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestScanFixtureJSON(t *testing.T) {
	// Not parallel: this test changes the working directory.
	fixture, err := filepath.Abs(filepath.Join("..", "..", "testdata", "analysis", "fixture"))
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

	code, stdout, stderr := run(t, "scan", "--reproducible", "--cache-status", "--fail-on", "never", "--format", "json", "./...")
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr=%s", code, stderr)
	}
	if !strings.Contains(stderr, "analysis cache: source=local hit=false") {
		t.Fatalf("cache status missing: %s", stderr)
	}
	var doc struct {
		SchemaVersion string `json:"schema_version"`
		Operations    []struct {
			ID string `json:"id"`
		} `json:"operations"`
		CallSites []struct {
			Operation string `json:"operation"`
		} `json:"call_sites"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("scan output is not valid JSON: %v\n%s", err, stdout[:clampLen(len(stdout), 300)])
	}
	if doc.SchemaVersion == "" {
		t.Errorf("missing schema_version")
	}
	if len(doc.Operations) != 1 || doc.Operations[0].ID != "users.get" {
		t.Errorf("operations = %+v, want users.get", doc.Operations)
	}
	if len(doc.CallSites) < 2 {
		t.Errorf("call sites = %d, want >= 2", len(doc.CallSites))
	}
}

func clampLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
