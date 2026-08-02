package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Voskan/BatchWeaver/internal/buildinfo"
	"github.com/Voskan/BatchWeaver/internal/gocommand"
	"github.com/Voskan/BatchWeaver/internal/transform"
)

// newTransformCommand returns the "transform" command.
func newTransformCommand() *Command {
	return &Command{
		Name:    "transform",
		Summary: "Plan, inspect, diff, materialize, and revert transformations",
		Usage:   "transform plan|diff|inspect|verify|materialize|revert|clean|recover [args]",
		Run:     runTransform,
	}
}

func cwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}

// splitDashDash splits args at the first "--" separator into the portion before
// (BatchWeaver flags) and after (arguments forwarded to the Go command).
func splitDashDash(args []string) (before, after []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

func runTransform(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "transform requires a subcommand"}
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "plan":
		return runTransformPlan(ctx, app, rest)
	case "diff":
		return runTransformDiff(ctx, app, rest)
	case "inspect":
		return runTransformInspect(ctx, app, rest)
	case "verify":
		return runTransformVerify(ctx, app, rest)
	case "materialize":
		return runTransformMaterialize(ctx, app, rest)
	case "revert":
		return runTransformRevert(ctx, app, rest)
	case "clean":
		return runTransformClean(app)
	case "recover":
		return runTransformRecover(app)
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown transform subcommand %q", sub)}
	}
}

func planFromArgs(ctx context.Context, args []string, filter transform.Filter, strategies []transform.StrategyID) (*transform.Plan, error) {
	patterns := args
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	return transform.BuildPlan(ctx, transform.Request{
		Patterns: patterns, Dir: cwd(), ToolVersion: buildinfo.Get().Version,
		Toolchain: buildinfo.Get().GoVersion, Filter: filter, Strategies: strategies,
	})
}

// knownStrategies is the closed set of transformation strategies the CLI accepts.
var knownStrategies = map[string]transform.StrategyID{
	"static-loop-prefetch":    transform.StrategyStaticLoopPrefetch,
	"runtime-call-coalescing": transform.StrategyRuntimeCallCoalescing,
	"static-sibling-fusion":   transform.StrategyStaticSiblingFusion,
	"fanout-coalescing":       transform.StrategyFanoutCoalescing,
	"errgroup-coalescing":     transform.StrategyErrgroupCoalescing,
}

// parseStrategies parses a comma-separated strategy list, rejecting unknown IDs.
func parseStrategies(s string) ([]transform.StrategyID, error) {
	if s == "" {
		return nil, nil
	}
	var out []transform.StrategyID
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, ok := knownStrategies[part]
		if !ok {
			return nil, fmt.Errorf("unknown strategy %q", part)
		}
		out = append(out, id)
	}
	return out, nil
}

