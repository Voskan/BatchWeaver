package transform

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// copyTree recursively copies src into dst.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyTree(t, s, d)
			continue
		}
		b, err := os.ReadFile(s)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(d, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMaterializeRevertRoundTrip(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, fixtureDir(t, "proof", "fixture"), dir)

	svc := filepath.Join(dir, "svc", "svc.go")
	before, err := os.ReadFile(svc)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(context.Background(), Request{Patterns: []string{"./..."}, Dir: dir, ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(plan.Files))
	}
	if err := SavePlan(dir, plan); err != nil {
		t.Fatal(err)
	}

	res, err := Materialize(dir, "test", plan)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	after, _ := os.ReadFile(svc)
	if string(after) == string(before) {
		t.Fatal("materialize did not change the source file")
	}

	rev, err := Revert(dir, res.MaterializationID)
	if err != nil {
		t.Fatalf("revert: %v", err)
	}
	if len(rev.Conflicts) != 0 {
		t.Fatalf("unexpected revert conflicts: %v", rev.Conflicts)
	}
	restored, _ := os.ReadFile(svc)
	if string(restored) != string(before) {
		t.Fatal("revert did not restore the original bytes exactly")
	}
}

func TestMaterializePreconditionChanged(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, fixtureDir(t, "proof", "fixture"), dir)
	plan, err := BuildPlan(context.Background(), Request{Patterns: []string{"./..."}, Dir: dir, ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := SavePlan(dir, plan); err != nil {
		t.Fatal(err)
	}
	// Change the source after planning; materialization must refuse.
	svc := filepath.Join(dir, "svc", "svc.go")
	b, _ := os.ReadFile(svc)
	if err := os.WriteFile(svc, append(b, []byte("\n// drift\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(dir, "test", plan); err == nil {
		t.Fatal("expected precondition failure after source drift")
	}
}

func TestRevertConflictAfterUserEdit(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, fixtureDir(t, "proof", "fixture"), dir)
	plan, _ := BuildPlan(context.Background(), Request{Patterns: []string{"./..."}, Dir: dir, ToolVersion: "test"})
	_ = SavePlan(dir, plan)
	res, err := Materialize(dir, "test", plan)
	if err != nil {
		t.Fatal(err)
	}
	// Edit the materialized file, then revert: it must be reported as a conflict.
	svc := filepath.Join(dir, "svc", "svc.go")
	b, _ := os.ReadFile(svc)
	_ = os.WriteFile(svc, append(b, []byte("\n// user edit\n")...), 0o644)
	rev, err := Revert(dir, res.MaterializationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rev.Conflicts) == 0 {
		t.Fatal("expected a revert conflict after a post-materialization edit")
	}
}
