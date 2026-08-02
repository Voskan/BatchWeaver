package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Voskan/BatchWeaver/internal/buildinfo"
)

// newVersionCommand returns the "version" command, which prints deterministic
// build and platform information.
func newVersionCommand() *Command {
	return &Command{
		Name:    "version",
		Summary: "Print BatchWeaver version and build information",
		Usage:   "version [--json]",
		Run: func(_ context.Context, app *App, args []string) error {
			info := buildinfo.Get()
			if len(args) == 1 && args[0] == "--json" {
				enc := json.NewEncoder(app.Stdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			if len(args) != 0 {
				return &CommandError{Code: ExitUsage, Message: "usage: batchweaver version [--json]"}
			}
			fmt.Fprintln(app.Stdout(), info.String())
			return nil
		},
	}
}
