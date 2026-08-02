package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/Voskan/BatchWeaver/internal/buildinfo"
	"github.com/Voskan/BatchWeaver/internal/daemon"
	"github.com/Voskan/BatchWeaver/internal/lsp/server"
)

// newEditorCommand returns the "editor" command.
func newEditorCommand() *Command {
	return &Command{
		Name:    "editor",
		Summary: "Editor integration diagnostics (doctor)",
		Usage:   "editor doctor [--json]",
		Run:     runEditor,
	}
}

func runEditor(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "doctor" {
		return &CommandError{Code: ExitUsage, Message: "usage: editor doctor [--json]"}
	}
	asJSON := len(args) > 1 && args[1] == "--json"

	info := buildinfo.Get()
	report := map[string]any{
		"batchweaver_version": info.Version,
		"go_version":          runtime.Version(),
		"lsp_protocol":        server.LSPVersion,
		"lsp_server":          server.ServerName,
		"platform":            runtime.GOOS + "/" + runtime.GOARCH,
	}

	goplsPath, goplsVer := detectGopls(ctx)
	report["gopls_path"] = goplsPath
	report["gopls_version"] = goplsVer

	if _, _, err := daemon.Status("."); err != nil {
		report["daemon"] = "not running"
	} else {
		report["daemon"] = "running"
	}

	w := app.Stdout()
	if asJSON {
		b, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(w, string(b))
		return nil
	}
	fmt.Fprintln(w, "BatchWeaver editor doctor")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  BatchWeaver:   %s\n", report["batchweaver_version"])
	fmt.Fprintf(w, "  Go:            %s\n", report["go_version"])
	fmt.Fprintf(w, "  Platform:      %s\n", report["platform"])
	fmt.Fprintf(w, "  LSP protocol:  %s\n", report["lsp_protocol"])
	fmt.Fprintf(w, "  LSP server:    %s\n", report["lsp_server"])
	if goplsPath == "" {
		fmt.Fprintln(w, "  gopls:         not found on PATH (proxy mode unavailable; sidecar mode with gopls recommended)")
	} else {
		fmt.Fprintf(w, "  gopls:         %s (%s)\n", goplsPath, goplsVer)
	}
	fmt.Fprintf(w, "  daemon:        %s\n", report["daemon"])
	return nil
}

// detectGopls locates gopls and reports its version without importing it. It
// never downloads gopls.
func detectGopls(ctx context.Context) (path, version string) {
	p, err := exec.LookPath("gopls")
	if err != nil {
		return "", ""
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cctx, p, "version").Output()
	if err != nil {
		return p, "unknown"
	}
	v := strings.TrimSpace(string(out))
	if i := strings.IndexByte(v, '\n'); i >= 0 {
		v = v[:i]
	}
	return p, v
}
