package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// run executes the app with args and returns exit code plus captured streams.
func run(t *testing.T, args ...string) (ExitCode, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr)
	code := app.Run(context.Background(), args)
	return code, stdout.String(), stderr.String()
}

func TestRunNoArgsShowsHelp(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := run(t)
	if code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout, "Usage:") || !strings.Contains(stdout, "Commands:") {
		t.Errorf("stdout does not look like help:\n%s", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestRunHelpFlag(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{"-h", "--help", "help"} {
		code, stdout, stderr := run(t, flag)
		if code != ExitOK {
			t.Errorf("%s: exit code = %d, want %d", flag, code, ExitOK)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Errorf("%s: stdout missing usage:\n%s", flag, stdout)
		}
		if stderr != "" {
			t.Errorf("%s: stderr = %q, want empty", flag, stderr)
		}
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := run(t, "definitely-not-a-command")
	if code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr missing 'unknown command':\n%s", stderr)
	}
	if strings.Contains(stderr, "goroutine") || strings.Contains(stderr, ".go:") {
		t.Errorf("stderr looks like a stack trace:\n%s", stderr)
	}
}

func TestRunHelpForCommand(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := run(t, "help", "version")
	if code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout, "version") {
		t.Errorf("help version stdout missing 'version':\n%s", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestRunHelpForUnknownCommand(t *testing.T) {
	t.Parallel()

	code, _, stderr := run(t, "help", "nope")
	if code != ExitUsage {
		t.Errorf("exit code = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("stderr missing 'unknown command':\n%s", stderr)
	}
}

func TestHelpListsOnlyRealCommands(t *testing.T) {
	t.Parallel()

	_, stdout, _ := run(t, "help")
	// Command entries are rendered as two-space-indented lines; check those
	// specifically so that prose mentioning a word (for example "build") is not
	// mistaken for an advertised command.
	for _, unavailable := range []string{"build", "verify", "profile"} {
		if strings.Contains(stdout, "\n  "+unavailable+" ") {
			t.Errorf("help advertises unimplemented command %q:\n%s", unavailable, stdout)
		}
	}
	for _, real := range []string{"help", "version"} {
		if !strings.Contains(stdout, real) {
			t.Errorf("help missing real command %q:\n%s", real, stdout)
		}
	}
}
