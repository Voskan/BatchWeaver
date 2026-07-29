package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Voskan/BatchWeaver/config"
	"github.com/Voskan/BatchWeaver/diagnostics"
)

// newConfigCommand returns the "config" command, which groups configuration
// validation and inspection subcommands.
func newConfigCommand() *Command {
	return &Command{
		Name:    "config",
		Summary: "Validate and inspect BatchWeaver configuration",
		Usage:   "config <validate|print> [--file path] [--format ...]",
		Run:     runConfig,
	}
}

func runConfig(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "config requires a subcommand: validate or print"}
	}
	switch args[0] {
	case "validate":
		return runConfigValidate(ctx, app, args[1:])
	case "print":
		return runConfigPrint(ctx, app, args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(app.Stdout(), "Usage:\n  batchweaver config validate [--file path] [--format text|json]\n  batchweaver config print [--file path] [--format json]")
		return nil
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown config subcommand %q", args[0])}
	}
}

// loadConfigFromPath loads configuration, discovering upward when path is empty.
func loadConfigFromPath(ctx context.Context, path string) (config.LoadResult, error) {
	opts := config.LoadOptions{Path: path, Discover: path == ""}
	if path == "" {
		if wd, err := os.Getwd(); err == nil {
			opts.WorkingDirectory = wd
		}
	}
	return config.Load(ctx, opts)
}

// loadExitCode maps a load result to an exit code.
func loadExitCode(res config.LoadResult) ExitCode {
	switch {
	case !res.Found:
		return ExitConfigNotFound
	case res.HasErrors():
		return ExitConfigInvalid
	default:
		return ExitOK
	}
}

func runConfigValidate(ctx context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("config validate", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("file", "", "configuration file path (discovered when omitted)")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *format != "text" && *format != "json" {
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown format %q (want text or json)", *format)}
	}

	res, err := loadConfigFromPath(ctx, *file)
	if err != nil {
		return err
	}
	code := loadExitCode(res)

	switch {
	case *format == "json":
		if err := diagnostics.NewJSONFormatter().Format(app.Stdout(), res.Diagnostics); err != nil {
			return err
		}
	case code != ExitOK:
		if err := diagnostics.NewTextFormatter().Format(app.Stderr(), res.Diagnostics); err != nil {
			return err
		}
	default:
		fmt.Fprintf(app.Stdout(), "Configuration is valid.\nSchema: %d\nFiles: %d\nOperations: %d\nDigest: %s\n",
			res.Config.Version, len(res.Files), res.Catalog.Len(), res.Digest)
	}

	if code != ExitOK {
		return &CommandError{Code: code}
	}
	return nil
}

func runConfigPrint(ctx context.Context, app *App, args []string) error {
	fs := flag.NewFlagSet("config print", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("file", "", "configuration file path (discovered when omitted)")
	format := fs.String("format", "json", "output format: json")
	if err := fs.Parse(args); err != nil {
		return &CommandError{Code: ExitUsage}
	}

	res, err := loadConfigFromPath(ctx, *file)
	if err != nil {
		return err
	}
	if code := loadExitCode(res); code != ExitOK {
		if err := diagnostics.NewTextFormatter().Format(app.Stderr(), res.Diagnostics); err != nil {
			return err
		}
		return &CommandError{Code: code}
	}

	switch *format {
	case "json":
		out, err := config.RenderJSON(res.Config)
		if err != nil {
			return err
		}
		_, err = app.Stdout().Write(out)
		return err
	case "yaml":
		return &CommandError{Code: ExitUsage, Message: "YAML output is not yet implemented; use --format json"}
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown format %q (want json)", *format)}
	}
}
