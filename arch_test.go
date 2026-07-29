package batchweaver_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/Voskan/BatchWeaver"

// TestDependencyDirection enforces BatchWeaver's documented one-way dependency
// direction by scanning package imports with the standard library only, so the
// check is not fragile against external tooling. See
// docs/architecture/package-boundaries.md.
func TestDependencyDirection(t *testing.T) {
	t.Parallel()

	// For each package directory, the module-local import prefixes it must not use.
	// An entry equal to modulePath bans importing the root package itself.
	forbidden := map[string][]string{
		"diagnostics": {modulePath, "/config", "/operation", "/runtime", "/internal", "/cmd"},
		"operation":   {modulePath, "/config", "/runtime", "/internal", "/cmd"},
		// runtime is a higher layer: it may import the root package, operation,
		// diagnostics, and internal packages, but not config or the CLI.
		"runtime": {"/config", "/cmd"},
		"config":  {"/runtime", "/cmd"},
		".":       {"/config", "/diagnostics", "/runtime", "/internal", "/cmd"},
	}

	root := repoRoot(t)
	for dir, rules := range forbidden {
		imports := packageImports(t, filepath.Join(root, dir))
		for _, imp := range imports {
			if !strings.HasPrefix(imp, modulePath) {
				continue // standard library or external, always allowed
			}
			for _, rule := range rules {
				if violates(imp, rule) {
					t.Errorf("package %q must not import %q (rule %q)", dir, imp, rule)
				}
			}
		}
	}
}

// violates reports whether module-local import imp breaks rule. A rule equal to
// modulePath matches the root package exactly; a rule beginning with "/" matches
// modulePath+rule and any subpackage beneath it.
func violates(imp, rule string) bool {
	if rule == modulePath {
		return imp == modulePath
	}
	full := modulePath + rule
	return imp == full || strings.HasPrefix(imp, full+"/")
}

// packageImports returns the deduplicated import paths of the non-test Go files
// directly in dir.
func packageImports(t *testing.T, dir string) []string {
	t.Helper()
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	seen := map[string]struct{}{}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, "\"")
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				out = append(out, p)
			}
		}
	}
	return out
}

// repoRoot returns the repository root (the directory containing go.mod).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
