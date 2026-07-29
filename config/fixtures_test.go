package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestValidFixtures loads every fixture under testdata/config/valid and requires
// that none produces error diagnostics.
func TestValidFixtures(t *testing.T) {
	t.Parallel()
	dir := "../testdata/config/valid"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	for _, e := range entries {
		e := e
		t.Run(e.Name(), func(t *testing.T) {
			t.Parallel()
			res, err := Load(context.Background(), LoadOptions{Path: filepath.Join(dir, e.Name())})
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if res.HasErrors() {
				t.Errorf("valid fixture %q has errors:\n%s", e.Name(), renderDiags(res.Diagnostics))
			}
		})
	}
}

// TestInvalidFixtures loads every fixture under testdata/config/invalid and
// requires that each produces at least one error diagnostic.
func TestInvalidFixtures(t *testing.T) {
	t.Parallel()
	dir := "../testdata/config/invalid"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	for _, e := range entries {
		e := e
		t.Run(e.Name(), func(t *testing.T) {
			t.Parallel()
			res, err := Load(context.Background(), LoadOptions{Path: filepath.Join(dir, e.Name())})
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if !res.HasErrors() {
				t.Errorf("invalid fixture %q produced no errors", e.Name())
			}
		})
	}
}
