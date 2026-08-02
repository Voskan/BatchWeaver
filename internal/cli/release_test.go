package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := New(&stdout, &stderr).Run(context.Background(), []string{"version", "--json"})
	if code != ExitOK {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	var info struct {
		Version      string `json:"version"`
		RuntimeABI   string `json:"runtime_abi"`
		ConfigSchema int    `json:"config_schema"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info.Version == "" || !strings.HasPrefix(info.RuntimeABI, "batchweaver.runtime/") || info.ConfigSchema != 1 {
		t.Fatalf("incomplete version JSON: %s", stdout.String())
	}
}

func TestReleaseRejectsNonSnapshotBuild(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := New(&stdout, &stderr).Run(context.Background(), []string{"release", "build", "--output", t.TempDir()})
	if code != ExitError || !strings.Contains(stderr.String(), "BW9018") {
		t.Fatalf("exit=%d stderr=%q", code, stderr.String())
	}
}

func TestCompatibilityReportDoesNotPromoteUntested(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := New(&stdout, &stderr).Run(context.Background(), []string{"compatibility", "report"})
	if code != ExitOK {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "not-tested") || !strings.Contains(stdout.String(), "not-supported") {
		t.Fatalf("report loses explicit unsupported states: %s", stdout.String())
	}
}
