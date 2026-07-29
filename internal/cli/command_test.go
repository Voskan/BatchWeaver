package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestCommandErrorMapsToExitError(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr)

	// Register a command that fails, to verify error-to-exit-code mapping without
	// depending on any real command failing.
	app.register(&Command{
		Name:    "boom",
		Summary: "test command that always fails",
		Usage:   "boom",
		Run: func(_ context.Context, _ *App, _ []string) error {
			return errors.New("intentional failure")
		},
	})

	code := app.Run(context.Background(), []string{"boom"})
	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
	if got := stderr.String(); !bytes.Contains([]byte(got), []byte("intentional failure")) {
		t.Errorf("stderr missing error text:\n%s", got)
	}
}

func TestExitCodeValues(t *testing.T) {
	t.Parallel()

	// These values are part of the CLI contract; guard them against accidental change.
	if ExitOK != 0 || ExitError != 1 || ExitUsage != 2 {
		t.Errorf("exit codes changed: ok=%d error=%d usage=%d", ExitOK, ExitError, ExitUsage)
	}
}
