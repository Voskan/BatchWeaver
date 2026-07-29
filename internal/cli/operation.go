package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"text/tabwriter"

	"github.com/Voskan/BatchWeaver/config"
	"github.com/Voskan/BatchWeaver/diagnostics"
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
	case "-h", "--help", "help":
		fmt.Fprintln(app.Stdout(), "Usage:\n  batchweaver operation list [--file path] [--format text|json]")
		return nil
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown operation subcommand %q", args[0])}
	}
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
