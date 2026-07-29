package analysis

import (
	"go/ast"
	"go/types"

	"github.com/Voskan/BatchWeaver/operation"
	"golang.org/x/tools/go/packages"
)

// rootPackagePath is the import path of BatchWeaver's root package, used to
// recognize typed declaration helpers statically.
const rootPackagePath = "github.com/Voskan/BatchWeaver"

// resolver resolves operation symbols to Go type objects using loaded package
// type information. It never executes code.
type resolver struct {
	byPath map[string]*packages.Package
}

// newResolver indexes loaded packages by import path, preferring non-test
// variants so a symbol resolves to its canonical package.
func newResolver(pkgs []*packages.Package) *resolver {
	m := make(map[string]*packages.Package)
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		if p.Types == nil {
			return
		}
		if existing, ok := m[p.PkgPath]; !ok || (isTestVariant(existing) && !isTestVariant(p)) {
			m[p.PkgPath] = p
		}
	})
	return &resolver{byPath: m}
}

// isTestVariant reports whether a package is a test variant.
func isTestVariant(p *packages.Package) bool {
	return len(p.ID) > 0 && (containsRune(p.ID, '[') || hasSuffix(p.ID, ".test"))
}

func containsRune(s string, r byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == r {
			return true
		}
	}
	return false
}

func hasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}

// resolveFunc resolves an operation symbol to its *types.Func, or reports false
// when it cannot be resolved in the loaded program.
func (r *resolver) resolveFunc(sym operation.Symbol) (*types.Func, bool) {
	p := r.byPath[sym.ImportPath()]
	if p == nil || p.Types == nil {
		return nil, false
	}
	scope := p.Types.Scope()
	if !sym.IsMethod() {
		fn, ok := scope.Lookup(sym.Name()).(*types.Func)
		return fn, ok
	}
	tn, ok := scope.Lookup(sym.Receiver()).(*types.TypeName)
	if !ok {
		return nil, false
	}
	recv := tn.Type()
	if sym.PointerReceiver() {
		recv = types.NewPointer(recv)
	}
	obj, _, _ := types.LookupFieldOrMethod(recv, true, p.Types, sym.Name())
	fn, ok := obj.(*types.Func)
	return fn, ok
}

// symbolFromExpr builds an operation.Symbol from a function reference or method
// expression, using type information. It handles package functions and method
// expressions such as (*Repository).GetUser.
func symbolFromExpr(info *types.Info, expr ast.Expr) (operation.Symbol, bool) {
	for {
		if paren, ok := expr.(*ast.ParenExpr); ok {
			expr = paren.X
			continue
		}
		break
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return operation.Symbol{}, false
	}
	obj := info.Uses[sel.Sel]
	if obj == nil {
		if s, ok := info.Selections[sel]; ok {
			obj = s.Obj()
		}
	}
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil {
		return operation.Symbol{}, false
	}
	return symbolFromFunc(fn)
}

// symbolFromFunc builds the canonical operation.Symbol for a *types.Func.
func symbolFromFunc(fn *types.Func) (operation.Symbol, bool) {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return operation.Symbol{}, false
	}
	pkg := fn.Pkg()
	if pkg == nil {
		return operation.Symbol{}, false
	}
	if recv := sig.Recv(); recv != nil {
		recvType := recv.Type()
		pointer := false
		if ptr, ok := recvType.(*types.Pointer); ok {
			pointer = true
			recvType = ptr.Elem()
		}
		named, ok := recvType.(*types.Named)
		if !ok || named.Obj().Pkg() == nil {
			return operation.Symbol{}, false
		}
		var s string
		if pointer {
			s = named.Obj().Pkg().Path() + ".(*" + named.Obj().Name() + ")." + fn.Name()
		} else {
			s = named.Obj().Pkg().Path() + ".(" + named.Obj().Name() + ")." + fn.Name()
		}
		sym, err := operation.ParseSymbol(s)
		return sym, err == nil
	}
	sym, err := operation.ParseSymbol(pkg.Path() + "." + fn.Name())
	return sym, err == nil
}
