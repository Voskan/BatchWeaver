package cli

import (
	"context"
	"flag"
	"fmt"
	"sort"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/internal/buildinfo"
	"github.com/Voskan/BatchWeaver/internal/proof"
	"github.com/Voskan/BatchWeaver/operation"
)

// newProveCommand returns the "prove" command, which evaluates semantic
// eligibility for every discovered candidate without modifying source.
func newProveCommand() *Command {
	return &Command{
		Name:    "prove",
		Summary: "Prove semantic batching eligibility for discovered candidates",
		Usage:   "prove [--format text|json] [--reproducible] [--strategy id] [--decision state] [--candidate id] [--max-candidates n] [--fail-on-unproven] [packages]",
		Run:     runProve,
	}
}

// proofInputs runs analysis and loads operation contracts, then returns a proof
// report for the requested packages.
func proofInputs(ctx context.Context, patterns []string, reproducible bool, limits proof.Limits) (*proof.Report, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	snap, _, err := sharedAnalyze(ctx, analysis.Request{
		Patterns: patterns, Reproducible: reproducible, ToolVersion: buildinfo.Get().Version,
	})
	if err != nil {
		return nil, err
	}
	specs := map[string]operation.Spec{}
	contractDigest := ""
	if res, cErr := loadConfigFromPath(ctx, ""); cErr == nil && res.Found && !res.HasErrors() {
		for _, sp := range res.Catalog.List() {
			specs[sp.ID().String()] = sp
		}
		contractDigest = res.Digest
	}
	return proof.Prove(ctx, proof.Input{
		Snapshot: snap, Specs: specs, ContractDigest: contractDigest,
		Reproducible: reproducible, ToolVersion: buildinfo.Get().Version, Limits: limits,
	})
}

func runProve(ctx context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("prove", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	format := fs.String("format", "text", "output format: text or json")
	reproducible := fs.Bool("reproducible", false, "omit volatile fields for byte-stable output")
	strategy := fs.String("strategy", "", "list only candidates eligible for this strategy")
	decision := fs.String("decision", "", "list only candidates with this decision")
	candidate := fs.String("candidate", "", "list only this candidate ID")
	maxCand := fs.Int("max-candidates", 0, "maximum candidates to prove (0 = default budget)")
	failOnUnproven := fs.Bool("fail-on-unproven", false, "exit nonzero if any candidate is not proven eligible")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *format != "text" && *format != "json" {
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown format %q (want text or json)", *format)}
	}
	if *strategy != "" {
		if _, ok := proof.Strategy(*strategy); !ok {
			return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown strategy %q (see `batchweaver strategy list`)", *strategy)}
		}
	}

	report, err := proofInputs(ctx, fs.Args(), *reproducible, proof.Limits{MaxCandidates: *maxCand})
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}

	if *format == "json" {
		if err := proof.RenderJSON(app.Stdout(), report); err != nil {
			return err
		}
	} else {
		if err := proof.RenderText(app.Stdout(), report); err != nil {
			return err
		}
		if *strategy != "" || *decision != "" || *candidate != "" {
			printFilteredCandidates(app, report, *strategy, *decision, *candidate)
		}
	}

	if *failOnUnproven && report.DecisionCounts[string(proof.DecisionProvenEligible)] != report.Candidates {
		return &CommandError{Code: ExitConfigInvalid}
	}
	return nil
}

// printFilteredCandidates lists candidate IDs matching the active filters.
func printFilteredCandidates(app *App, report *proof.Report, strategy, decision, candidate string) {
	var matched []proof.CandidateProof
	for _, c := range report.CandidateProofs {
		if candidate != "" && c.ID != candidate {
			continue
		}
		if decision != "" && string(c.Decision) != decision {
			continue
		}
		if strategy != "" && !candidateHasEligibleStrategy(c, strategy) {
			continue
		}
		matched = append(matched, c)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].ID < matched[j].ID })
	fmt.Fprintln(app.Stdout(), "\nMatching candidates:")
	if len(matched) == 0 {
		fmt.Fprintln(app.Stdout(), "  (none)")
		return
	}
	for _, c := range matched {
		fmt.Fprintf(app.Stdout(), "  %s  %s  %s\n", c.ID, c.Decision, c.Operation)
	}
}

func candidateHasEligibleStrategy(c proof.CandidateProof, strategy string) bool {
	for _, s := range c.AllowedStrategies {
		if s.Strategy == strategy && s.Status == proof.DecisionProvenEligible {
			return true
		}
	}
	return false
}

// newCandidateCommand returns the "candidate" command.
func newCandidateCommand() *Command {
	return &Command{
		Name:    "candidate",
		Summary: "Inspect a proven or rejected candidate",
		Usage:   "candidate inspect <candidate-id> [packages]",
		Run:     runCandidate,
	}
}

func runCandidate(ctx context.Context, app *App, args []string) error {
	if len(args) < 2 || args[0] != "inspect" {
		return &CommandError{Code: ExitUsage, Message: "usage: candidate inspect <candidate-id> [packages]"}
	}
	id := args[1]
	report, err := proofInputs(ctx, args[2:], false, proof.Limits{})
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	c, ok := report.FindCandidate(id)
	if !ok {
		return &CommandError{Code: ExitConfigNotFound, Message: fmt.Sprintf("candidate %q not found", id)}
	}
	proof.RenderCandidate(app.Stdout(), c)
	return nil
}

