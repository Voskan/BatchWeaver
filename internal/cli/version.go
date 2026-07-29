package cli

import (
	"context"
	"fmt"

	"github.com/Voskan/BatchWeaver/internal/buildinfo"
)

// newVersionCommand returns the "version" command, which prints deterministic
// build and platform information.
func newVersionCommand() *Command {
	return &Command{
		Name:    "version",
		Summary: "Print BatchWeaver version and build information",
		Usage:   "version",
		Run: func(_ context.Context, app *App, _ []string) error {
			fmt.Fprintln(app.Stdout(), buildinfo.Get().String())
			return nil
		},
	}
}
