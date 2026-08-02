// Package cli implements BatchWeaver's command-line interface using only the
// standard library.
//
// The design keeps output injectable and avoids os.Exit inside library code:
// App.Run returns a typed ExitCode that the executable maps to a process exit
// status. This makes every command deterministically testable. The command set
// is intentionally minimal in the bootstrap; only functionality that actually
// exists is advertised.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// programName is the name shown in usage and help output.
const programName = "batchweaver"

// App is a BatchWeaver CLI instance. Construct it with New. Output is written
// to the configured writers so tests can capture it.
type App struct {
	stdout   io.Writer
	stderr   io.Writer
	commands map[string]*Command
	order    []string
}

// New returns an App that writes to stdout and stderr. It registers the
// commands available in this build.
func New(stdout, stderr io.Writer) *App {
	a := &App{
		stdout:   stdout,
		stderr:   stderr,
		commands: make(map[string]*Command),
	}
	a.register(newVersionCommand())
	a.register(newConfigCommand())
	a.register(newOperationCommand())
	a.register(newScanCommand())
	a.register(newProveCommand())
	a.register(newCandidateCommand())
	a.register(newProofCommand())
	a.register(newAssumptionCommand())
	a.register(newStrategyCommand())
	a.register(newTransformCommand())
	a.register(newBuildCommand())
	a.register(newTestCommand())
	a.register(newRunCommand())
	a.register(newRuntimeCommand())
	a.register(newBarrierCommand())
	a.register(newToolExecCommand())
	a.register(newToolexecCommand())
	a.register(newAdapterCommand())
	a.register(newGraphQLCommand())
	a.register(newGRPCCommand())
	a.register(newHTTPCommand())
	a.register(newOpenAPICommand())
	a.register(newProfileCommand())
	a.register(newTuneCommand())
	a.register(newFairnessCommand())
	a.register(newOverloadCommand())
	a.register(newWaveCommand())
	a.register(newRecursiveCommand())
	a.register(newLSPCommand())
	a.register(newDaemonCommand())
	a.register(newEditorCommand())
	a.register(newDoctorCommand())
	a.register(newVerifyCommand())
	a.register(newCompatibilityCommand())
	a.register(newSecurityCommand())
	a.register(newReleaseCommand())
	a.register(newHelpCommand())
	return a
}

// Stdout returns the writer used for standard output.
func (a *App) Stdout() io.Writer { return a.stdout }

// Stderr returns the writer used for standard error.
func (a *App) Stderr() io.Writer { return a.stderr }

// register adds a command, keeping a stable, sorted invocation order for help.
func (a *App) register(c *Command) {
	a.commands[c.Name] = c
	a.order = append(a.order, c.Name)
	sort.Strings(a.order)
}

// Run executes the CLI with the given arguments, which must exclude the program
// name (that is, pass os.Args[1:]). It returns a typed ExitCode and never calls
// os.Exit.
func (a *App) Run(ctx context.Context, args []string) ExitCode {
	// No arguments, or a top-level help flag: show general help successfully.
	if len(args) == 0 {
		a.printHelp()
		return ExitOK
	}
	switch args[0] {
	case "-h", "--help", "help":
		return a.runHelp(ctx, args[1:])
	}

	name := args[0]
	cmd, ok := a.commands[name]
	if !ok {
		fmt.Fprintf(a.stderr, "%s: unknown command %q\n", programName, name)
		fmt.Fprintf(a.stderr, "Run '%s help' for the list of commands.\n", programName)
		return ExitUsage
	}

	if err := cmd.Run(ctx, a, args[1:]); err != nil {
		var ce *CommandError
		if errors.As(err, &ce) {
			if !ce.silent() {
				fmt.Fprintf(a.stderr, "%s %s: %s\n", programName, name, ce.Message)
			}
			return ce.Code
		}
		fmt.Fprintf(a.stderr, "%s %s: %v\n", programName, name, err)
		return ExitError
	}
	return ExitOK
}

// runHelp implements both "help" and "help <command>".
func (a *App) runHelp(_ context.Context, args []string) ExitCode {
	if len(args) == 0 {
		a.printHelp()
		return ExitOK
	}
	name := args[0]
	cmd, ok := a.commands[name]
	if !ok {
		fmt.Fprintf(a.stderr, "%s: unknown command %q\n", programName, name)
		fmt.Fprintf(a.stderr, "Run '%s help' for the list of commands.\n", programName)
		return ExitUsage
	}
	a.printCommandHelp(cmd)
	return ExitOK
}

// printHelp writes the general help text to stdout.
func (a *App) printHelp() {
	var b strings.Builder
	fmt.Fprintf(&b, "%s is an automatic semantic batching compiler and runtime for Go.\n\n", programName)
	fmt.Fprintf(&b, "The compiler and runtime are under active development. This build provides\n")
	fmt.Fprintf(&b, "the commands listed below.\n\n")
	fmt.Fprintf(&b, "Usage:\n  %s <command> [arguments]\n\n", programName)
	fmt.Fprintf(&b, "Commands:\n")

	width := 0
	for _, name := range a.order {
		if len(name) > width {
			width = len(name)
		}
	}
	for _, name := range a.order {
		cmd := a.commands[name]
		fmt.Fprintf(&b, "  %-*s  %s\n", width, cmd.Name, cmd.Summary)
	}
	fmt.Fprintf(&b, "\nRun '%s help <command>' for details on a command.\n", programName)

	fmt.Fprint(a.stdout, b.String())
}

// printCommandHelp writes help for a single command to stdout.
func (a *App) printCommandHelp(cmd *Command) {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", cmd.Summary)
	usage := cmd.Usage
	if usage == "" {
		usage = cmd.Name
	}
	fmt.Fprintf(&b, "Usage:\n  %s %s\n", programName, usage)
	fmt.Fprint(a.stdout, b.String())
}

// newHelpCommand registers "help" so it appears in the command list. The actual
// dispatch is handled in Run and runHelp because help needs access to the full
// command set.
func newHelpCommand() *Command {
	return &Command{
		Name:    "help",
		Summary: "Show help for BatchWeaver or a specific command",
		Usage:   "help [command]",
		Run: func(ctx context.Context, app *App, args []string) error {
			// Reachable only if invoked as a positional command name; delegate to
			// the same logic used by the "help" keyword.
			app.runHelp(ctx, args)
			return nil
		},
	}
}
