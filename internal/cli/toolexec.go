package cli

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/Voskan/BatchWeaver/internal/gocommand"
	"github.com/Voskan/BatchWeaver/internal/transform"
)

// newToolexecCommand returns the hidden "-toolexec" driver command. It is
// invoked by the Go command as `go build -toolexec="batchweaver toolexec" ...`
// and delegates each tool faithfully, preventing recursion.
func newToolexecCommand() *Command {
	return &Command{
		Name:    "toolexec",
		Summary: "Internal -toolexec driver (invoked by the Go command)",
		Usage:   "toolexec <tool> [args]",
		Run:     runToolexec,
	}
}

func runToolexec(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "toolexec requires a tool"}
	}
	d := gocommand.Toolexec{Policy: gocommand.RecursionDelegate}
	code, err := d.Run(ctx, args)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	if code != 0 {
		return &CommandError{Code: ExitGoCommand}
	}
	return nil
}

// newToolExecCommand returns the "tool-exec" inspection command (doctor/explain).
func newToolExecCommand() *Command {
	return &Command{
		Name:    "tool-exec",
		Summary: "Diagnose and explain the -toolexec integration",
		Usage:   "tool-exec doctor | tool-exec explain",
		Run:     runToolExec,
	}
}

func runToolExec(_ context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "tool-exec requires a subcommand: doctor or explain"}
	}
	switch args[0] {
	case "doctor":
		return toolExecDoctor(app)
	case "explain":
		return toolExecExplain(app)
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown tool-exec subcommand %q", args[0])}
	}
}

func toolExecDoctor(app *App) error {
	w := app.Stdout()
	fmt.Fprintln(w, "BatchWeaver tool-exec doctor")
	fmt.Fprintln(w)
	check := func(name, status, detail string) {
		fmt.Fprintf(w, "  %-28s %s", name+":", status)
		if detail != "" {
			fmt.Fprintf(w, "  (%s)", detail)
		}
		fmt.Fprintln(w)
	}
	check("go toolchain", "OK", runtime.Version())
	check("-overlay support", "OK", "Go 1.16+ supports build overlays")
	if gf := os.Getenv("GOFLAGS"); gf != "" {
		check("GOFLAGS", "review", gf)
	} else {
		check("GOFLAGS", "clean", "")
	}
	if os.Getenv(gocommand.ToolexecMarker) != "" {
		check("recursion marker", "active", "already inside a delegated tool; nesting policy applies")
	} else {
		check("recursion marker", "clear", "")
	}
	if root, err := transform.ModuleRoot(cwd()); err == nil {
		check("workspace module", "OK", root)
	} else {
		check("workspace module", "none", "run inside a Go module")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The default architecture is overlay-first: transformations are applied")
	fmt.Fprintln(w, "through a Go -overlay, and the -toolexec driver delegates every tool.")
	return nil
}

func toolExecExplain(app *App) error {
	w := app.Stdout()
	fmt.Fprintln(w, "How BatchWeaver integrates with the Go command:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, strings.TrimSpace(`
1. `+programName+` plans certified transformations and writes an overlay manifest
   plus content-addressed transformed and generated files under .batchweaver/.
2. build/test/run invoke the Go command with -overlay=<manifest>, so the Go
   toolchain compiles the transformed bytes without editing the source tree.
3. The optional -toolexec driver ("`+programName+` toolexec") observes each tool
   invocation, prevents recursive -toolexec nesting via `+gocommand.ToolexecMarker+`,
   and delegates every tool faithfully, preserving its exit code.
`))
	return nil
}
