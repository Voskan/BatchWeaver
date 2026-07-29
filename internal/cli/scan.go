package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/internal/buildinfo"
)

// newScanCommand returns the "scan" command, which statically analyzes Go
// packages and reports potential batching structure without modifying source.
func newScanCommand() *Command {
	return &Command{
		Name:    "scan",
		Summary: "Statically analyze Go packages for batching structure",
		Usage:   "scan [--format text|json] [--reproducible] [--tests] [--goos os] [--goarch arch] [--tags t1,t2] [--fail-on error|warning|never] [packages]",
		Run:     runScan,
	}
}

func runScan(ctx context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	format := fs.String("format", "text", "output format: text or json")
	reproducible := fs.Bool("reproducible", false, "omit volatile fields for byte-stable output")
	tests := fs.Bool("tests", false, "include test package variants")
	goos := fs.String("goos", "", "target GOOS (default: current)")
	goarch := fs.String("goarch", "", "target GOARCH (default: current)")
	tags := fs.String("tags", "", "comma-separated build tags")
	cgo := fs.Bool("cgo", false, "enable cgo")
	failOn := fs.String("fail-on", "error", "nonzero exit on: error, warning, or never")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *format != "text" && *format != "json" {
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown format %q (want text or json)", *format)}
	}
	if *failOn != "error" && *failOn != "warning" && *failOn != "never" {
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown fail-on %q (want error, warning, or never)", *failOn)}
	}

	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	snap, err := analysis.Analyze(ctx, analysis.Request{
		Patterns:     patterns,
		Reproducible: *reproducible,
		ToolVersion:  buildinfo.Get().Version,
		BuildContext: analysis.BuildContext{
			GOOS: *goos, GOARCH: *goarch, CGOEnabled: *cgo,
			Tags: splitTags(*tags), Tests: *tests,
		},
	})
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}

	switch *format {
	case "json":
		if err := analysis.RenderJSON(app.Stdout(), snap); err != nil {
			return err
		}
	default:
		if err := analysis.RenderText(app.Stdout(), snap); err != nil {
			return err
		}
	}

	switch {
	case *failOn == "never":
		return nil
	case snap.HasErrors():
		return &CommandError{Code: ExitConfigInvalid}
	case *failOn == "warning" && snap.HasWarnings():
		return &CommandError{Code: ExitConfigInvalid}
	default:
		return nil
	}
}

// splitTags splits a comma-separated tag list, dropping empty entries.
func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, t := range strings.Split(s, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}
