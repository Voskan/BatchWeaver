package diagnostics

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidDiagnostic is returned by Diagnostic.Validate when a diagnostic is
// missing required fields or carries an invalid severity.
var ErrInvalidDiagnostic = errors.New("invalid diagnostic")

// Position identifies a location in a source file. Line and Column are
// 1-based; a zero Line or Column means that component is unknown. An empty
// File means the diagnostic is not tied to a specific file.
type Position struct {
	// File is the path of the source file, if known.
	File string
	// Line is the 1-based line number, or 0 if unknown.
	Line int
	// Column is the 1-based column number, or 0 if unknown.
	Column int
}

// IsZero reports whether the position carries no location information.
func (p Position) IsZero() bool {
	return p.File == "" && p.Line == 0 && p.Column == 0
}

// String renders the position in the conventional "file:line:column" form,
// omitting components that are unknown. A zero Position renders as "-".
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

// Diagnostic is a single finding produced by BatchWeaver tooling. Values are
// immutable by convention: construct a Diagnostic and pass it by value rather
// than mutating fields after creation.
type Diagnostic struct {
	// Code is the reserved-format identifier ("BWxxxx"), if assigned.
	Code string
	// Severity classifies the finding's importance.
	Severity Severity
	// Message is a concise, human-readable summary. It is required.
	Message string
	// Position locates the finding in source, if applicable.
	Position Position
	// Details optionally elaborates on the finding.
	Details string
}

// Validate reports whether the diagnostic is well-formed. A valid diagnostic
// has a valid severity and a non-empty message. Errors wrap
// ErrInvalidDiagnostic so callers can match with errors.Is.
func (d Diagnostic) Validate() error {
	if !d.Severity.Valid() {
		return fmt.Errorf("%w: severity %q is not valid", ErrInvalidDiagnostic, d.Severity)
	}
	if strings.TrimSpace(d.Message) == "" {
		return fmt.Errorf("%w: message is empty", ErrInvalidDiagnostic)
	}
	return nil
}

// String renders the diagnostic in a stable, single-line form:
//
//	<position>: <severity>: [<code>] <message>
//
// The position is omitted when unknown, and the code is omitted when unset.
func (d Diagnostic) String() string {
	var b strings.Builder
	if !d.Position.IsZero() {
		b.WriteString(d.Position.String())
		b.WriteString(": ")
	}
	b.WriteString(d.Severity.String())
	b.WriteString(": ")
	if d.Code != "" {
		b.WriteByte('[')
		b.WriteString(d.Code)
		b.WriteString("] ")
	}
	b.WriteString(d.Message)
	return b.String()
}
