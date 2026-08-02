package cli

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"
	"text/tabwriter"

	batchweaver "github.com/Voskan/BatchWeaver"
	"github.com/Voskan/BatchWeaver/internal/adapter"
	"github.com/Voskan/BatchWeaver/internal/transform"
)

// newAdapterCommand returns the "adapter" command.
func newAdapterCommand() *Command {
	return &Command{
		Name:    "adapter",
		Summary: "Inspect backend adapters and proof-gated SQL synthesis",
		Usage:   "adapter list | inspect --sql=... | plan-sql --sql=... --package-name=... --package-path=... --output=... --constant=... --operation=... | explain --sql=... | verify | doctor",
		Run:     runAdapter,
	}
}

func runAdapter(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "adapter requires a subcommand: list, inspect, plan-sql, explain, verify, doctor"}
	}
	switch args[0] {
	case "list":
		return adapterList(app, args[1:])
	case "inspect":
		return adapterInspect(app, args[1:])
	case "plan-sql":
		return adapterPlanSQL(ctx, app, args[1:])
	case "explain":
		return adapterExplain(app, args[1:])
	case "verify":
		return adapterVerify(ctx, app)
	case "doctor":
		return adapterDoctor(app)
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown adapter subcommand %q", args[0])}
	}
}

func adapterPlanSQL(ctx context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("adapter plan-sql", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	sqlText := fs.String("sql", "", "scalar SELECT query to synthesize")
	keyType := fs.String("key-type", "bigint", "SQL type of one key")
	keyTypes := fs.String("key-types", "", "comma-separated SQL types for a composite key")
	joinCardinality := fs.String("join-cardinality", "", "join proof contract: at-most-one")
	packageName := fs.String("package-name", "", "target Go package name")
	packagePath := fs.String("package-path", "", "target Go import path")
	output := fs.String("output", "", "workspace-relative generated .go file")
	constant := fs.String("constant", "", "generated Go query constant")
	operation := fs.String("operation", "", "operation identity recorded in the plan")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *sqlText == "" || *packageName == "" || *packagePath == "" || *output == "" || *constant == "" || *operation == "" {
		return &CommandError{Code: ExitUsage, Message: "adapter plan-sql requires --sql, --package-name, --package-path, --output, --constant, and --operation"}
	}
	synthesis, rejection := synthesizeCLI(*sqlText, *keyType, *keyTypes, *joinCardinality)
	if rejection != nil {
		printRejection(app, rejection)
		return nil
	}
	root, err := transform.ModuleRoot(cwd())
	if err != nil {
		return err
	}
	plan, err := transform.BuildSQLSynthesisPlan(ctx, transform.SQLPlanRequest{
		Workspace: root, PackageName: *packageName, PackagePath: *packagePath,
		Output: *output, Constant: *constant, Operation: *operation, Synthesis: synthesis,
	})
	if err != nil {
		return err
	}
	overlay, count, err := transform.EnsureSavedOverlay(root, plan)
	if err != nil {
		return err
	}
	fmt.Fprintln(app.Stdout(), "SQL synthesis transformation planned")
	fmt.Fprintf(app.Stdout(), "\nPlan:\n  %s\nStrategy:\n  %s\nGenerated file:\n  %s\nOverlay:\n  %s (%d file)\n",
		plan.ID, plan.Transformations[0].Strategy, *output, overlay, count)
	fmt.Fprintln(app.Stdout(), "\nNo workspace source file was written. Use transform inspect/diff/verify/materialize with the saved plan.")
	return nil
}

func adapterList(app *App, args []string) error {
	fs := flag.NewFlagSet("adapter list", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	category := fs.String("category", "", "filter by category: backend or network")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	w := app.Stdout()
	fmt.Fprintln(w, "BatchWeaver adapters")
	for _, m := range adapter.Manifests() {
		if *category != "" && m.Category != *category {
			continue
		}
		fmt.Fprintf(w, "\n%s\n", m.AdapterID)
		fmt.Fprintf(w, "  version:      %d\n", m.Version)
		fmt.Fprintf(w, "  status:       %s\n", m.Status)
		if len(m.Dialects) > 0 {
			fmt.Fprintf(w, "  dialects:     %v\n", m.Dialects)
		}
		if len(m.Clients) > 0 {
			fmt.Fprintf(w, "  clients:      %v\n", m.Clients)
		}
		fmt.Fprintln(w, "  capabilities:")
		for _, c := range m.Capabilities {
			fmt.Fprintf(w, "    %s\n", c)
		}
		if m.Notes != "" {
			fmt.Fprintf(w, "  notes:        %s\n", m.Notes)
		}
	}
	return nil
}

func adapterInspect(app *App, args []string) error {
	fs := flag.NewFlagSet("adapter inspect", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	sqlText := fs.String("sql", "", "scalar SELECT query to synthesize")
	keyType := fs.String("key-type", "bigint", "SQL type of one key (for array synthesis)")
	keyTypes := fs.String("key-types", "", "comma-separated SQL types for a composite key")
	joinCardinality := fs.String("join-cardinality", "", "join proof contract: at-most-one")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *sqlText == "" {
		return &CommandError{Code: ExitUsage, Message: "adapter inspect requires --sql"}
	}
	plan, srej := synthesizeCLI(*sqlText, *keyType, *keyTypes, *joinCardinality)
	if srej != nil {
		printRejection(app, srej)
		return nil
	}
	w := app.Stdout()
	fmt.Fprint(w, "Binding mode:\n  synthesized exact/composite-key read\n\n")
	fmt.Fprintln(w, "Scalar query:")
	fmt.Fprintf(w, "  %s\n\n", *sqlText)
	fmt.Fprintln(w, "Generated batch query:")
	for _, line := range splitLinesLocal(plan.Query) {
		fmt.Fprintf(w, "  %s\n", line)
	}
	fmt.Fprintf(w, "\nResult contract:\n  %s\n", plan.Contract)
	fmt.Fprintf(w, "Missing behavior:\n  %s\n", plan.MissingError)
	return nil
}

func synthesizeCLI(sqlText, keyType, keyTypes, joinCardinality string) (adapter.SynthPlan, *adapter.Rejection) {
	q, rejection := adapter.ParseExactKeySelect(sqlText)
	if rejection != nil {
		return adapter.SynthPlan{}, rejection
	}
	types := []string(nil)
	if keyTypes != "" {
		for _, value := range strings.Split(keyTypes, ",") {
			types = append(types, strings.TrimSpace(value))
		}
	}
	return adapter.SynthesizeExactKey(adapter.SynthInput{
		Query: q, KeyType: keyType, KeyTypes: types,
		JoinCardinality: adapter.JoinCardinality(joinCardinality),
	})
}

func adapterExplain(app *App, args []string) error {
	fs := flag.NewFlagSet("adapter explain", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	sqlText := fs.String("sql", "", "scalar SELECT query to evaluate")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *sqlText == "" {
		return &CommandError{Code: ExitUsage, Message: "adapter explain requires --sql"}
	}
	parsed, rej := adapter.ParseExactKeySelect(*sqlText)
	if rej == nil {
		if parsed.Join != nil {
			fmt.Fprintln(app.Stdout(), "The SQL shape is supported; synthesis requires --join-cardinality=at-most-one evidence.")
			return nil
		}
		fmt.Fprintln(app.Stdout(), "Automatic exact/composite-key SQL batch synthesis is available for this query.")
		return nil
	}
	printRejection(app, rej)
	return nil
}

func printRejection(app *App, rej *adapter.Rejection) {
	w := app.Stdout()
	fmt.Fprintln(w, "Automatic SQL batch synthesis rejected")
	fmt.Fprintf(w, "\nReason:\n  %s\n", rej.Reason)
	fmt.Fprintf(w, "\nDiagnostic:\n  %s\n", rej.Code)
	if rej.Node != "" {
		fmt.Fprintf(w, "\nUnsupported node:\n  %q (offset %d)\n", rej.Node, rej.Offset)
	}
	fmt.Fprintln(w, "\nResolution:\n  provide an explicit batch provider with documented semantics,\n  or keep scalar execution.")
}

// adapterVerify runs a hermetic in-memory contract verification demonstration.
func adapterVerify(ctx context.Context, app *App) error {
	data := map[int]string{1: "alice", 2: "bob", 3: "carol"}
	scalar := func(_ context.Context, k int) (string, error) {
		if v, ok := data[k]; ok {
			return v, nil
		}
		return "", sql.ErrNoRows
	}
	batch := func(_ context.Context, ks []int) ([]batchweaver.Outcome[string], error) {
		out := make([]batchweaver.Outcome[string], len(ks))
		for i, k := range ks {
			if v, ok := data[k]; ok {
				out[i] = batchweaver.Success(batchweaver.RequestID(i+1), v)
			} else {
				out[i] = batchweaver.Failure[string](batchweaver.RequestID(i+1), sql.ErrNoRows)
			}
		}
		return out, nil
	}
	vc := adapter.VerifyReadOnly(ctx, "users.get", "database/sql", scalar, batch,
		func(a, b string) bool { return a == b },
		[]adapter.VerifyCase[int]{
			{Name: "unique keys", Keys: []int{1, 2, 3}},
			{Name: "duplicate keys", Keys: []int{1, 1, 2}},
			{Name: "missing key", Keys: []int{1, 9}},
			{Name: "empty input", Keys: nil},
			{Name: "one key", Keys: []int{2}},
		})
	w := app.Stdout()
	fmt.Fprintln(w, "Backend contract verification (in-memory demonstration)")
	fmt.Fprintf(w, "\nOperation:\n  %s\nAdapter:\n  %s\n\nCases:\n", vc.Operation, vc.Adapter)
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	for _, c := range vc.Cases {
		status := "PASS"
		if !c.Passed {
			status = "FAIL: " + c.Detail
		}
		fmt.Fprintf(tw, "  %s:\t%s\n", c.Name, status)
	}
	_ = tw.Flush()
	fmt.Fprintf(w, "\nContract digest:\n  %s\n", vc.Digest)
	if !vc.Passed {
		return &CommandError{Code: ExitConfigInvalid}
	}
	return nil
}

func adapterDoctor(app *App) error {
	w := app.Stdout()
	fmt.Fprintln(w, "BatchWeaver adapter doctor")
	fmt.Fprintln(w)
	for _, m := range adapter.Manifests() {
		status := "OK"
		if err := m.Validate(); err != nil {
			status = "INVALID: " + err.Error()
		} else if m.RuntimeABI != adapter.RuntimeABIVersion {
			status = "ABI MISMATCH"
		}
		fmt.Fprintf(w, "  %-14s %s  status=%s\n", m.AdapterID, status, m.Status)
	}
	fmt.Fprintf(w, "\nRuntime ABI: %s\n", adapter.RuntimeABIVersion)
	fmt.Fprintln(w, "No backend connection was opened.")
	return nil
}

// splitLinesLocal splits on newlines for indented printing.
func splitLinesLocal(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
