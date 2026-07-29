package project

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Voskan/BatchWeaver/internal/filesystem"
)

// mkTree creates the given relative directories and files under root.
func mkTree(t *testing.T, root string, dirs, files []string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	for _, f := range files {
		path := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir for %s: %v", f, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
}

func TestFindRootFromSubdirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mkTree(t, root, []string{"internal/cli"}, []string{"go.mod"})

	loc := New(filesystem.OS())
	start := filepath.Join(root, "internal", "cli")
	got, err := loc.FindRoot(start)
	if err != nil {
		t.Fatalf("FindRoot: %v", err)
	}

	// Resolve symlinks so macOS /var vs /private/var differences do not fail the test.
	wantResolved, _ := filepath.EvalSymlinks(root)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != wantResolved {
		t.Errorf("FindRoot = %q, want %q", gotResolved, wantResolved)
	}
}

func TestFindRootUsesGitMarker(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mkTree(t, root, []string{".git", "pkg"}, nil)

	loc := New(nil) // nil defaults to OS filesystem
	got, err := loc.FindRoot(filepath.Join(root, "pkg"))
	if err != nil {
		t.Fatalf("FindRoot: %v", err)
	}
	gotResolved, _ := filepath.EvalSymlinks(got)
	wantResolved, _ := filepath.EvalSymlinks(root)
	if gotResolved != wantResolved {
		t.Errorf("FindRoot = %q, want %q", gotResolved, wantResolved)
	}
}

func TestFindRootNotFound(t *testing.T) {
	t.Parallel()

	// A temp dir with no markers and no marked ancestors. On the standard test
	// runners t.TempDir() has no go.mod/.git above it.
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	loc := New(nil)
	_, err := loc.FindRoot(sub)
	if err == nil {
		t.Fatalf("FindRoot = nil error, want ErrRootNotFound")
	}
	if !errors.Is(err, ErrRootNotFound) {
		t.Errorf("FindRoot error = %v, want it to wrap ErrRootNotFound", err)
	}
}

func TestFindRootAcceptsRelativeInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mkTree(t, root, nil, []string{"go.mod"})

	// Change into the temp dir; restore afterward. Not parallel-safe with other
	// tests that chdir, but this test controls its own directory.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	loc := New(nil)
	got, err := loc.FindRoot(".")
	if err != nil {
		t.Fatalf("FindRoot: %v", err)
	}
	gotResolved, _ := filepath.EvalSymlinks(got)
	wantResolved, _ := filepath.EvalSymlinks(root)
	if gotResolved != wantResolved {
		t.Errorf("FindRoot(\".\") = %q, want %q", gotResolved, wantResolved)
	}
}

func TestRootFromWorkingDir(t *testing.T) {
	// This test relies on the module's own go.mod being discoverable from the
	// package directory, which is always true when the test binary runs.
	if runtime.GOOS == "js" {
		t.Skip("working directory semantics differ on js/wasm")
	}

	loc := New(nil)
	got, err := loc.RootFromWorkingDir()
	if err != nil {
		t.Fatalf("RootFromWorkingDir: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("RootFromWorkingDir = %q, want an absolute path", got)
	}
	if _, err := os.Stat(filepath.Join(got, "go.mod")); err != nil {
		t.Errorf("discovered root %q has no go.mod: %v", got, err)
	}
}
