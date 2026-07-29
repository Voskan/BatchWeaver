package proof

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

// decisionOrder is the stable display order of decisions in summaries.
var decisionOrder = []Decision{
	DecisionProvenEligible, DecisionProvenIneligible,
	DecisionRequiresAssumption, DecisionUnknown, DecisionDeferred,
}

var decisionLabels = map[Decision]string{
	DecisionProvenEligible:     "proven eligible",
	DecisionProvenIneligible:   "proven ineligible",
	DecisionRequiresAssumption: "requires explicit assumption",
	DecisionUnknown:            "unknown",
	DecisionDeferred:           "deferred",
}

// RenderJSON writes the report as deterministic, indented JSON with a trailing
// newline.
func RenderJSON(w io.Writer, r *Report) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// RenderText writes the human-readable proof summary.
func RenderText(w io.Writer, r *Report) error {
	fmt.Fprintln(w, "BatchWeaver semantic proof")
	fmt.Fprintln(w)
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "Workspace:\t%s\n", r.Workspace)
	fmt.Fprintf(tw, "Declared operations:\t%d\n", r.DeclaredOperations)
	fmt.Fprintf(tw, "Operation call sites:\t%d\n", r.OperationCallSites)
	fmt.Fprintf(tw, "Structural candidates:\t%d\n", r.Candidates)
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Proof outcomes:")
	otw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	for _, d := range decisionOrder {
		fmt.Fprintf(otw, "  %s:\t%d\n", decisionLabels[d], r.DecisionCounts[string(d)])
	}
	if err := otw.Flush(); err != nil {
		return err
	}

	if len(r.StrategyCounts) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Eligible strategies:")
		stw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		for _, s := range Strategies() {
			if n := r.StrategyCounts[s.ID]; n > 0 {
				fmt.Fprintf(stw, "  %s:\t%d\n", s.Title, n)
			}
		}
		if err := stw.Flush(); err != nil {
			return err
		}
	}

	if len(r.Assumptions) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Required assumptions: %d (none applied automatically)\n", len(r.Assumptions))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "No source files were changed.")
	fmt.Fprintln(w, "Run `batchweaver candidate inspect <candidate-id>` for evidence.")
	return nil
}

// FindCandidate returns the candidate proof with the given candidate ID.
func (r *Report) FindCandidate(id string) (CandidateProof, bool) {
	for _, c := range r.CandidateProofs {
		if c.ID == id {
			return c, true
		}
	}
	return CandidateProof{}, false
}

// FindProof returns the candidate proof with the given proof ID.
func (r *Report) FindProof(proofID string) (CandidateProof, bool) {
	for _, c := range r.CandidateProofs {
		if c.ProofID == proofID {
			return c, true
		}
	}
	return CandidateProof{}, false
}

// RenderCandidate writes the detailed inspection of a single candidate proof.
func RenderCandidate(w io.Writer, c CandidateProof) {
	fmt.Fprintf(w, "Candidate: %s\n", c.ID)
	if c.Location != "" {
		fmt.Fprintf(w, "Location: %s\n", redact(c.Location))
	}
	fmt.Fprintf(w, "Operation: %s\n", c.Operation)
	fmt.Fprintf(w, "Structure: %s\n\n", c.Structure)

	fmt.Fprintln(w, "Decision:")
	fmt.Fprintf(w, "  %s\n", strings.ToUpper(strings.ReplaceAll(string(c.Decision), "_", " ")))

	if elig := eligibleStrategies(c); len(elig) > 0 {
		fmt.Fprintln(w, "\nAllowed strategies:")
		for _, s := range elig {
			fmt.Fprintf(w, "  - %s\n", s)
		}
	}

	fmt.Fprintf(w, "\nProof certificate:\n  %s\n", c.ProofID)

	if c.Decision == DecisionProvenEligible {
		fmt.Fprintln(w, "\nSatisfied obligations:")
		for _, o := range c.Obligations {
			if o.Status == ObligationSatisfied {
				fmt.Fprintf(w, "  \u2713 %s\n", redact(o.Summary))
			}
		}
	} else {
		renderBlocking(w, c)
	}

	if len(c.Assumptions) > 0 {
		fmt.Fprintln(w, "\nAssumptions:")
		for _, a := range c.Assumptions {
			fmt.Fprintf(w, "  - %s\n", assumptionText(a))
		}
	}
	if len(c.Limitations) > 0 {
		fmt.Fprintln(w, "\nNot guaranteed:")
		for _, l := range c.Limitations {
			fmt.Fprintf(w, "  - %s\n", redact(l))
		}
	}
	fmt.Fprintln(w, "\nNo source files were changed.")
}

// renderBlocking prints the failed or unknown obligation and any witness for a
// non-eligible candidate.
func renderBlocking(w io.Writer, c CandidateProof) {
	var blocking *ObligationResult
	for i := range c.Obligations {
		o := c.Obligations[i]
		if o.Status == ObligationViolated {
			blocking = &c.Obligations[i]
			break
		}
	}
	if blocking == nil {
		for i := range c.Obligations {
			if c.Obligations[i].Status == ObligationUnknown {
				blocking = &c.Obligations[i]
				break
			}
		}
	}
	if blocking == nil {
		return
	}
	label := "Failed obligation:"
	if blocking.Status == ObligationUnknown {
		label = "Unknown obligation:"
	}
	fmt.Fprintf(w, "\n%s\n  %s\n  %s\n", label, blocking.ID, redact(blocking.Summary))
	if blocking.Witness != "" {
		for _, wit := range c.Witnesses {
			if wit.ID == blocking.Witness {
				fmt.Fprintln(w, "\nWitness:")
				fmt.Fprintf(w, "  %s\n", redact(wit.Summary))
				for _, s := range wit.Steps {
					fmt.Fprintf(w, "    %s\n", redact(s))
				}
			}
		}
	}
	// Report the independent runtime coalescing eligibility, which is often the
	// most actionable alternative for a rejected static candidate.
	for _, s := range c.AllowedStrategies {
		if s.Strategy == StrategyRuntimeScopeCoalescing {
			fmt.Fprintf(w, "\nRuntime coalescing eligibility:\n  %s\n",
				strings.ReplaceAll(string(s.Status), "_", " "))
		}
	}
}