// newProofCommand returns the "proof" command (inspect/explain/graph).
func newProofCommand() *Command {
	return &Command{
		Name:    "proof",
		Summary: "Inspect, explain, or graph a proof certificate",
		Usage:   "proof inspect|explain|graph <proof-id> [--obligation id] [--format dot|json] [packages]",
		Run:     runProof2,
	}
}

func runProof2(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "usage: proof inspect|explain|graph <proof-id>"}
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "inspect":
		if len(rest) == 0 {
			return &CommandError{Code: ExitUsage, Message: "proof inspect requires a proof ID"}
		}
		report, err := proofInputs(ctx, rest[1:], false, proof.Limits{})
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		c, ok := report.FindProof(rest[0])
		if !ok {
			return &CommandError{Code: ExitConfigNotFound, Message: fmt.Sprintf("proof %q not found", rest[0])}
		}
		proof.RenderCandidate(app.Stdout(), c)
		return nil
	case "explain":
		if len(rest) == 0 {
			return &CommandError{Code: ExitUsage, Message: "proof explain requires a proof ID"}
		}
		proofID := rest[0]
		fs := flag.NewFlagSet("proof explain", flag.ContinueOnError)
		fs.SetOutput(app.Stderr())
		obligation := fs.String("obligation", "", "obligation ID or alias to explain")
		if err := fs.Parse(rest[1:]); err != nil {
			return &CommandError{Code: ExitUsage}
		}
		report, err := proofInputs(ctx, fs.Args(), false, proof.Limits{})
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		c, ok := report.FindProof(proofID)
		if !ok {
			return &CommandError{Code: ExitConfigNotFound, Message: fmt.Sprintf("proof %q not found", proofID)}
		}
		if *obligation == "" {
			proof.RenderCandidate(app.Stdout(), c)
			return nil
		}
		if err := proof.ExplainObligation(app.Stdout(), c, *obligation); err != nil {
			return &CommandError{Code: ExitConfigNotFound, Message: err.Error()}
		}
		return nil
	case "graph":
		if len(rest) == 0 {
			return &CommandError{Code: ExitUsage, Message: "proof graph requires a proof ID"}
		}
		proofID := rest[0]
		fs := flag.NewFlagSet("proof graph", flag.ContinueOnError)
		fs.SetOutput(app.Stderr())
		format := fs.String("format", "dot", "output format: dot or json")
		if err := fs.Parse(rest[1:]); err != nil {
			return &CommandError{Code: ExitUsage}
		}
		if *format != "dot" && *format != "json" {
			return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown format %q (want dot or json)", *format)}
		}
		report, err := proofInputs(ctx, fs.Args(), false, proof.Limits{})
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		c, ok := report.FindProof(proofID)
		if !ok {
			return &CommandError{Code: ExitConfigNotFound, Message: fmt.Sprintf("proof %q not found", proofID)}
		}
		if *format == "json" {
			return proof.RenderJSON(app.Stdout(), &proof.Report{CandidateProofs: []proof.CandidateProof{c}, SchemaVersion: proof.SchemaVersion})
		}
		proof.RenderDOT(app.Stdout(), c)
		return nil
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown proof subcommand %q", sub)}
	}
}

// newAssumptionCommand returns the "assumption" command.
func newAssumptionCommand() *Command {
	return &Command{
		Name:    "assumption",
		Summary: "List assumptions required by candidates",
		Usage:   "assumption list [packages]",
		Run:     runAssumption,
	}
}

func runAssumption(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "list" {
		return &CommandError{Code: ExitUsage, Message: "usage: assumption list [packages]"}
	}
	report, err := proofInputs(ctx, args[1:], false, proof.Limits{})
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	proof.RenderAssumptions(app.Stdout(), report)
	return nil
}

// newStrategyCommand returns the "strategy" command.
func newStrategyCommand() *Command {
	return &Command{
		Name:    "strategy",
		Summary: "List and inspect transformation strategies",
		Usage:   "strategy list | strategy inspect <strategy-id>",
		Run:     runStrategy,
	}
}

func runStrategy(_ context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] == "list" {
		proof.RenderStrategies(app.Stdout())
		return nil
	}
	if args[0] == "inspect" && len(args) >= 2 {
		s, ok := proof.Strategy(args[1])
		if !ok {
			return &CommandError{Code: ExitConfigNotFound, Message: fmt.Sprintf("strategy %q not found", args[1])}
		}
		fmt.Fprintf(app.Stdout(), "Strategy: %s\nTitle: %s\n\n%s\n\nRequired obligations:\n", s.ID, s.Title, s.Summary)
		for _, o := range s.Required {
			spec, _ := proof.Obligation(o)
			fmt.Fprintf(app.Stdout(), "  %s  %s\n", o, spec.Title)
		}
		return nil
	}
	return &CommandError{Code: ExitUsage, Message: "usage: strategy list | strategy inspect <strategy-id>"}
}
