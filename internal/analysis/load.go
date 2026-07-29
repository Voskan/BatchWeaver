package analysis

import (
	"context"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// BuildContext describes the build configuration under which packages are
// loaded. Its zero value is completed from the current Go environment; the
// analysis never silently changes a user's configuration and always reports the
// effective context.
type BuildContext struct {
	// GOOS is the target operating system; empty means the current GOOS.
	GOOS string
	// GOARCH is the target architecture; empty means the current GOARCH.
	GOARCH string
	// CGOEnabled reports whether cgo is enabled.
	CGOEnabled bool
	// Tags are additional build tags.
	Tags []string
	// Tests reports whether test variants are loaded.
	Tests bool
}

// withDefaults returns a copy with empty fields filled from the environment.
func (bc BuildContext) withDefaults() BuildContext {
	if bc.GOOS == "" {
		bc.GOOS = envOr("GOOS", runtime.GOOS)
	}
	if bc.GOARCH == "" {
		bc.GOARCH = envOr("GOARCH", runtime.GOARCH)
	}
	return bc
}

// digest returns a stable digest of the build context.
func (bc BuildContext) digest() string {
	return "sha256:" + shortDigest(bc.GOOS, bc.GOARCH, strconv.FormatBool(bc.CGOEnabled),
		strings.Join(bc.Tags, ","), strconv.FormatBool(bc.Tests))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loaded holds the result of package loading.
type loaded struct {
	pkgs   []*packages.Package
	root   string
	module string
	pc     pathContext
	bc     BuildContext
}

// loadPackages loads the packages matching patterns under the given build
// context and working directory. It surfaces load and type errors as
// diagnostics rather than dropping packages silently.
func loadPackages(ctx context.Context, patterns []string, bc BuildContext, dir string) (*loaded, []Diag, error) {
	bc = bc.withDefaults()

	env := os.Environ()
	env = append(env, "GOOS="+bc.GOOS, "GOARCH="+bc.GOARCH)
	if bc.CGOEnabled {
		env = append(env, "CGO_ENABLED=1")
	} else {
		env = append(env, "CGO_ENABLED=0")
	}

	cfg := &packages.Config{
		Context: ctx,
		Dir:     dir,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedDeps | packages.NeedTypes |
			packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedModule |
			packages.NeedTypesSizes,
		Tests: bc.Tests,
		Env:   env,
	}
	if len(bc.Tags) > 0 {
		cfg.BuildFlags = []string{"-tags=" + strings.Join(bc.Tags, ",")}
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, nil, fmt.Errorf("load packages: %w", err)
	}

	l := &loaded{
		pkgs: pkgs,
		bc:   bc,
		pc: pathContext{
			goroot:     build.Default.GOROOT,
			gomodcache: gomodcache(),
		},
	}
	// Resolve the main module root for repository-relative paths.
	for _, p := range pkgs {
		if p.Module != nil && p.Module.Main && p.Module.Dir != "" {
			l.root = p.Module.Dir
			l.module = p.Module.Path
			break
		}
	}
	if l.root == "" {
		if wd, werr := os.Getwd(); werr == nil {
			l.root = wd
		}
	}
	l.pc.root = l.root

	var diags []Diag
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			diags = append(diags, Diag{
				Code:        "BW3000",
				Severity:    "error",
				Message:     fmt.Sprintf("package %q: %s", p.PkgPath, e.Msg),
				Location:    l.pc.portable(errorPos(e)),
				Fingerprint: shortDigest("BW3000", p.PkgPath, e.Msg),
				Phase:       "package-loading",
			})
		}
	})
	return l, diags, nil
}

// errorPos extracts a file path from a packages error position ("file:line:col").
func errorPos(e packages.Error) string {
	if e.Pos == "" {
		return ""
	}
	if i := strings.IndexByte(e.Pos, ':'); i > 0 {
		return e.Pos[:i]
	}
	return e.Pos
}

// gomodcache returns the module cache directory.
func gomodcache() string {
	if v := os.Getenv("GOMODCACHE"); v != "" {
		return v
	}
	gp := os.Getenv("GOPATH")
	if gp == "" {
		gp = build.Default.GOPATH
	}
	if gp == "" {
		return ""
	}
	// GOPATH may be a list; use the first entry.
	if i := strings.IndexByte(gp, filepath.ListSeparator); i > 0 {
		gp = gp[:i]
	}
	return filepath.Join(gp, "pkg", "mod")
}
