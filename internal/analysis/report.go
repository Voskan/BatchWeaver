package analysis

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// RenderJSON writes the snapshot as deterministic, indented JSON with a trailing
// newline. In reproducible mode the caller must have built the snapshot without
// volatile fields.
func RenderJSON(w io.Writer, snap *Snapshot) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(snap)
}

// RenderText writes a concise, human-readable analysis summary. It never claims
// that any discovered structure is safe to transform.
func RenderText(w io.Writer, snap *Snapshot) error {
	scalarCalls := len(snap.CallSites)
	ambiguous := 0
	for _, s := range snap.CallSites {
		if s.Dispatch == DispatchInterface {
			ambiguous++
		}
	}
	invalid := 0
	for _, op := range snap.Operations {
		if op.Compatibility == "invalid" {
			invalid++
		}
	}
	counts := map[string]int{}
	for _, c := range snap.Candidates {
		counts[c.State]++
	}

	fmt.Fprintln(w, "BatchWeaver static analysis")
	fmt.Fprintln(w)
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "Workspace:\t%s\n", snap.Workspace)
	fmt.Fprintf(tw, "Go packages loaded:\t%d\n", snap.PackagesLoaded)
	fmt.Fprintf(tw, "Application packages:\t%d\n", snap.ApplicationPackages)
	fmt.Fprintf(tw, "Test package variants:\t%d\n", snap.TestVariants)
	fmt.Fprintf(tw, "SSA functions indexed:\t%d\n", snap.SSAFunctions)
	fmt.Fprintf(tw, "Call graph edges:\t%d\n", snap.CallGraphEdges)
	fmt.Fprintf(tw, "Declared operations:\t%d\n", len(snap.Operations))
	fmt.Fprintf(tw, "Scalar operation calls:\t%d\n", scalarCalls)
	fmt.Fprintf(tw, "Ambiguous call sites:\t%d\n", ambiguous)
	fmt.Fprintf(tw, "Invalid declarations:\t%d\n", invalid)
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Candidates by structural context:")
	ctw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintf(ctw, "  loops:\t%d\n", counts[StatePotentialLoop])
	fmt.Fprintf(ctw, "  goroutine fan-out:\t%d\n", counts[StatePotentialFanout])
	fmt.Fprintf(ctw, "  sibling calls:\t%d\n", counts[StatePotentialSiblings])
	fmt.Fprintf(ctw, "  direct isolated:\t%d\n", counts[StateDirectIsolated])
	fmt.Fprintf(ctw, "  ambiguous targets:\t%d\n", counts[StateAmbiguousTarget])
	if err := ctw.Flush(); err != nil {
		return err
	}

	if errs := snap.errorDiagnostics(); len(errs) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Diagnostics:")
		for _, d := range errs {
			loc := d.Location
			if loc == "" {
				loc = "-"
			}
			fmt.Fprintf(w, "  %s %s: %s (%s)\n", d.Severity, d.Code, d.Message, loc)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "No source files were changed.")
	fmt.Fprintln(w, "Potential batching structure discovered. Semantic safety has not yet been proven.")
	if snap.Incomplete {
		fmt.Fprintln(w, "Analysis is incomplete; some results may be partial.")
	}
	return nil
}

// errorDiagnostics returns error- and warning-severity diagnostics in order.
func (snap *Snapshot) errorDiagnostics() []Diag {
	var out []Diag
	for _, d := range snap.Diagnostics {
		if d.Severity == "error" || d.Severity == "warning" {
			out = append(out, d)
		}
	}
	return out
}

// HasErrors reports whether the snapshot has any error-severity diagnostic.
func (snap *Snapshot) HasErrors() bool {
	for _, d := range snap.Diagnostics {
		if d.Severity == "error" {
			return true
		}
	}
	return false
}

// HasWarnings reports whether the snapshot has any warning-severity diagnostic.
func (snap *Snapshot) HasWarnings() bool {
	for _, d := range snap.Diagnostics {
		if d.Severity == "warning" {
			return true
		}
	}
	return false
}

// LoadFailed reports whether any package-loading diagnostic is present.
func (snap *Snapshot) LoadFailed() bool {
	for _, d := range snap.Diagnostics {
		if d.Code == "BW3000" {
			return true
		}
	}
	return false
}
