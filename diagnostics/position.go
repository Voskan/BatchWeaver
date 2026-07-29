package diagnostics

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidPosition is returned when a Position or Range fails validation.
var ErrInvalidPosition = errors.New("invalid position")

// Position identifies a location in a source file.
//
// Line and Column are one-based when present; a zero value means that component
// is unknown. Offset is a zero-based byte offset when present; a zero Offset is
// ambiguous with "start of file" and is therefore only meaningful alongside a
// known Line. Negative values are always invalid. File paths shown to users
// should be repository-relative; absolute paths must not appear in canonical
// machine-readable output unless explicitly requested.
type Position struct {
	// File is the source file path, if known.
	File string
	// Offset is the zero-based byte offset, if known.
	Offset int
	// Line is the one-based line number, or 0 if unknown.
	Line int
	// Column is the one-based column number, or 0 if unknown.
	Column int
}

// IsZero reports whether the position carries no location information.
func (p Position) IsZero() bool {
	return p.File == "" && p.Offset == 0 && p.Line == 0 && p.Column == 0
}

// Validate reports whether the position is well-formed. Negative components are
// invalid, and a Column without a Line is invalid because a column is
// meaningless on its own.
func (p Position) Validate() error {
	if p.Offset < 0 || p.Line < 0 || p.Column < 0 {
		return fmt.Errorf("%w: negative component in %+v", ErrInvalidPosition, p)
	}
	if p.Column > 0 && p.Line == 0 {
		return fmt.Errorf("%w: column without line in %+v", ErrInvalidPosition, p)
	}
	return nil
}

// before reports whether p is strictly before other within the same file, using
// line then column. It returns false when the two positions are in different
// files or are not comparable.
func (p Position) before(other Position) bool {
	if p.File != other.File {
		return false
	}
	if p.Line != other.Line {
		return p.Line < other.Line
	}
	return p.Column < other.Column
}

// String renders the position as "file:line:column", omitting unknown
// components. A zero Position renders as "-".
func (p Position) String() string {
	if p.IsZero() {
		return "-"
	}
	var b strings.Builder
	if p.File != "" {
		b.WriteString(p.File)
	}
	if p.Line > 0 {
		if b.Len() > 0 {
			b.WriteByte(':')
		}
		b.WriteString(strconv.Itoa(p.Line))
		if p.Column > 0 {
			b.WriteByte(':')
			b.WriteString(strconv.Itoa(p.Column))
		}
	}
	return b.String()
}

// Range is a span between two positions. When only a single point is known,
// Start and End may be equal, or End may be zero.
type Range struct {
	// Start is the beginning of the range.
	Start Position
	// End is the end of the range; it may be zero when unknown.
	End Position
}

// IsZero reports whether the range carries no location information.
func (r Range) IsZero() bool {
	return r.Start.IsZero() && r.End.IsZero()
}

// Validate reports whether the range is well-formed: both endpoints are valid
// and, when both are in the same comparable file, End is not before Start.
func (r Range) Validate() error {
	if err := r.Start.Validate(); err != nil {
		return fmt.Errorf("range start: %w", err)
	}
	if err := r.End.Validate(); err != nil {
		return fmt.Errorf("range end: %w", err)
	}
	if !r.End.IsZero() && r.End.before(r.Start) {
		return fmt.Errorf("%w: end %s is before start %s", ErrInvalidPosition, r.End, r.Start)
	}
	return nil
}

// File returns the file of the range's start position.
func (r Range) File() string { return r.Start.File }

// String renders the range using its start position.
func (r Range) String() string { return r.Start.String() }

// AtPosition returns a Range that begins and ends at p.
func AtPosition(p Position) Range { return Range{Start: p, End: p} }
