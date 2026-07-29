package analysis

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"

	"github.com/Voskan/BatchWeaver/config"
	"github.com/Voskan/BatchWeaver/operation"
)

// declareFuncs are the Prompt 02 typed declaration helpers recognized statically.
var declareFuncs = map[string]struct{}{
	"MustDeclareMethod": {}, "DeclareMethod": {},
	"MustDeclareFunction": {}, "DeclareFunction": {},
}

// discoverOperations discovers operations from typed declarations and
// configuration, merges them with provenance and conflict diagnostics, and
// resolves symbol compatibility conservatively.
func discoverOperations(ctx context.Context, l *loaded, r *resolver) ([]Operation, []Diag, string) {
	typed, tdiags := discoverTyped(l)
	cfgOps, cdiags, digest := discoverConfig(ctx, l)

	byID := make(map[string]*Operation)
	order := []string{}
	add := func(op Operation) {
		if existing, ok := byID[op.ID]; ok {
			existing.Sources = append(existing.Sources, op.Sources...)
			return
		}
		cp := op
		byID[op.ID] = &cp
		order = append(order, op.ID)
	}
	for _, op := range typed {
		add(op)
	}
	var diags []Diag
	diags = append(diags, tdiags...)
	diags = append(diags, cdiags...)

	for _, cop := range cfgOps {
		if existing, ok := byID[cop.ID]; ok {
			// Configuration overrides typed declarations, but disagreement is
			// reported as a conflict.
			if cop.ScalarSymbol != "" && existing.ScalarSymbol != "" && cop.ScalarSymbol != existing.ScalarSymbol {
				diags = append(diags, conflictDiag(cop.ID, "scalar symbol", existing.ScalarSymbol, cop.ScalarSymbol, cop.Sources))
			}
			if cop.BatchSymbol != "" && existing.BatchSymbol != "" && cop.BatchSymbol != existing.BatchSymbol {
				diags = append(diags, conflictDiag(cop.ID, "batch symbol", existing.BatchSymbol, cop.BatchSymbol, cop.Sources))
			}
			existing.Sources = append(existing.Sources, cop.Sources...)
			if cop.ScalarSymbol != "" {
				existing.ScalarSymbol = cop.ScalarSymbol
			}
			if cop.BatchSymbol != "" {
				existing.BatchSymbol = cop.BatchSymbol
			}
			if cop.Kind != "" {
				existing.Kind = cop.Kind
			}
			continue
		}
		add(cop)
	}

	sort.Strings(order)
	ops := make([]Operation, 0, len(order))
	for _, id := range order {
		op := byID[id]
		diags = append(diags, resolveCompatibility(r, op)...)
		ops = append(ops, *op)
	}
	return ops, diags, digest
}

// discoverTyped scans package syntax for typed declaration helper calls.
func discoverTyped(l *loaded) ([]Operation, []Diag) {
	var ops []Operation
	var diags []Diag
	for _, p := range l.pkgs {
		if p.TypesInfo == nil {
			continue
		}
		for _, file := range p.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if _, isDeclare := declareFuncs[sel.Sel.Name]; !isDeclare {
					return true
				}
				fn, ok := p.TypesInfo.Uses[sel.Sel].(*types.Func)
				if !ok || fn.Pkg() == nil || fn.Pkg().Path() != rootPackagePath {
					return true
				}
				if len(call.Args) < 3 {
					return true
				}
				id := extractOpID(call.Args[0])
				if id == "" {
					return true
				}
				op := Operation{
					ID:      id,
					Sources: []DeclarationSource{{Kind: "typed", Location: l.location(p.Fset, call.Pos())}},
				}
				if sym, ok := symbolFromExpr(p.TypesInfo, call.Args[1]); ok {
					op.ScalarSymbol = sym.String()
				}
				if sym, ok := symbolFromExpr(p.TypesInfo, call.Args[2]); ok {
					op.BatchSymbol = sym.String()
				}
				ops = append(ops, op)
				return true
			})
		}
	}
	return ops, diags
}

// extractOpID finds a MustParseID/ParseID string literal within an expression.
func extractOpID(expr ast.Expr) string {
	id := ""
	ast.Inspect(expr, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "MustParseID" && sel.Sel.Name != "ParseID") || len(call.Args) != 1 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if s, err := strconv.Unquote(lit.Value); err == nil {
			id = s
		}
		return false
	})
	return id
}