func runTransformPlan(ctx context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("transform plan", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	format := fs.String("format", "text", "output format: text or json")
	candidate := fs.String("candidate", "", "only this candidate ID")
	operation := fs.String("operation", "", "only this operation ID")
	file := fs.String("file", "", "only candidates in this file")
	strategy := fs.String("strategy", "", "comma-separated strategies (default: static-loop-prefetch)")
	max := fs.Int("max-transformations", 0, "maximum transformations (0 = unlimited)")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *format != "text" && *format != "json" {
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown format %q", *format)}
	}
	strategies, serr := parseStrategies(*strategy)
	if serr != nil {
		return &CommandError{Code: ExitUsage, Message: serr.Error()}
	}
	plan, err := planFromArgs(ctx, fs.Args(), transform.Filter{Candidate: *candidate, Operation: *operation, File: *file, Max: *max}, strategies)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	if plan.Validation.TypeCheck == transform.ValidationFailed {
		if *format == "json" {
			_ = transform.RenderPlanJSON(app.Stdout(), plan)
		} else {
			_ = transform.RenderPlanText(app.Stdout(), plan)
		}
		return &CommandError{Code: ExitError}
	}
	root, err := transform.ModuleRoot(cwd())
	if err == nil {
		_ = transform.SavePlan(root, plan)
	}
	if *format == "json" {
		return transform.RenderPlanJSON(app.Stdout(), plan)
	}
	return transform.RenderPlanText(app.Stdout(), plan)
}

func loadOrPlan(ctx context.Context, args []string) (*transform.Plan, error) {
	root, err := transform.ModuleRoot(cwd())
	if err != nil {
		return nil, err
	}
	if len(args) > 0 && looksLikePlanID(args[0]) {
		return transform.LoadPlan(root, args[0])
	}
	return planFromArgs(ctx, args, transform.Filter{}, nil)
}

func looksLikePlanID(s string) bool {
	return len(s) > len("bwplan_") && s[:len("bwplan_")] == "bwplan_"
}

func runTransformDiff(ctx context.Context, app *App, args []string) error {
	plan, err := loadOrPlan(ctx, args)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	fmt.Fprint(app.Stdout(), transform.PlanDiff(plan, transform.DefaultDiffContext))
	return nil
}

func runTransformInspect(ctx context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("transform inspect", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	candidate := fs.String("candidate", "", "restrict to one candidate")
	// Allow the plan ID as the first positional before flags.
	var planArg []string
	if len(args) > 0 && looksLikePlanID(args[0]) {
		planArg = args[:1]
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	plan, err := loadOrPlan(ctx, append(planArg, fs.Args()...))
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	transform.RenderInspect(app.Stdout(), plan, *candidate)
	return nil
}

func runTransformVerify(ctx context.Context, app *App, args []string) error {
	root, err := transform.ModuleRoot(cwd())
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	if len(args) == 0 || !looksLikePlanID(args[0]) {
		return &CommandError{Code: ExitUsage, Message: "verify requires a plan ID"}
	}
	stored, err := transform.LoadPlan(root, args[0])
	if err != nil {
		return &CommandError{Code: ExitStale, Message: err.Error()}
	}
	if stored.ProofSchema == "batchweaver.sql-proof/v1alpha1" {
		if err := transform.VerifySQLSynthesisPlan(stored); err != nil {
			fmt.Fprintf(app.Stdout(), "VERIFY FAILED\n  %v\n", err)
			return &CommandError{Code: ExitStale}
		}
		fmt.Fprintf(app.Stdout(), "VERIFY PASS\n  SQL synthesis plan %s\n  generated content, source map, structure, and canonical digest match\n", stored.ID)
		return nil
	}
	fresh, err := planFromArgs(ctx, args[1:], transform.Filter{}, nil)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	if fresh.Digest != stored.Digest {
		fmt.Fprintf(app.Stdout(), "VERIFY FAILED\n  stored plan digest differs from a fresh plan\n  stored: %s\n  fresh:  %s\n", stored.ID, fresh.ID)
		return &CommandError{Code: ExitStale}
	}
	if fresh.Validation.TypeCheck != transform.ValidationPassed {
		fmt.Fprintf(app.Stdout(), "VERIFY FAILED\n  %s\n", fresh.Validation.Detail)
		return &CommandError{Code: ExitError}
	}
	fmt.Fprintf(app.Stdout(), "VERIFY PASS\n  plan %s\n  parse, type check, and deterministic digest match\n", stored.ID)
	return nil
}

func runTransformMaterialize(_ context.Context, app *App, args []string) error {
	root, err := transform.ModuleRoot(cwd())
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "materialize requires a plan ID"}
	}
	plan, err := transform.LoadPlan(root, args[0])
	if err != nil {
		return &CommandError{Code: ExitStale, Message: err.Error()}
	}
	res, err := transform.Materialize(root, buildinfo.Get().Version, plan)
	if err != nil {
		return &CommandError{Code: ExitMaterialize, Message: err.Error()}
	}
	fmt.Fprintf(app.Stdout(), "Materialization complete\n\nFiles changed:\n  %d\n\nMaterialization ID:\n  %s\n\nRevert:\n  batchweaver transform revert %s\n",
		res.FilesChanged, res.MaterializationID, res.MaterializationID)
	return nil
}

func runTransformRevert(_ context.Context, app *App, args []string) error {
	root, err := transform.ModuleRoot(cwd())
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "revert requires a materialization ID"}
	}
	res, err := transform.Revert(root, args[0])
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	if len(res.Conflicts) > 0 {
		fmt.Fprintf(app.Stdout(), "Revert conflict: %d file(s) changed after materialization and were not overwritten:\n", len(res.Conflicts))
		for _, c := range res.Conflicts {
			fmt.Fprintf(app.Stdout(), "  %s\n", c)
		}
		return &CommandError{Code: ExitMaterialize}
	}
	fmt.Fprintf(app.Stdout(), "Revert complete\n\nFiles restored:\n  %d\n", res.FilesRestored)
	return nil
}

