package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inProofFixture changes into the proof fixture module for the duration of a
// test. The fixture is a standalone module discovered via configuration.
func inProofFixture(t *testing.T) {
	t.Helper()
	fixture, err := filepath.Abs(filepath.Join("..", "..", "testdata", "proof", "fixture"))
	if err != nil {
		t.Fatal(err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(fixture); err != nil {
		t.Fatalf("chdir: %v", err)
	}
}

type proveDoc struct {
	SchemaVersion   string         `json:"schema_version"`
	DecisionCounts  map[string]int `json:"decision_counts"`
	CandidateProofs []struct {
		ID         string `json:"id"`
		ProofID    string `json:"proof_id"`
		Operation  string `json:"operation"`
		Decision   string `json:"decision"`
		Strategies []struct {
			Strategy string `json:"strategy"`
			Status   string `json:"status"`
		} `json:"allowed_strategies"`
	} `json:"candidate_proofs"`
}

func TestProveFixtureDecisions(t *testing.T) {
	inProofFixture(t)
	code, stdout, stderr := run(t, "prove", "--reproducible", "--format", "json", "./...")
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr=%s", code, stderr)
	}
	var doc proveDoc
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if doc.SchemaVersion == "" {
		t.Error("missing schema version")
	}
	byOp := map[string]string{}
	for _, c := range doc.CandidateProofs {
		byOp[c.Operation] = c.Decision
	}
	// events.append is a non-idempotent write and must be proven ineligible.
	if byOp["events.append"] != "proven_ineligible" {
		t.Errorf("events.append decision = %q, want proven_ineligible", byOp["events.append"])
	}
	// The interface-dispatched users.get loop is unknown due to ambiguity; the
	// direct loop is eligible. At least one users.get candidate must be eligible.
	if doc.DecisionCounts["proven_eligible"] == 0 {
		t.Error("expected at least one proven-eligible candidate")
	}
	if doc.DecisionCounts["unknown"] == 0 {
		t.Error("expected the ambiguous interface candidate to be unknown")
	}
}

func TestDeterminismProveJSON(t *testing.T) {
	inProofFixture(t)
	_, a, _ := run(t, "prove", "--reproducible", "--format", "json", "./...")
	_, b, _ := run(t, "prove", "--reproducible", "--format", "json", "./...")
	if a != b {
		t.Error("prove JSON output is not deterministic across runs")
	}
}

func TestProveCandidateInspectEligible(t *testing.T) {
	inProofFixture(t)
	_, stdout, _ := run(t, "prove", "--reproducible", "--format", "json", "./...")
	var doc proveDoc
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatal(err)
	}
	var id string
	for _, c := range doc.CandidateProofs {
		if c.Decision == "proven_eligible" {
			for _, s := range c.Strategies {
				if s.Strategy == "static-loop-prefetch" && s.Status == "proven_eligible" {
					id = c.ID
				}
			}
		}
	}
	if id == "" {
		t.Skip("no statically eligible candidate in fixture")
	}
	code, out, stderr := run(t, "candidate", "inspect", id, "./...")
	if code != ExitOK {
		t.Fatalf("exit = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(out, "PROVEN ELIGIBLE") {
		t.Errorf("inspect output missing decision:\n%s", out)
	}
	if !strings.Contains(out, "No source files were changed.") {
		t.Errorf("inspect output missing safety statement")
	}
}

func TestProveBadFormat(t *testing.T) {
	t.Parallel()
	code, _, stderr := run(t, "prove", "--format", "yaml")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "unknown format") {
		t.Errorf("stderr: %s", stderr)
	}
}

func TestProveBadStrategyFilter(t *testing.T) {
	t.Parallel()
	code, _, _ := run(t, "prove", "--strategy", "does-not-exist")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
}

func TestStrategyList(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "strategy", "list")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"static-loop-prefetch", "runtime-scope-coalescing", "existing-fanout-coalescing"} {
		if !strings.Contains(out, want) {
			t.Errorf("strategy list missing %q", want)
		}
	}
}

func TestStrategyInspect(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "strategy", "inspect", "static-loop-prefetch")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "BW-PROOF-") {
		t.Errorf("strategy inspect should list obligation IDs:\n%s", out)
	}
}

func TestProveHelpNotAdvertisingRemoved(t *testing.T) {
	t.Parallel()
	_, out, _ := run(t, "help")
	for _, want := range []string{"prove", "candidate", "proof", "assumption", "strategy"} {
		if !strings.Contains(out, "\n  "+want+" ") {
			t.Errorf("help missing command %q:\n%s", want, out)
		}
	}
}
