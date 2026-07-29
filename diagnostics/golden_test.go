package diagnostics

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden reports whether golden files should be rewritten. Set
// UPDATE_GOLDEN=1 to regenerate; golden files are never rewritten silently.
func updateGolden() bool { return os.Getenv("UPDATE_GOLDEN") == "1" }

// checkGolden compares got against the golden file named under testdata/golden,
// rewriting it when UPDATE_GOLDEN=1.
func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if updateGolden() {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with UPDATE_GOLDEN=1 to create): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output does not match golden %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

// sampleCollection builds a representative collection used by formatter goldens.
func sampleCollection() Collection {
	var c Collection
	c.Add(Diagnostic{
		Code:     "BWCFG021",
		Severity: SeverityError,
		Message:  `unknown field "max_batch_items"`,
		Details:  `did you mean "max_size"?`,
		Source:   "config",
		Range:    Range{Start: Position{File: "batchweaver.yaml", Line: 27, Column: 5}},
	})
	c.Add(Diagnostic{
		Code:     "BWCFG012",
		Severity: SeverityError,
		Message:  "configuration include cycle detected",
		Source:   "config",
		Range:    Range{Start: Position{File: "a.yaml", Line: 1, Column: 1}},
		Related: []RelatedInformation{
			{Message: "a.yaml includes b.yaml", Range: Range{Start: Position{File: "a.yaml", Line: 2, Column: 3}}},
			{Message: "b.yaml includes a.yaml", Range: Range{Start: Position{File: "b.yaml", Line: 2, Column: 3}}},
		},
	})
	c.Add(Diagnostic{
		Code:     "BWOP005",
		Severity: SeverityWarning,
		Message:  "operation declares reorderable results",
		Source:   "operation",
		Range:    Range{Start: Position{File: "batchweaver.yaml", Line: 40, Column: 7}},
	})
	return c
}
