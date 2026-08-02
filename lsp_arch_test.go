package batchweaver_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoGoplsInternalImports enforces the editor architecture rule that the
// editor, LSP, and daemon layers must not import gopls internal packages. The
// proxy composes gopls only over the public LSP wire protocol. See
// docs/adr/0072-standalone-lsp-server.md.
func TestNoGoplsInternalImports(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	const banned = "golang.org/x/tools/gopls/internal"
	dirs := []string{
		filepath.Join(root, "internal", "lsp"),
		filepath.Join(root, "internal", "editor"),
		filepath.Join(root, "internal", "daemon"),
	}
	for _, dir := range dirs {
		walkGoFiles(t, dir, func(path string, imports []string) {
			for _, imp := range imports {
				if strings.HasPrefix(imp, banned) {
					t.Errorf("%s imports banned gopls internal package %q", path, imp)
				}
			}
		})
	}
}

// walkGoFiles parses every non-test Go file under dir and reports its imports.
func walkGoFiles(t *testing.T, dir string, fn func(path string, imports []string)) {
	t.Helper()
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		f, _ := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if f == nil {
			return nil
		}
		var imports []string
		for _, spec := range f.Imports {
			imports = append(imports, strings.Trim(spec.Path.Value, `"`))
		}
		fn(path, imports)
		return nil
	})
}
