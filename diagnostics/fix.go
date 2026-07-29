package diagnostics

import (
	"errors"
	"fmt"
)

// ErrInvalidFix is returned when a Fix or TextEdit fails validation.
var ErrInvalidFix = errors.New("invalid fix")

// TextEdit describes a replacement of the text within Range by NewText. Edits
// are advisory in this release: BatchWeaver does not yet apply them to source.
type TextEdit struct {
	// Range is the span to replace.
	Range Range
	// NewText is the replacement text; it may be empty to represent a deletion.
	NewText string
}

// Validate reports whether the edit has a well-formed range.
func (e TextEdit) Validate() error {
	if err := e.Range.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidFix, err)
	}
	return nil
}

// Fix is an advisory, implementation-independent suggested change consisting of
// a human-readable message and a set of non-overlapping text edits.
type Fix struct {
	// Message describes the fix; it must be non-empty.
	Message string
	// Edits are the text edits that make up the fix; within one fix they must
	// not overlap.
	Edits []TextEdit
}

// Validate reports whether the fix is well-formed: a non-empty message, valid
// edits, and no two edits overlapping within the same file.
func (f Fix) Validate() error {
	if f.Message == "" {
		return fmt.Errorf("%w: message is empty", ErrInvalidFix)
	}
	for i := range f.Edits {
		if err := f.Edits[i].Validate(); err != nil {
			return err
		}
	}
	for i := range f.Edits {
		for j := i + 1; j < len(f.Edits); j++ {
			if editsOverlap(f.Edits[i], f.Edits[j]) {
				return fmt.Errorf("%w: edits %d and %d overlap", ErrInvalidFix, i, j)
			}
		}
	}
	return nil
}

// editsOverlap reports whether two edits target overlapping spans in the same
// file. Edits in different files never overlap.
func editsOverlap(a, b TextEdit) bool {
	if a.Range.File() != b.Range.File() {
		return false
	}
	// Overlap when a starts before b ends and b starts before a ends. A zero end
	// is treated as a point at start.
	aStart, aEnd := a.Range.Start, a.Range.End
	bStart, bEnd := b.Range.Start, b.Range.End
	if aEnd.IsZero() {
		aEnd = aStart
	}
	if bEnd.IsZero() {
		bEnd = bStart
	}
	return !aEnd.before(bStart) && !bEnd.before(aStart) &&
		!positionsEqualExclusive(aEnd, bStart) && !positionsEqualExclusive(bEnd, aStart)
}

// positionsEqualExclusive reports whether end == start at a shared boundary,
// which is treated as touching but not overlapping.
func positionsEqualExclusive(end, start Position) bool {
	return end.File == start.File && end.Line == start.Line && end.Column == start.Column
}
