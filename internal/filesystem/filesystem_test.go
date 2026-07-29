package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOSStatAndReadFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	want := []byte("batchweaver")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	fsys := OS()

	info, err := fsys.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.IsDir() {
		t.Errorf("Stat reported directory for a regular file")
	}

	got, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("ReadFile = %q, want %q", got, want)
	}
}

func TestOSAbs(t *testing.T) {
	t.Parallel()

	fsys := OS()
	abs, err := fsys.Abs(".")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}
	if !filepath.IsAbs(abs) {
		t.Errorf("Abs(%q) = %q, want an absolute path", ".", abs)
	}
}

func TestOSGetwd(t *testing.T) {
	t.Parallel()

	fsys := OS()
	wd, err := fsys.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if !filepath.IsAbs(wd) {
		t.Errorf("Getwd() = %q, want an absolute path", wd)
	}
}
