package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Voskan/BatchWeaver/internal/buildinfo"
	"github.com/Voskan/BatchWeaver/internal/lsp/proxy"
	"github.com/Voskan/BatchWeaver/internal/lsp/server"
)

// newLSPCommand returns the "lsp" command.
func newLSPCommand() *Command {
	return &Command{
		Name:    "lsp",
		Summary: "Run the BatchWeaver language server (LSP)",
		Usage:   "lsp [--stdio] [--proxy-gopls --gopls-command=gopls]",
		Run:     runLSP,
	}
}

func runLSP(ctx context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("lsp", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	stdio := fs.Bool("stdio", true, "communicate over stdin/stdout (the only supported transport)")
	proxyGopls := fs.Bool("proxy-gopls", false, "run in proxy mode, delegating standard Go features to gopls")
	goplsCmd := fs.String("gopls-command", "gopls", "gopls executable to launch in proxy mode")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if !*stdio {
		return &CommandError{Code: ExitUsage, Message: "only --stdio transport is supported"}
	}
	// In stdio mode, logs must never go to stdout (that channel is the protocol).
	logf := func(format string, a ...any) { fmt.Fprintf(os.Stderr, "batchweaver lsp: "+format+"\n", a...) }

	if *proxyGopls {
		p := proxy.New(proxy.Options{
			GoplsCommand: *goplsCmd,
			ToolVersion:  buildinfo.Get().Version,
			Logf:         logf,
		})
		if err := p.Run(ctx, os.Stdin, os.Stdout); err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		return nil
	}

	srv := server.New(server.Options{ToolVersion: buildinfo.Get().Version, Logf: logf})
	if err := srv.Run(ctx, os.Stdin, os.Stdout); err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	return nil
}
