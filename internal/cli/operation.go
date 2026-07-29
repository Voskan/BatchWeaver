package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"text/tabwriter"

	"github.com/Voskan/BatchWeaver/config"
	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/internal/buildinfo"
)

// operationListSchemaVersion versions the JSON output of "operation list".
const operationListSchemaVersion = 1

// newOperationCommand returns the "operation" command.
func newOperationCommand() *Command {
	return &Command{
		Name:    "operation",
		Summary: "Inspect declared operations",
		Usage:   "operation list [--file path] [--format text|json]",
		Run:     runOperation,
	}
}

func runOperation(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "operation requires a subcommand: list"}
	}
	switch args[0] {
	case "list":
		return runOperationList(ctx, app, args[1:])
	case "inspect":
		return runOperationInspect(ctx, app, args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(app.Stdout(), "Usage:\n  batchweaver operation list [--file path] [--format text|json]\n  batchweaver operation inspect <operation-id> [packages]")
		return nil
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown operation subcommand %q", args[0])}
	}
}

// runOperationInspect runs static analysis and prints one operation's discovery
// details. It never claims that any call site is safe to transform.
func runOperationInspect(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "operation inspect requires an operation ID"}
	}
	id := args[0]
	patterns := args[1:]
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	snap, err := analysis.Analyze(ctx, analysis.Request{Patterns: patterns, ToolVersion: buildinfo.Get().Version})
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	for _, op := range snap.Operations {
		if op.ID != id {
			continue
		}
		fmt.Fprintf(app.Stdout(), "Operation: %s\n\n", op.ID)
		fmt.Fprintln(app.Stdout(), "Declaration sources:")
		for _, s := range op.Sources {
			fmt.Fprintf(app.Stdout(), "  %s: %s\n", s.Kind, s.Location)
		}
		fmt.Fprintf(app.Stdout(), "\nScalar symbol: %s\nBatch symbol:  %s\nCompatibility: %s\n",
			orDash(op.ScalarSymbol), orDash(op.BatchSymbol), op.Compatibility)
		fmt.Fprintf(app.Stdout(), "\nDiscovered scalar call sites: %d\n", len(op.CallSiteIDs))
		printStructural(app, snap, op.ID)
		fmt.Fprintln(app.Stdout(), "\nStatus:\n  discovered and indexed\n  semantic batching safety has not yet been proven")
		return nil
	}
	return &CommandError{Code: ExitConfigNotFound, Message: fmt.Sprintf("operation %q not found", id)}
}

// printStructural prints candidate structural context counts for an operation.
func printStructural(app *App, snap *analysis.Snapshot, opID string) {
	counts := map[string]int{}
	for _, c := range snap.Candidates {
		if c.Operation == opID {
			counts[c.State]++
		}
	}
	if len(counts) == 0 {
		return
	}
	fmt.Fprintln(app.Stdout(), "\nStructural contexts:")
	for _, st := range []string{"potential_loop", "potential_fanout", "potential_siblings", "direct_isolated", "ambiguous_target"} {
		if counts[st] > 0 {
			fmt.Fprintf(app.Stdout(), "  %s: %d\n", st, counts[st])
		}
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

type operationRow struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Result   string `json:"result_mode"`
	Scope    string `json:"scope"`
	MaxSize  int    `json:"max_size"`
	MaxWait  string `json:"max_wait"`
	Fallback string `json:"fallback"`
}

type operationListJSON struct {
	SchemaVersion int            `json:"schema_version"`
	Operations    []operationRow `json:"operations"`
}

func runOperationList(ctx context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("operation list", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("file", "", "configuration file path (discovered when omitted)")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *format != "text" && *format != "json" {
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown format %q (want text or json)", *format)}
	}

	res, err := loadConfigFromPath(ctx, *file)
	if err != nil {
		return err
	}
	if code := loadExitCode(res); code != ExitOK {
		if err := diagnostics.NewTextFormatter().Format(app.Stderr(), res.Diagnostics); err != nil {
			return err
		}
		return &CommandError{Code: code}
	}

	rows := make([]operationRow, 0, res.Catalog.Len())
	for _, sp := range res.Catalog.List() { // sorted by ID
		rows = append(rows, operationRow{
			ID:       sp.ID().String(),
			Kind:     sp.Semantics().Kind().String(),
			Result:   sp.ResultContract().Mode().String(),
			Scope:    sp.PartitionContract().Scope().String(),
			MaxSize:  sp.SchedulerPolicy().MaxBatchSize(),
			MaxWait:  config.Duration(sp.SchedulerPolicy().MaxWait()).String(),
			Fallback: sp.FallbackPolicy().Mode().String(),
		})
	}

	if *format == "json" {
		enc := json.NewEncoder(app.Stdout())
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(operationListJSON{SchemaVersion: operationListSchemaVersion, Operations: rows})
	}

	tw := tabwriter.NewWriter(app.Stdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "OPERATION\tKIND\tRESULT\tSCOPE\tMAX SIZE\tMAX WAIT\tFALLBACK")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n", r.ID, r.Kind, r.Result, r.Scope, r.MaxSize, r.MaxWait, r.Fallback)
	}
	return tw.Flush()
}
