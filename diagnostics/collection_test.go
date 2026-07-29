package diagnostics

import "testing"

func mkDiag(code Code, sev Severity, file string, line, col int, msg string) Diagnostic {
	return Diagnostic{
		Code:     code,
		Severity: sev,
		Message:  msg,
		Source:   "test",
		Range:    Range{Start: Position{File: file, Line: line, Column: col}},
	}
}

func TestCollectionBasics(t *testing.T) {
	t.Parallel()
	var c Collection
	if !c.Empty() || c.Len() != 0 || c.HasErrors() {
		t.Fatalf("zero collection not empty")
	}
	c.Add(mkDiag("BWCFG001", SeverityWarning, "a", 1, 1, "w"))
	c.Add(mkDiag("BWCFG002", SeverityError, "a", 2, 1, "e"))
	if c.Len() != 2 || c.Empty() {
		t.Fatalf("len = %d", c.Len())
	}
	if !c.HasErrors() {
		t.Errorf("HasErrors = false")
	}
	if c.Count(SeverityError) != 1 || c.Count(SeverityWarning) != 1 {
		t.Errorf("counts wrong")
	}
}

func TestCollectionDiagnosticsCopy(t *testing.T) {
	t.Parallel()
	var c Collection
	c.Add(mkDiag("BWCFG001", SeverityInfo, "a", 1, 1, "m"))
	got := c.Diagnostics()
	got[0].Message = "mutated"
	if c.Diagnostics()[0].Message != "m" {
		t.Errorf("Diagnostics() exposed internal storage")
	}
}

func TestCollectionSorted(t *testing.T) {
	t.Parallel()
	var c Collection
	c.Add(mkDiag("BWCFG002", SeverityWarning, "b.yaml", 1, 1, "b1"))
	c.Add(mkDiag("BWCFG001", SeverityError, "a.yaml", 2, 3, "a-err"))
	c.Add(mkDiag("BWCFG003", SeverityInfo, "a.yaml", 2, 1, "a-info"))
	sorted := c.Sorted().Diagnostics()
	// a.yaml before b.yaml; within a.yaml, line 2 col 1 before col 3.
	wantOrder := []string{"a-info", "a-err", "b1"}
	for i, w := range wantOrder {
		if sorted[i].Message != w {
			t.Errorf("sorted[%d] = %q, want %q", i, sorted[i].Message, w)
		}
	}
	// Original not mutated.
	if c.Diagnostics()[0].Message != "b1" {
		t.Errorf("Sorted mutated the receiver")
	}
}

func TestCollectionSortSeverityTiebreak(t *testing.T) {
	t.Parallel()
	var c Collection
	c.Add(mkDiag("BWCFG002", SeverityInfo, "a", 1, 1, "info"))
	c.Add(mkDiag("BWCFG001", SeverityError, "a", 1, 1, "error"))
	sorted := c.Sorted().Diagnostics()
	if sorted[0].Message != "error" {
		t.Errorf("errors should sort before info at same position; got %q first", sorted[0].Message)
	}
}

func TestCollectionDeduplicated(t *testing.T) {
	t.Parallel()
	var c Collection
	d := mkDiag("BWCFG001", SeverityError, "a", 1, 1, "same")
	c.Add(d)
	c.Add(d)
	c.Add(mkDiag("BWCFG001", SeverityError, "a", 1, 1, "different"))
	dedup := c.Deduplicated()
	if dedup.Len() != 2 {
		t.Errorf("Deduplicated len = %d, want 2", dedup.Len())
	}
}

func TestCollectionFilters(t *testing.T) {
	t.Parallel()
	var c Collection
	c.Add(Diagnostic{Code: "BWCFG001", Severity: SeverityError, Message: "a", Source: "config"})
	c.Add(Diagnostic{Code: "BWOP001", Severity: SeverityWarning, Message: "b", Source: "operation"})
	if got := c.FilterBySeverity(SeverityError).Len(); got != 1 {
		t.Errorf("FilterBySeverity = %d", got)
	}
	if got := c.FilterBySource("operation").Len(); got != 1 {
		t.Errorf("FilterBySource = %d", got)
	}
	if got := c.FilterByCodePrefix("BWCFG").Len(); got != 1 {
		t.Errorf("FilterByCodePrefix = %d", got)
	}
}

func TestCollectionAddCollection(t *testing.T) {
	t.Parallel()
	var a, b Collection
	a.Add(mkDiag("BWCFG001", SeverityError, "a", 1, 1, "a"))
	b.Add(mkDiag("BWCFG002", SeverityInfo, "b", 1, 1, "b"))
	a.AddCollection(b)
	if a.Len() != 2 {
		t.Errorf("AddCollection len = %d, want 2", a.Len())
	}
	// Appending more to b must not affect a.
	b.Add(mkDiag("BWCFG003", SeverityInfo, "c", 1, 1, "c"))
	if a.Len() != 2 {
		t.Errorf("a shares storage with b: len = %d", a.Len())
	}
}