// ExplainObligation writes a focused explanation of one obligation within a
// proof certificate.
func ExplainObligation(w io.Writer, c CandidateProof, obligationID string) error {
	var found *ObligationResult
	for i := range c.Obligations {
		if c.Obligations[i].ID == obligationID || obligationAlias(c.Obligations[i], obligationID) {
			found = &c.Obligations[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("obligation %q is not part of proof %s", obligationID, c.ProofID)
	}
	spec, _ := Obligation(found.ID)
	fmt.Fprintf(w, "Obligation: %s\n", spec.Title)
	fmt.Fprintf(w, "Identifier: %s\n", found.ID)
	fmt.Fprintf(w, "Family: %s\n", found.Family)
	fmt.Fprintf(w, "Status: %s\n\n", found.Status)
	fmt.Fprintf(w, "Summary:\n  %s\n", redact(found.Summary))
	if len(found.Evidence) > 0 {
		fmt.Fprintln(w, "\nEvidence:")
		for _, e := range found.Evidence {
			fmt.Fprintf(w, "  %s: %s", e.Kind, redact(e.Detail))
			if e.Location != "" {
				fmt.Fprintf(w, " (%s)", redact(e.Location))
			}
			fmt.Fprintln(w)
		}
	}
	if found.Witness != "" {
		for _, wit := range c.Witnesses {
			if wit.ID == found.Witness {
				fmt.Fprintln(w, "\nWitness:")
				fmt.Fprintf(w, "  %s\n", redact(wit.Summary))
				for _, s := range wit.Steps {
					fmt.Fprintf(w, "    %s\n", redact(s))
				}
			}
		}
	}
	return nil
}

// obligationAlias allows a short alias (e.g. "error-order") to match an
// obligation by family or a suffix of its ID.
func obligationAlias(o ObligationResult, alias string) bool {
	a := strings.ToLower(alias)
	switch a {
	case "error-order", "error":
		return o.ID == OblFirstErrorOrder
	case "order":
		return o.Family == FamilyOrder
	case "key":
		return o.ID == OblKeyIndependent
	case "receiver":
		return o.ID == OblReceiverInvariant
	case "context":
		return o.ID == OblContextInvariant
	case "target":
		return o.ID == OblTargetResolved
	default:
		return false
	}
}

// RenderDOT writes the evidence graph of a candidate proof as a DOT digraph. It
// escapes all labels.
func RenderDOT(w io.Writer, c CandidateProof) {
	fmt.Fprintf(w, "digraph proof {\n  label=%q;\n  rankdir=LR;\n", c.ProofID)
	fmt.Fprintf(w, "  candidate [shape=box,label=%q];\n", dotEscape(c.ID+"\n"+string(c.Decision)))
	for i, o := range c.Obligations {
		node := fmt.Sprintf("o%d", i)
		fmt.Fprintf(w, "  %s [label=%q];\n", node, dotEscape(o.ID+"\n"+string(o.Status)))
		edge := "supports"
		if o.Status == ObligationViolated {
			edge = "contradicts"
		}
		fmt.Fprintf(w, "  %s -> candidate [label=%q];\n", node, edge)
	}
	fmt.Fprintln(w, "}")
}

// RenderAssumptions writes the required-assumptions listing.
func RenderAssumptions(w io.Writer, r *Report) {
	fmt.Fprintln(w, "Required assumptions")
	fmt.Fprintln(w)
	if len(r.Assumptions) == 0 {
		fmt.Fprintln(w, "None. Every proven decision is derived from inferred evidence.")
		return
	}
	for _, a := range r.Assumptions {
		fmt.Fprintf(w, "%s  %s\n", a.ID, redact(a.Text))
		fmt.Fprintf(w, "  requested by candidates: %d\n", a.RequestedByCount)
		fmt.Fprintln(w, "  declaration source: none")
		fmt.Fprintln(w, "  suggested action: add a scoped operation contract or leave candidates unknown")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "No assumption has been applied automatically.")
}

// RenderStrategies writes the strategy registry listing.
func RenderStrategies(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "STRATEGY\tDESCRIPTION")
	for _, s := range Strategies() {
		fmt.Fprintf(tw, "%s\t%s\n", s.ID, s.Summary)
	}
	_ = tw.Flush()
}

// eligibleStrategies returns the human titles of eligible strategies, sorted by
// canonical rank.
func eligibleStrategies(c CandidateProof) []string {
	var out []string
	for _, s := range c.AllowedStrategies {
		if s.Status == DecisionProvenEligible {
			out = append(out, s.Strategy)
		}
	}
	sort.Slice(out, func(i, j int) bool { return strategyRank(out[i]) < strategyRank(out[j]) })
	return out
}

// assumptionText renders a known assumption ID as prose.
func assumptionText(id string) string {
	if id == builtinRaceFree {
		return "the analyzed program is free of data races"
	}
	return id
}

// dotEscape escapes a label for DOT output.
func dotEscape(s string) string {
	s = strings.ReplaceAll(redact(s), "\"", "\\\"")
	return s
}
