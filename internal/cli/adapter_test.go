package cli

import (
	"strings"
	"testing"
)

func TestAdapterList(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "adapter", "list")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"database/sql", "pgx", "redis", "exact-key-read-synthesis"} {
		if !strings.Contains(out, want) {
			t.Errorf("adapter list missing %q", want)
		}
	}
}

func TestAdapterInspectSynthesizes(t *testing.T) {
	t.Parallel()
	code, out, stderr := run(t, "adapter", "inspect", "--sql", "SELECT id, name FROM users WHERE id = $1", "--key-type", "bigint")
	if code != ExitOK {
		t.Fatalf("exit = %d stderr=%s", code, stderr)
	}
	for _, want := range []string{"unnest($1::bigint[]) WITH ORDINALITY", "LEFT JOIN users", "sql.ErrNoRows"} {
		if !strings.Contains(out, want) {
			t.Errorf("inspect output missing %q:\n%s", want, out)
		}
	}
}

func TestAdapterExplainRejects(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "adapter", "explain", "--sql", "SELECT * FROM users WHERE id = $1")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "rejected") || !strings.Contains(out, "BW6105") {
		t.Errorf("explain output:\n%s", out)
	}
}

func TestAdapterVerify(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "adapter", "verify")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "PASS") || !strings.Contains(out, "bwcontract_") {
		t.Errorf("verify output:\n%s", out)
	}
}

func TestAdapterDoctor(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "adapter", "doctor")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "database/sql") || !strings.Contains(out, "Runtime ABI") {
		t.Errorf("doctor output:\n%s", out)
	}
}

func TestAdapterUnknownSubcommand(t *testing.T) {
	t.Parallel()
	code, _, _ := run(t, "adapter", "frobnicate")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
}
