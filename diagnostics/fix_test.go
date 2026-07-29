package diagnostics

import (
	"errors"
	"testing"
)

func TestFixValidate(t *testing.T) {
	t.Parallel()
	edit := TextEdit{Range: Range{Start: Position{File: "a", Line: 1, Column: 1}, End: Position{File: "a", Line: 1, Column: 5}}, NewText: "max_size"}
	if err := (Fix{Message: "rename field", Edits: []TextEdit{edit}}).Validate(); err != nil {
		t.Errorf("valid fix error: %v", err)
	}
	if err := (Fix{Edits: []TextEdit{edit}}).Validate(); !errors.Is(err, ErrInvalidFix) {
		t.Errorf("empty-message fix error = %v, want ErrInvalidFix", err)
	}
}

func TestFixOverlappingEdits(t *testing.T) {
	t.Parallel()
	a := TextEdit{Range: Range{Start: Position{File: "f", Line: 1, Column: 1}, End: Position{File: "f", Line: 1, Column: 10}}, NewText: "x"}
	b := TextEdit{Range: Range{Start: Position{File: "f", Line: 1, Column: 5}, End: Position{File: "f", Line: 1, Column: 15}}, NewText: "y"}
	if err := (Fix{Message: "m", Edits: []TextEdit{a, b}}).Validate(); !errors.Is(err, ErrInvalidFix) {
		t.Errorf("overlapping edits error = %v, want ErrInvalidFix", err)
	}
}

func TestFixTouchingEditsAllowed(t *testing.T) {
	t.Parallel()
	// Edit b starts exactly where a ends: touching but not overlapping.
	a := TextEdit{Range: Range{Start: Position{File: "f", Line: 1, Column: 1}, End: Position{File: "f", Line: 1, Column: 5}}, NewText: "x"}
	b := TextEdit{Range: Range{Start: Position{File: "f", Line: 1, Column: 5}, End: Position{File: "f", Line: 1, Column: 9}}, NewText: "y"}
	if err := (Fix{Message: "m", Edits: []TextEdit{a, b}}).Validate(); err != nil {
		t.Errorf("touching edits should be allowed, got %v", err)
	}
}

func TestFixEditsDifferentFiles(t *testing.T) {
	t.Parallel()
	a := TextEdit{Range: Range{Start: Position{File: "f1", Line: 1, Column: 1}, End: Position{File: "f1", Line: 1, Column: 10}}, NewText: "x"}
	b := TextEdit{Range: Range{Start: Position{File: "f2", Line: 1, Column: 1}, End: Position{File: "f2", Line: 1, Column: 10}}, NewText: "y"}
	if err := (Fix{Message: "m", Edits: []TextEdit{a, b}}).Validate(); err != nil {
		t.Errorf("edits in different files should not overlap, got %v", err)
	}
}
