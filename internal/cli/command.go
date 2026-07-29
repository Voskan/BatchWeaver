package cli

import "context"

// ExitCode is a typed process exit status. Library code returns an ExitCode
// rather than calling os.Exit, so that behavior stays testable; the executable
// entry point performs the single os.Exit call.
type ExitCode int

const (
	// ExitOK indicates successful execution.
	ExitOK ExitCode = 0
	// ExitError indicates a runtime error while executing a command.
	ExitError ExitCode = 1
	// ExitUsage indicates the command line was used incorrectly, for example an
	// unknown command or bad flags.
	ExitUsage ExitCode = 2
)

// Command is a single BatchWeaver subcommand. Commands are values with no
// hidden global state; everything they need is passed through the App and their
// arguments.
type Command struct {
	// Name is the token used to invoke the command, for example "version".
	Name string
	// Summary is a one-line description shown in the command list.
	Summary string
	// Usage is a short usage string, excluding the program name.
	Usage string
	// Run executes the command with the arguments that follow the command name.
	// It returns an error on failure; it must not call os.Exit.
	Run func(ctx context.Context, app *App, args []string) error
}
