package diagnostics

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// ErrInvalidDiagnostic is returned by Diagnostic.Validate for malformed
// diagnostics. Errors from related helpers may also wrap it.
var ErrInvalidDiagnostic = errors.New("invalid diagnostic")

// RelatedInformation points to a secondary source location that helps explain a
// diagnostic, such as a previous definition or another edge of an include
// cycle.
type RelatedInformation struct {
	// Message describes the related location; it must be non-empty.
	Message string
	// Range locates the related information.
	Range Range
}

// Validate reports whether the related information is well-formed.
func (r RelatedInformation) Validate() error {
	if strings.TrimSpace(r.Message) == "" {
		return fmt.Errorf("%w: related information message is empty", ErrInvalidDiagnostic)
	}
	if err := r.Range.Validate(); err != nil {
		return fmt.Errorf("%w: related information range: %w", ErrInvalidDiagnostic, err)
	}
	return nil
}

// Diagnostic is a single finding produced by BatchWeaver tooling. Values are
// immutable by convention: construct a Diagnostic and pass it by value rather
// than mutating fields after creation. The Related and Fixes slices are owned by
// the Diagnostic; callers must not retain and mutate a slice passed in.
type Diagnostic struct {
	// Code is the stable diagnostic identifier.
	Code Code
	// Severity classifies the finding's importance.
	Severity Severity
	// Message is a concise, single-line, human-readable summary. Required.
	Message string
	// Range locates the finding in source, if applicable.
	Range Range
	// Details optionally elaborates on the finding across one or more lines.
	Details string
	// Source is a short, stable identifier for the producing subsystem, for
	// example "config" or "operation".
	Source string
	// Related lists secondary source locations.
	Related []RelatedInformation
	// Fixes lists advisory suggested changes.
	Fixes []Fix
}

// Validate reports whether the diagnostic is well-formed. It checks the code,
// severity, message, range, related entries, and fixes, and rejects control
// characters and trailing whitespace in the message.
func (d Diagnostic) Validate() error {
	if err := d.Code.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDiagnostic, err)
	}
	if !d.Severity.Valid() {
		return fmt.Errorf("%w: severity %q is not valid", ErrInvalidDiagnostic, d.Severity)
	}
	if strings.TrimSpace(d.Message) == "" {
		return fmt.Errorf("%w: message is empty", ErrInvalidDiagnostic)
	}
	if d.Message != strings.TrimRight(d.Message, " \t") {
		return fmt.Errorf("%w: message has trailing whitespace", ErrInvalidDiagnostic)
	}
	if hasControlChars(d.Message) {
		return fmt.Errorf("%w: message contains control characters", ErrInvalidDiagnostic)
	}
	if strings.ContainsAny(d.Message, "\n\r") {
		return fmt.Errorf("%w: message must be a single line", ErrInvalidDiagnostic)
	}
	if err := d.Range.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDiagnostic, err)
	}
	for i := range d.Related {
		if err := d.Related[i].Validate(); err != nil {
			return err
		}
	}
	for i := range d.Fixes {
		if err := d.Fixes[i].Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidDiagnostic, err)
		}
	}
	return nil
}

// IsError reports whether the diagnostic has error severity.
func (d Diagnostic) IsError() bool { return d.Severity == SeverityError }

// identity returns the semantic identity used for deduplication. It excludes
// advisory fixes and related information so that two diagnostics describing the
// same problem at the same place deduplicate even if their fixes differ.
func (d Diagnostic) identity() string {
	var b strings.Builder
	b.WriteString(string(d.Code))
	b.WriteByte('\x00')
	b.WriteString(d.Severity.String())
	b.WriteByte('\x00')
	b.WriteString(d.Source)
	b.WriteByte('\x00')
	b.WriteString(d.Range.Start.File)
	b.WriteByte('\x00')
	fmt.Fprintf(&b, "%d\x00%d\x00", d.Range.Start.Line, d.Range.Start.Column)
	b.WriteString(d.Message)
	return b.String()
}

// String renders the diagnostic in a stable single-line form:
//
//	<range>: <severity> <code>: <message>
//
// The range is omitted when unknown.
func (d Diagnostic) String() string {
	var b strings.Builder
	if !d.Range.IsZero() {
		b.WriteString(d.Range.String())
		b.WriteString(": ")
	}
	b.WriteString(d.Severity.String())
	if d.Code != "" {
		b.WriteByte(' ')
		b.WriteString(string(d.Code))
	}
	b.WriteString(": ")
	b.WriteString(d.Message)
	return b.String()
}

// hasControlChars reports whether s contains ASCII control characters other
// than tab, which some details legitimately use for indentation.
func hasControlChars(s string) bool {
	for _, r := range s {
		if r == '\t' {
			continue
		}
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
