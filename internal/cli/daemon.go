package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Voskan/BatchWeaver/internal/daemon"
)

// newDaemonCommand returns the "daemon" command.
func newDaemonCommand() *Command {
	return &Command{
		Name:    "daemon",
		Summary: "Manage the local workspace daemon",
		Usage:   "daemon start|status|stop|clean",
		Run:     runDaemon,
	}
}

func runDaemon(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "daemon requires a subcommand: start, status, stop, clean"}
	}
	root, err := os.Getwd()
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	switch args[0] {
	case "start":
		return daemonStart(ctx, app, root)
	case "status":
		return daemonStatus(app, root, args[1:])
	case "stop":
		return daemonStop(app, root)
	case "clean":
		return daemonClean(app, root)
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown daemon subcommand %q", args[0])}
	}
}

func daemonStart(ctx context.Context, app *App, root string) error {
	s, err := daemon.Start(ctx, root)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	fmt.Fprintf(app.Stderr(), "batchweaver daemon: listening on %s\n", s.Socket())
	<-ctx.Done()
	s.Close()
	return nil
}

func daemonStatus(app *App, root string, args []string) error {
	asJSON := len(args) > 0 && args[0] == "--json"
	info, health, err := daemon.Status(root)
	w := app.Stdout()
	if asJSON {
		out := map[string]any{"running": err == nil, "info": info, "health": health}
		if err != nil {
			out["error"] = err.Error()
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(w, string(b))
		return nil
	}
	switch {
	case errors.Is(err, daemon.ErrNotRunning):
		fmt.Fprintln(w, "daemon: not running")
	case err != nil:
		fmt.Fprintf(w, "daemon: %v\n", err)
	default:
		fmt.Fprintf(w, "daemon: running\n  pid:      %d\n  protocol: %s\n  uptime:   %ds\n  socket:   %s\n",
			health.PID, health.ProtocolVersion, health.UptimeSeconds, info.Socket)
	}
	return nil
}

func daemonStop(app *App, root string) error {
	if err := daemon.Stop(root); err != nil {
		if errors.Is(err, daemon.ErrNotRunning) {
			fmt.Fprintln(app.Stdout(), "daemon: not running")
			return nil
		}
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	fmt.Fprintln(app.Stdout(), "daemon: stop requested")
	return nil
}

func daemonClean(app *App, root string) error {
	if err := daemon.Clean(root); err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	fmt.Fprintln(app.Stdout(), "daemon: cleaned stale state")
	return nil
}
