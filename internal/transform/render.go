package transform

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

// RenderPlanJSON writes the plan as deterministic, indented JSON.
func RenderPlanJSON(w io.Writer, plan *Plan) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(plan)
}

// RenderPlanText writes the human-readable transformation-plan summary.
func RenderPlanText(w io.Writer, plan *Plan) error {
	fmt.Fprintln(w, "BatchWeaver transformation plan")
	fmt.Fprintln(w)
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "Workspace:\t%s\n", plan.Workspace)
	fmt.Fprintf(tw, "Analysis digest:\t%s\n", short(plan.AnalysisDigest))
	fmt.Fprintf(tw, "Requested strategy:\t%s\n", StrategyStaticLoopPrefetch)
	fmt.Fprintf(tw, "Planned transformations:\t%d\n", len(plan.Transformations))
	if err := tw.Flush(); err != nil {
		return err
	}

	if len(plan.Skipped) > 0 {
		counts := map[string]int{}
		for _, s := range plan.Skipped {
			counts[s.Reason]++
		}
		fmt.Fprintln(w, "\nSkipped:")
		stw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		reasons := make([]string, 0, len(counts))
		for r := range counts {
			reasons = append(reasons, r)
		}
		sort.Strings(reasons)
		for _, r := range reasons {
			fmt.Fprintf(stw, "  %s:\t%d\n", r, counts[r])
		}
		if err := stw.Flush(); err != nil {
			return err
		}
	}

	var ins, rem int
	for _, f := range plan.Files {
		ins += f.InsertedLines
		rem += f.RemovedLines
	}
	fmt.Fprintf(w, "\nFiles replaced through overlay: %d\n", len(plan.Files))
	fmt.Fprintf(w, "Net source lines: inserted %d, removed %d\n", ins, rem)

	fmt.Fprintln(w, "\nValidation:")
	vtw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintf(vtw, "  parse:\t%s\n", plan.Validation.Parse)
	fmt.Fprintf(vtw, "  type check:\t%s\n", plan.Validation.TypeCheck)
	fmt.Fprintf(vtw, "  proof preconditions:\t%s\n", plan.Validation.Preconditions)
	fmt.Fprintf(vtw, "  structural verification:\t%s\n", plan.Validation.Structural)
	if err := vtw.Flush(); err != nil {
		return err
	}

	if len(plan.Diagnostics) > 0 {
		fmt.Fprintln(w, "\nDiagnostics:")
		for _, d := range plan.Diagnostics {
			fmt.Fprintf(w, "  %s %s: %s\n", d.Severity, d.Code, d.Message)
		}
	}

	fmt.Fprintln(w, "\nNo source files were changed.")
	fmt.Fprintf(w, "\nPlan:\n  %s\n", plan.ID)
	if len(plan.Transformations) > 0 {
		fmt.Fprintf(w, "\nInspect:\n  batchweaver transform inspect %s\n", plan.ID)
	}
	return nil
}

// RenderInspect writes a detailed inspection of a plan, optionally filtered to
// one candidate.
func RenderInspect(w io.Writer, plan *Plan, candidate string) {
	fmt.Fprintf(w, "Plan: %s\n", plan.ID)
	fmt.Fprintf(w, "Strategy version: %s\n", plan.StrategyVersion)
	for _, tr := range plan.Transformations {
		if candidate != "" && tr.CandidateID != candidate {
			continue
		}
		fmt.Fprintf(w, "\nTransformation:\n  %s\n", tr.ID)
		fmt.Fprintf(w, "Candidate:\n  %s\n", tr.CandidateID)
		fmt.Fprintf(w, "Proof certificate:\n  %s\n", tr.CertificateID)
		fmt.Fprintf(w, "Strategy:\n  %s\n", tr.Strategy)
		fmt.Fprintf(w, "Operation:\n  %s\n", tr.Operation)
		fmt.Fprintf(w, "Source:\n  %s:%d:%d-%d:%d\n", tr.Source.File,
			tr.Source.StartLine, tr.Source.StartCol, tr.Source.EndLine, tr.Source.EndCol)
		fmt.Fprintf(w, "Source snapshot:\n  %s\n", tr.Source.Resolution)
		fmt.Fprintln(w, "Generated phases:")
		for i, p := range tr.Phases {
			fmt.Fprintf(w, "  %d. %s\n", i+1, p)
		}
		fmt.Fprintln(w, "Generated symbols:")
		for _, s := range tr.GeneratedSymbols {
			fmt.Fprintf(w, "  %s\n", s)
		}
		if len(tr.Assumptions) > 0 {
			fmt.Fprintln(w, "Assumptions:")
			for _, a := range tr.Assumptions {
				fmt.Fprintf(w, "  - %s\n", a)
			}
		}
	}
	fmt.Fprintln(w, "\nNo source files were changed.")
}

func short(digest string) string {
	if len(digest) > 20 {
		return digest[:20]
	}
	return digest
}
