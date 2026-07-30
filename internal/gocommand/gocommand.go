// Package gocommand runs the installed Go tool for transformed build, test, and
// run commands. It never uses a shell, passes arguments as an explicit array,
// streams standard output and error, and preserves the child exit code.
package gocommand

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Runner executes the Go tool with an overlay.
type Runner struct {
	// GoBin is the Go executable; when empty, "go" from PATH (or GOROOT) is used.
	GoBin string
	// Dir is the working directory for the command.
	Dir string
	// Stdout and Stderr receive the child streams; defaults are os.Stdout/os.Stderr.
	Stdout io.Writer
	Stderr io.Writer
	// Env, when non-nil, replaces the child environment.
	Env []string
}

// goExecutable resolves the Go binary, preferring an explicit path, then GOROOT,
// then PATH.
func (r Runner) goExecutable() string {
	if r.GoBin != "" {
		return r.GoBin
	}
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		cand := filepath.Join(goroot, "bin", "go")
		if runtime.GOOS == "windows" {
			cand += ".exe"
		}
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return "go"
}

// Run executes `go <subcommand> -overlay=<overlay> <args...>` and returns the
// process exit code. A non-nil error indicates the command could not be started
// or was interrupted; a non-zero code with a nil error means the Go command ran
// and reported failure.
func (r Runner) Run(ctx context.Context, subcommand, overlayPath string, args []string) (int, error) {
	full := []string{subcommand}
	if overlayPath != "" {
		full = append(full, "-overlay="+overlayPath)
	}
	full = append(full, args...)

	cmd := exec.CommandContext(ctx, r.goExecutable(), full...)
	cmd.Dir = r.Dir
	cmd.Stdout = writerOr(r.Stdout, os.Stdout)
	cmd.Stderr = writerOr(r.Stderr, os.Stderr)
	cmd.Stdin = os.Stdin
	if r.Env != nil {
		// Prevent recursive wrapper invocation via inherited tool flags.
		cmd.Env = sanitize(r.Env)
	} else {
		cmd.Env = sanitize(os.Environ())
	}

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}

// sanitize removes GOFLAGS entries that could recursively re-invoke a wrapper
// and strips any BatchWeaver-internal environment.
func sanitize(env []string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if len(e) >= 3 && e[:3] == "BW_" {
			continue
		}
		out = append(out, e)
	}
	return out
}

func writerOr(w, def io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return def
}
