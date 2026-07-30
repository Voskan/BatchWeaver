package transform

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// loaded holds packages loaded with full syntax and type information for
// rewriting, together with the workspace root used to portablize paths.
type loaded struct {
	root string
	fset *token.FileSet
	pkgs []*packages.Package
	// byFile maps a workspace-relative slash path to the file it was parsed from.
	byFile map[string]*fileInfo
}

// fileInfo bundles a parsed file with its owning package and on-disk bytes.
type fileInfo struct {
	pkg  *packages.Package
	file *ast.File
	abs  string
	rel  string
	src  []byte
}

// loadForRewrite loads the requested packages with syntax and types. It returns
// a typed error when loading cannot proceed at all.
func loadForRewrite(ctx context.Context, dir string, patterns []string, bc BuildConfig) (*loaded, error) {
	root, err := moduleRoot(dir)
	if err != nil {
		return nil, err
	}
	env := os.Environ()
	if bc.GOOS != "" {
		env = append(env, "GOOS="+bc.GOOS)
	}
	if bc.GOARCH != "" {
		env = append(env, "GOARCH="+bc.GOARCH)
	}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports |
			packages.NeedDeps | packages.NeedModule,
		Context: ctx,
		Dir:     dir,
		Env:     env,
		Tests:   bc.Tests,
		Fset:    token.NewFileSet(),
	}
	if len(bc.Tags) > 0 {
		cfg.BuildFlags = []string{"-tags=" + strings.Join(bc.Tags, ",")}
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}
	l := &loaded{root: root, fset: cfg.Fset, pkgs: pkgs, byFile: map[string]*fileInfo{}}
	for _, p := range pkgs {
		for i, f := range p.Syntax {
			if i >= len(p.GoFiles) {
				continue
			}
			abs := p.GoFiles[i]
			src, readErr := os.ReadFile(abs)
			if readErr != nil {
				continue
			}
			rel := portableRel(root, abs)
			l.byFile[rel] = &fileInfo{pkg: p, file: f, abs: abs, rel: rel, src: src}
		}
	}
	return l, nil
}

// moduleRoot walks upward from dir to find the directory containing go.mod.
func moduleRoot(dir string) (string, error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd
	}
	d, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(d, "go.mod")); statErr == nil {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("no go.mod found from %s", dir)
		}
		d = parent
	}
}

// portableRel returns a slash-separated workspace-relative path, or the base
// name when the file is outside the root.
func portableRel(root, abs string) string {
	rel, err := filepath.Rel(root, filepath.Clean(abs))
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(filepath.Base(abs))
	}
	return filepath.ToSlash(rel)
}

// findScalarCall finds the method-call expression on the given 1-based line in
// the workspace-relative file whose selector resolves to scalarSym. Matching by
// line and resolved symbol is robust against differences between the AST call
// position and the position the analysis stage recorded. It returns the file
// info, the call, and the enclosing node path (innermost first).
func (l *loaded) findScalarCall(relFile string, line int, scalarSym string) (*fileInfo, *ast.CallExpr, []ast.Node, bool) {
	fi, ok := l.byFile[relFile]
	if !ok {
		return nil, nil, nil, false
	}
	var found *ast.CallExpr
	ast.Inspect(fi.file, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pos := l.fset.Position(call.Lparen)
		if pos.Line != line && l.fset.Position(call.Pos()).Line != line {
			return true
		}
		if l.selectorResolvesTo(fi, sel, scalarSym) {
			found = call
		}
		return true
	})
	if found == nil {
		return fi, nil, nil, false
	}
	path, _ := astutil.PathEnclosingInterval(fi.file, found.Pos(), found.End())
	return fi, found, path, true
}

// funcDeclName renders a readable name for a function declaration.
func funcDeclName(fn *ast.FuncDecl) string {
	if fn == nil {
		return ""
	}
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		return "(" + typeString(fn.Recv.List[0].Type) + ")." + fn.Name.Name
	}
	return fn.Name.Name
}

// typeString renders an AST type expression briefly for anchors.
func typeString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	default:
		return "?"
	}
}