// discoverConfig discovers operations from BatchWeaver configuration.
func discoverConfig(ctx context.Context, l *loaded) ([]Operation, []Diag, string) {
	res, err := config.Load(ctx, config.LoadOptions{
		WorkingDirectory: l.root, Discover: true, RepositoryRoot: l.root,
	})
	if err != nil || !res.Found {
		return nil, nil, ""
	}
	var diags []Diag
	for _, d := range res.Diagnostics.Sorted().Diagnostics() {
		diags = append(diags, Diag{
			Code: "BW3100", Severity: d.Severity.String(),
			Message:  fmt.Sprintf("configuration: %s", d.Message),
			Location: l.pc.portable(d.Range.Start.File), Phase: "declaration-discovery",
			Fingerprint: shortDigest("BW3100", string(d.Code), d.Message),
		})
	}
	if res.HasErrors() {
		return nil, diags, ""
	}
	var ops []Operation
	for _, spec := range res.Catalog.List() {
		op := Operation{
			ID:      spec.ID().String(),
			Kind:    spec.Semantics().Kind().String(),
			Sources: []DeclarationSource{{Kind: "configuration", Location: spec.Source().Range.Start.String()}},
		}
		if s := spec.ScalarSymbol(); !s.IsZero() {
			op.ScalarSymbol = s.String()
		}
		if s := spec.BatchSymbol(); !s.IsZero() {
			op.BatchSymbol = s.String()
		}
		ops = append(ops, op)
	}
	return ops, diags, res.Digest
}

// resolveCompatibility resolves scalar and batch symbols and sets the operation
// compatibility status conservatively.
func resolveCompatibility(r *resolver, op *Operation) []Diag {
	var diags []Diag
	op.Compatibility = "valid"
	resolved := 0
	symbols := 0

	check := func(symStr, kind, unresolvedCode string) *types.Func {
		if symStr == "" {
			return nil
		}
		symbols++
		sym, err := operation.ParseSymbol(symStr)
		if err != nil {
			op.Compatibility = "invalid"
			diags = append(diags, opDiag("BW3105", "error", op.ID, fmt.Sprintf("invalid %s symbol %q", kind, symStr)))
			return nil
		}
		fn, ok := r.resolveFunc(sym)
		if !ok {
			op.Compatibility = "unresolved"
			diags = append(diags, opDiag(unresolvedCode, "warning", op.ID, fmt.Sprintf("%s symbol %q could not be resolved in the loaded program", kind, symStr)))
			return nil
		}
		resolved++
		return fn
	}
	scalar := check(op.ScalarSymbol, "scalar", "BW3102")
	batch := check(op.BatchSymbol, "batch", "BW3103")

	if scalar != nil && !returnsError(scalar) {
		op.Compatibility = "invalid"
		diags = append(diags, opDiag("BW3108", "error", op.ID, "scalar function must return an error as its last result"))
	}
	if batch != nil && !returnsError(batch) {
		op.Compatibility = "invalid"
		diags = append(diags, opDiag("BW3108", "error", op.ID, "batch function must return an error as its last result"))
	}
	if symbols == 0 {
		op.Compatibility = "unresolved"
	}
	return diags
}

// returnsError reports whether a function's last result is the error type.
func returnsError(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Results().Len() == 0 {
		return false
	}
	last := sig.Results().At(sig.Results().Len() - 1).Type()
	return types.Identical(last, errorType)
}

var errorType = types.Universe.Lookup("error").Type()

// location renders a portable "file:line:col" for a token position.
func (l *loaded) location(fset *token.FileSet, pos token.Pos) string {
	p := fset.Position(pos)
	return fmt.Sprintf("%s:%d:%d", l.pc.portable(p.Filename), p.Line, p.Column)
}

// conflictDiag builds a declaration-conflict diagnostic.
func conflictDiag(id, field, a, b string, sources []DeclarationSource) Diag {
	loc := ""
	if len(sources) > 0 {
		loc = sources[len(sources)-1].Location
	}
	return Diag{
		Code: "BW3101", Severity: "error",
		Message:  fmt.Sprintf("operation %q: conflicting %s (%q vs %q)", id, field, a, b),
		Location: loc, Phase: "declaration-discovery",
		Fingerprint: shortDigest("BW3101", id, field, a, b),
	}
}

// opDiag builds an operation-scoped diagnostic.
func opDiag(code, severity, id, msg string) Diag {
	return Diag{
		Code: code, Severity: severity,
		Message:     fmt.Sprintf("operation %q: %s", id, msg),
		Phase:       "declaration-discovery",
		Fingerprint: shortDigest(code, id, msg),
	}
}
