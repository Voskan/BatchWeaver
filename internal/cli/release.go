package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Voskan/BatchWeaver/internal/release"
)

func newReleaseCommand() *Command {
	return &Command{Name: "release", Summary: "Check, build, verify, and reproduce unpublished release snapshots", Usage: "release <check|build|verify|reproduce|notes|manifest|clean> [arguments]", Run: runRelease}
}

func runRelease(_ context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(app.Stdout(), "Usage: batchweaver release <check|build|verify|reproduce|notes|manifest|clean> [arguments]\n\nRelease commands never publish artifacts.")
		return nil
	}
	root, err := release.Root(".")
	if err != nil {
		return err
	}
	switch args[0] {
	case "check":
		fs := flag.NewFlagSet("release check", flag.ContinueOnError)
		fs.SetOutput(app.Stderr())
		jsonOut := fs.Bool("json", false, "emit JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return &CommandError{Code: ExitUsage, Message: ""}
		}
		report, err := release.Check(release.CheckOptions{Root: root})
		if err != nil {
			return err
		}
		if *jsonOut {
			enc := json.NewEncoder(app.Stdout())
			enc.SetIndent("", "  ")
			_ = enc.Encode(report)
		} else {
			printReadiness(app, report)
		}
		if !report.Ready {
			return &CommandError{Code: ExitError, Message: "release readiness failed; no publication was performed"}
		}
		return nil
	case "build":
		fs := flag.NewFlagSet("release build", flag.ContinueOnError)
		fs.SetOutput(app.Stderr())
		snapshot := fs.Bool("snapshot", false, "build an unpublished snapshot")
		output := fs.String("output", "dist", "output directory")
		version := fs.String("version", "", "version candidate (defaults to release/VERSION)")
		if err := fs.Parse(args[1:]); err != nil {
			return &CommandError{Code: ExitUsage, Message: ""}
		}
		manifest, err := release.Build(release.BuildOptions{Root: root, Output: *output, Version: *version, Snapshot: *snapshot})
		if err != nil {
			return err
		}
		fmt.Fprintf(app.Stdout(), "Release snapshot built\n\nCommit: %s\nArtifacts: %d\nChecksums: generated\nSBOM: SPDX 2.3 and CycloneDX 1.5 generated\nProvenance: local unsigned statement generated\nSignatures: disabled for unsigned snapshot\nPublication: disabled\n", manifest.Commit, len(manifest.Artifacts))
		return nil
	case "verify":
		if len(args) != 2 {
			return &CommandError{Code: ExitUsage, Message: "usage: batchweaver release verify <release-manifest.json>"}
		}
		if err := release.Verify(args[1]); err != nil {
			return err
		}
		fmt.Fprintln(app.Stdout(), "Release artifacts verified\nPublication: not performed")
		return nil
	case "reproduce":
		fs := flag.NewFlagSet("release reproduce", flag.ContinueOnError)
		fs.SetOutput(app.Stderr())
		manifest := fs.String("manifest", "", "manifest path")
		if err := fs.Parse(args[1:]); err != nil {
			return &CommandError{Code: ExitUsage, Message: ""}
		}
		if *manifest == "" {
			return &CommandError{Code: ExitUsage, Message: "--manifest is required"}
		}
		if err := release.Reproduce(*manifest, root); err != nil {
			return err
		}
		fmt.Fprintln(app.Stdout(), "Declared release artifacts are byte-reproducible under the local declared toolchain.")
		return nil
	case "manifest":
		path := "dist/release-manifest.json"
		if len(args) > 1 {
			path = args[1]
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Fprint(app.Stdout(), string(data))
		return nil
	case "notes":
		version, err := release.RecommendedVersion(root)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(filepath.Join(root, "docs", "release", "release-notes-"+version+".md"))
		if err != nil {
			return err
		}
		fmt.Fprint(app.Stdout(), string(data))
		return nil
	case "clean":
		fs := flag.NewFlagSet("release clean", flag.ContinueOnError)
		fs.SetOutput(app.Stderr())
		output := fs.String("output", "", "explicit output directory")
		if err := fs.Parse(args[1:]); err != nil {
			return &CommandError{Code: ExitUsage, Message: ""}
		}
		if *output == "" {
			return &CommandError{Code: ExitUsage, Message: "--output is required"}
		}
		if err := release.Clean(root, *output); err != nil {
			return err
		}
		fmt.Fprintf(app.Stdout(), "Removed release output %s\n", *output)
		return nil
	default:
		return &CommandError{Code: ExitUsage, Message: "unknown release subcommand " + args[0]}
	}
}