func runTransformClean(app *App) error {
	root, err := transform.ModuleRoot(cwd())
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	if err := transform.CleanPlans(root); err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	fmt.Fprintln(app.Stdout(), "Transformation plan cache cleaned.")
	return nil
}

func runTransformRecover(app *App) error {
	root, err := transform.ModuleRoot(cwd())
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	statuses, err := transform.Recover(root)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	if len(statuses) == 0 {
		fmt.Fprintln(app.Stdout(), "No materializations to recover.")
		return nil
	}
	for _, s := range statuses {
		fmt.Fprintf(app.Stdout(), "%s  state=%s committed=%d/%d\n", s.MaterializationID, s.State, s.CommittedFiles, s.TotalFiles)
	}
	return nil
}

// newBuildCommand, newTestCommand, newRunCommand implement transformed Go
// command wrappers.
func newBuildCommand() *Command {
	return &Command{Name: "build", Summary: "Build transformed packages through an overlay", Usage: "build [go build args]", Run: transformedGo("build")}
}

func newTestCommand() *Command {
	return &Command{Name: "test", Summary: "Test transformed packages through an overlay", Usage: "test [go test args]", Run: transformedGo("test")}
}

func newRunCommand() *Command {
	return &Command{Name: "run", Summary: "Run transformed code through an overlay", Usage: "run <pkg> [-- app args]", Run: transformedGo("run")}
}

// transformedGo returns a command that plans the whole module, builds an overlay,
// and delegates to the given Go subcommand, preserving its exit code.
func transformedGo(sub string) func(context.Context, *App, []string) error {
	return func(ctx context.Context, app *App, args []string) error {
		root, err := transform.ModuleRoot(cwd())
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		// Separate BatchWeaver flags (before "--") from Go arguments (after).
		bwArgs, goArgs := splitDashDash(args)
		fs := flag.NewFlagSet(sub, flag.ContinueOnError)
		fs.SetOutput(app.Stderr())
		strategy := fs.String("strategy", "", "comma-separated strategies (default: static-loop-prefetch)")
		if err := fs.Parse(bwArgs); err != nil {
			return &CommandError{Code: ExitUsage}
		}
		// Any positional BatchWeaver args before "--" join the Go arguments so the
		// common form `batchweaver build ./cmd/x` keeps working.
		goArgs = append(fs.Args(), goArgs...)
		strategies, serr := parseStrategies(*strategy)
		if serr != nil {
			return &CommandError{Code: ExitUsage, Message: serr.Error()}
		}
		plan, err := transform.BuildPlan(ctx, transform.Request{
			Patterns: []string{"./..."}, Dir: cwd(), ToolVersion: buildinfo.Get().Version,
			Toolchain: buildinfo.Get().GoVersion, Strategies: strategies,
		})
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		if plan.Validation.TypeCheck == transform.ValidationFailed {
			return &CommandError{Code: ExitError, Message: "transformed packages do not type-check: " + plan.Validation.Detail}
		}
		overlay := ""
		if len(plan.Files) > 0 {
			p, _, oerr := transform.EnsureSavedOverlay(root, plan)
			if oerr != nil {
				return &CommandError{Code: ExitError, Message: oerr.Error()}
			}
			overlay = p
		}
		runner := gocommand.Runner{Dir: cwd(), Stdout: app.Stdout(), Stderr: app.Stderr()}
		code, err := runner.Run(ctx, sub, overlay, goArgs)
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		if code != 0 {
			return &CommandError{Code: ExitGoCommand}
		}
		return nil
	}
}
