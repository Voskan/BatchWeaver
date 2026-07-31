package gocommand

import (
	"context"
	"errors"
	"os"
	"os/exec"
)

// ToolexecMarker is the environment variable the driver sets on delegated tool
// processes to detect and prevent recursive -toolexec invocation.
const ToolexecMarker = "BW_TOOLEXEC_ACTIVE"

// RecursionPolicy controls how the driver handles an already-active toolexec
// marker (a nested Go command inheriting -toolexec).
type RecursionPolicy string

const (
	// RecursionError fails when recursion is detected.
	RecursionError RecursionPolicy = "error"
	// RecursionDelegate delegates the tool unchanged when recursion is detected
	// (the safe default: the overlay already carries the transformation).
	RecursionDelegate RecursionPolicy = "strip-inner"
)

// ErrRecursiveToolexec is returned when a recursive invocation is detected under
// the error policy.
var ErrRecursiveToolexec = errors.New("recursive -toolexec invocation detected")

// Toolexec is the -toolexec driver. In the overlay-first architecture the
// transformation is applied through a Go build overlay, so the driver's role is
// to observe compile actions, prevent recursion and environment contamination,
// and delegate every tool faithfully while preserving its exit code. It never
// uses a shell.
type Toolexec struct {
	// Policy selects recursion handling; the zero value delegates safely.
	Policy RecursionPolicy
}

// Run executes the given tool with its arguments, preserving stdout, stderr, and
// exit code. args[0] is the absolute tool path; args[1:] are its arguments.
func (d Toolexec) Run(ctx context.Context, args []string) (int, error) {
	if len(args) == 0 {
		return 2, errors.New("toolexec: no tool specified")
	}
	if os.Getenv(ToolexecMarker) != "" && d.Policy == RecursionError {
		return 1, ErrRecursiveToolexec
	}

	tool := args[0]
	cmd := exec.CommandContext(ctx, tool, args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(sanitize(os.Environ()), ToolexecMarker+"=1")

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

// IsCompile reports whether a tool path is the Go compiler, for callers that
// want to transform only compile actions. Identification uses the tool base name
// as provided by the Go command.
func IsCompile(toolPath string) bool {
	base := baseName(toolPath)
	return base == "compile" || base == "compile.exe"
}

// baseName returns the final path element without a directory, using both
// separators so the check is cross-platform.
func baseName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}