func printReadiness(app *App, report *release.ReadinessReport) {
	fmt.Fprintln(app.Stdout(), "BatchWeaver release readiness")
	fmt.Fprintf(app.Stdout(), "\nSource commit: %s\nVersion recommendation: %s\n", report.Commit, report.Version)
	for _, check := range report.Checks {
		fmt.Fprintf(app.Stdout(), "%-30s %s — %s\n", check.Name+":", check.Status, oneLine(check.Detail))
	}
	fmt.Fprintf(app.Stdout(), "\nReady: %t\nRelease publication: %s\n", report.Ready, report.Publication)
}

func oneLine(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", "") }

func newVerifyCommand() *Command {
	return &Command{Name: "verify", Summary: "Run semantic release verification suites", Usage: "verify differential ./...", Run: func(_ context.Context, app *App, args []string) error {
		if len(args) < 1 || args[0] != "differential" {
			return &CommandError{Code: ExitUsage, Message: "usage: batchweaver verify differential ./..."}
		}
		root, err := release.Root(".")
		if err != nil {
			return err
		}
		if err := release.RunGoTest(root, "./internal/assurance", "^TestDifferential"); err != nil {
			return err
		}
		fmt.Fprintln(app.Stdout(), "Differential verification passed: deterministic randomized, fault-injection, and short-soak cases.")
		return nil
	}}
}

func newCompatibilityCommand() *Command {
	return &Command{Name: "compatibility", Summary: "Report supported and actually tested environments", Usage: "compatibility report [--json]", Run: func(_ context.Context, app *App, args []string) error {
		if len(args) == 0 || args[0] != "report" {
			return &CommandError{Code: ExitUsage, Message: "usage: batchweaver compatibility report [--json]"}
		}
		root, err := release.Root(".")
		if err != nil {
			return err
		}
		data, err := os.ReadFile(filepath.Join(root, "release", "compatibility.json"))
		if err != nil {
			return err
		}
		if len(args) > 1 && args[1] == "--json" {
			fmt.Fprint(app.Stdout(), string(data))
			return nil
		}
		var v struct {
			AsOf string `json:"as_of"`
			Rows []struct {
				Category    string `json:"category"`
				Environment string `json:"environment"`
				Status      string `json:"status"`
				Evidence    string `json:"evidence"`
			} `json:"rows"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		fmt.Fprintf(app.Stdout(), "BatchWeaver compatibility report (as of %s)\n\n", v.AsOf)
		for _, r := range v.Rows {
			fmt.Fprintf(app.Stdout(), "%-12s %-28s %-30s %s\n", r.Category, r.Environment, r.Status, r.Evidence)
		}
		return nil
	}}
}

func newSecurityCommand() *Command {
	return &Command{Name: "security", Summary: "Audit local security and supply-chain controls", Usage: "security audit [--format=text|json]", Run: func(_ context.Context, app *App, args []string) error {
		if len(args) == 0 || args[0] != "audit" {
			return &CommandError{Code: ExitUsage, Message: "usage: batchweaver security audit [--format=text|json]"}
		}
		root, err := release.Root(".")
		if err != nil {
			return err
		}
		report := release.AuditSecurity(root)
		format := "text"
		for _, arg := range args[1:] {
			if strings.HasPrefix(arg, "--format=") {
				format = strings.TrimPrefix(arg, "--format=")
			}
		}
		if format == "json" {
			enc := json.NewEncoder(app.Stdout())
			enc.SetIndent("", "  ")
			_ = enc.Encode(report)
		} else {
			fmt.Fprintln(app.Stdout(), "BatchWeaver security audit")
			for _, check := range report.Checks {
				fmt.Fprintf(app.Stdout(), "%s %s — %s\n", check.Status, check.Name, check.Detail)
			}
			fmt.Fprintln(app.Stdout(), "\nExternal checks:")
			for _, item := range report.ExternalScanners {
				fmt.Fprintln(app.Stdout(), "-", item)
			}
			fmt.Fprintln(app.Stdout(), "Sensitive details: withheld; network collection: none")
		}
		if report.BlockingFindings > 0 {
			return &CommandError{Code: ExitError, Message: "release-blocking security findings detected"}
		}
		return nil
	}}
}
