package cli

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/Voskan/BatchWeaver/internal/transform"
)

// newRuntimeCommand returns the "runtime" command (inspect).
func newRuntimeCommand() *Command {
	return &Command{
		Name:    "runtime",
		Summary: "Inspect runtime-lowered operations",
		Usage:   "runtime inspect [--operation id] [packages]",
		Run:     runRuntime,
	}
}

func runRuntime(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "inspect" {
		return &CommandError{Code: ExitUsage, Message: "usage: runtime inspect [--operation id] [packages]"}
	}
	fs := flag.NewFlagSet("runtime inspect", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	op := fs.String("operation", "", "restrict to one operation ID")
	if err := fs.Parse(args[1:]); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	strategies := []transform.StrategyID{
		transform.StrategyRuntimeCallCoalescing, transform.StrategyStaticSiblingFusion, transform.StrategyFanoutCoalescing,
	}
	plan, err := planFromArgs(ctx, fs.Args(), transform.Filter{Operation: *op}, strategies)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	byOp := map[string]int{}
	for _, tr := range plan.Transformations {
		if transform.RuntimeStrategies(tr.Strategy) {
			byOp[tr.Operation]++
		}
	}
	w := app.Stdout()
	fmt.Fprintln(w, "BatchWeaver runtime lowering")
	fmt.Fprintln(w)
	if len(byOp) == 0 {
		fmt.Fprintln(w, "No runtime-lowered operations in the current plan.")
	} else {
		ops := make([]string, 0, len(byOp))
		for o := range byOp {
			ops = append(ops, o)
		}
		sort.Strings(ops)
		for _, o := range ops {
			fmt.Fprintf(w, "  %s: %d transformed call site group(s)\n", o, byOp[o])
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Execution modes are selected at runtime (runtime-coalesced, batch-of-one,")
	fmt.Fprintln(w, "direct-scalar fallback). Live per-operation statistics require a running")
	fmt.Fprintln(w, "instrumented process and are not collected by this static command.")
	return nil
}

// newBarrierCommand returns the "barrier" command (inspect).
func newBarrierCommand() *Command {
	return &Command{
		Name:    "barrier",
		Summary: "Inspect batching barriers implied by observable effects",
		Usage:   "barrier inspect [packages]",
		Run:     runBarrier,
	}
}

// staticBarrierEffects maps analysis effect categories to barrier kinds that
// pending batched reads must not be reordered across.
var staticBarrierEffects = map[string]string{
	"global-write":    "observable write",
	"channel":         "channel communication",
	"synchronization": "lock or synchronization",
	"network":         "network side effect",
	"filesystem":      "filesystem side effect",
	"process":         "process execution",
	"database":        "database transaction",
	"unsafe":          "unsafe boundary",
	"unknown-call":    "unknown side effect",
}

func runBarrier(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "inspect" {
		return &CommandError{Code: ExitUsage, Message: "usage: barrier inspect [packages]"}
	}
	// A barrier listing is derived from the runtime-lowering plan's non-guarantees
	// and the operations it lowers: pending reads flush at observable barriers.
	plan, err := planFromArgs(ctx, args[1:], transform.Filter{},
		[]transform.StrategyID{transform.StrategyRuntimeCallCoalescing, transform.StrategyFanoutCoalescing})
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	w := app.Stdout()
	fmt.Fprintln(w, "BatchWeaver batching barriers")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The following barrier kinds flush pending batched reads. Insert an explicit")
	fmt.Fprintln(w, "barrier with bridge.Flush(ctx) before any of them when reads are pending:")
	kinds := make([]string, 0, len(staticBarrierEffects))
	for _, v := range staticBarrierEffects {
		kinds = append(kinds, v)
	}
	kinds = append(kinds, "transaction begin/commit/rollback", "scope end")
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Fprintf(w, "  - %s\n", k)
	}
	var ops []string
	seen := map[string]bool{}
	for _, tr := range plan.Transformations {
		if transform.RuntimeStrategies(tr.Strategy) && !seen[tr.Operation] {
			seen[tr.Operation] = true
			ops = append(ops, tr.Operation)
		}
	}
	sort.Strings(ops)
	if len(ops) > 0 {
		fmt.Fprintf(w, "\nRuntime-lowered operations subject to barriers: %s\n", strings.Join(ops, ", "))
	}
	return nil
}
