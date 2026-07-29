package diagnostics

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// TextFormatter renders diagnostics as deterministic, compiler-style text.
//
// Output has no color by default. The Color field records a capability for
// future use; this release never emits color codes so that output stays
// deterministic and safe for pipes and golden tests.
type TextFormatter struct {
	// Color records a future capability and currently has no effect.
	Color bool
}

// NewTextFormatter returns a TextFormatter with color disabled.
func NewTextFormatter() TextFormatter { return TextFormatter{} }

// Format writes the collection to w in sorted order. Each diagnostic is written
// as a single primary line followed by indented detail and related lines. The
// collection is sorted before writing, so output is independent of insertion
// order.
func (f TextFormatter) Format(w io.Writer, c Collection) error {
	bw := bufio.NewWriter(w)
	for _, d := range c.Sorted().diags {
		if err := f.writeOne(bw, d); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// writeOne renders a single diagnostic.
func (f TextFormatter) writeOne(w *bufio.Writer, d Diagnostic) error {
	if !d.Range.IsZero() {
		if _, err := fmt.Fprintf(w, "%s: ", d.Range.String()); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "%s %s: %s\n", d.Severity.String(), d.Code, d.Message); err != nil {
		return err
	}
	if d.Details != "" {
		if err := writeIndented(w, d.Details); err != nil {
			return err
		}
	}
	for _, rel := range d.Related {
		line := rel.Message
		if !rel.Range.IsZero() {
			line = fmt.Sprintf("%s (%s)", rel.Message, rel.Range.String())
		}
		if err := writeIndented(w, line); err != nil {
			return err
		}
	}
	return nil
}

// writeIndented writes text with each line indented by two spaces.
func writeIndented(w *bufio.Writer, text string) error {
	for _, line := range strings.Split(text, "\n") {
		if _, err := fmt.Fprintf(w, "  %s\n", line); err != nil {
			return err
		}
	}
	return nil
}
