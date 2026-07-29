package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestVersionCommandOutput(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := run(t, "version")
	if code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}

	wantContains := []string{
		"BatchWeaver ",
		"Go: " + runtime.Version(),
		"Platform: " + runtime.GOOS + "/" + runtime.GOARCH,
		"Commit: ",
		"Build date: ",
	}
	for _, want := range wantContains {
		if !strings.Contains(stdout, want) {
			t.Errorf("version output missing %q:\n%s", want, stdout)
		}
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Errorf("version output should end with a newline:\n%q", stdout)
	}
}
