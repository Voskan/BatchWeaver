package transform

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

// loopSite is a fully resolved static-loop-prefetch candidate.
type loopSite struct {
	fi        *fileInfo
	fn        *ast.FuncDecl
	rng       *ast.RangeStmt
	assign    *ast.AssignStmt
	call      *ast.CallExpr
	recvExpr  ast.Expr
	ctxArg    ast.Expr
	keyArg    ast.Expr
	valueName string
	errName   string
	batchName string
	keyType   string
	errReturn *ast.ReturnStmt
	names     genNames
}

// genNames are the deterministic generated identifiers for one transformation.
type genNames struct {
	keys, values, err, index string
}

// locateLoop resolves and validates the supported static-loop-prefetch shape for
// a certificate. It returns the resolved site, or a skip reason when the shape is
// not supported. It never guesses: any deviation from the certified, supported
// form is a conservative rejection.
func (l *loaded) locateLoop(loc, scalarSym, batchSym string) (*loopSite, string) {
	relFile, line, _, ok := parseLocation(loc)
	if !ok {
		return nil, SkipAnchorUnresolved
	}
	fi, call, path, ok := l.findScalarCall(relFile, line, scalarSym)
	if !ok {
		return nil, SkipAnchorUnresolved
	}

	var rng *ast.RangeStmt
	var fn *ast.FuncDecl
	for _, n := range path {
		switch t := n.(type) {
		case *ast.RangeStmt:
			if rng == nil {
				rng = t
			}
		case *ast.FuncLit:
			// A closure between the loop and the function is not supported here.
			if rng == nil {
				return nil, SkipUnsupportedLoopForm
			}
		case *ast.FuncDecl:
			fn = t
		}
	}
	if rng == nil || fn == nil {
		return nil, SkipUnsupportedLoopForm
	}

	// The range must be over a slice or array (or pointer to array).
	if !l.isSliceOrArray(fi, rng.X) {
		return nil, SkipUnsupportedLoopForm
	}

	// The scalar call must be the RHS of the first body statement, an := assign
	// with a value and error on the left.
	if len(rng.Body.List) < 2 {
		return nil, SkipUnsupportedLoopForm
	}
	assign, ok := rng.Body.List[0].(*ast.AssignStmt)
	if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 || assign.Rhs[0] != call {
		return nil, SkipUnsupportedLoopForm
	}
	valueIdent, ok1 := assign.Lhs[0].(*ast.Ident)
	errIdent, ok2 := assign.Lhs[1].(*ast.Ident)
	if !ok1 || !ok2 || valueIdent.Name == "_" || errIdent.Name == "_" {
		return nil, SkipUnsupportedLoopForm
	}

	// The scalar call must be a method selector receiver.Scalar(ctx, key).
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) != 2 {
		return nil, SkipUnsupportedLoopForm
	}
	if !l.selectorResolvesTo(fi, sel, scalarSym) {
		return nil, SkipUnsupportedLoopForm
	}
	if !l.isContextType(fi, call.Args[0]) {
		return nil, SkipUnsupportedLoopForm
	}

	// The second body statement must be `if err != nil { return ... }`.
	errReturn, ok := matchErrGuard(rng.Body.List[1], errIdent.Name)
	if !ok {
		return nil, SkipUnsupportedLoopForm
	}

	batchName := symbolMethodName(batchSym)
	if batchName == "" {
		return nil, SkipBindingUnavailable
	}
	keyType, ok := l.batchKeyType(fi, sel.X, batchName)
	if !ok {
		return nil, SkipBindingUnavailable
	}

	site := &loopSite{
		fi: fi, fn: fn, rng: rng, assign: assign, call: call,
		recvExpr: sel.X, ctxArg: call.Args[0], keyArg: call.Args[1],
		valueName: valueIdent.Name, errName: errIdent.Name,
		batchName: batchName, keyType: keyType, errReturn: errReturn,
	}
	site.names = l.allocNames(fn)
	return site, ""
}

// generate produces the transformed file bytes by replacing the range loop with
// the prefetch preamble plus a modified loop, splicing generated code into the
// original source and leaving all other bytes untouched.
func (l *loaded) generate(site *loopSite) (transformed []byte, replacement string, startOff, endOff int) {
	src := site.fi.src
	off := func(p token.Pos) int { return l.fset.Position(p).Offset }

	xSrc := srcOf(src, off, site.rng.X)
	keySrc := srcOf(src, off, site.keyArg)
	ctxSrc := srcOf(src, off, site.ctxArg)
	recvSrc := srcOf(src, off, site.recvExpr)
	keyVarSrc := srcOf(src, off, site.rng.Key)
	valVarSrc := srcOf(src, off, site.rng.Value)

	n := site.names
	var b strings.Builder
	// Phase: collect keys in source iteration order.
	usedByKey := func(name string) bool { return identUsed(site.keyArg, name) }
	fmt.Fprintf(&b, "%s := make([]%s, 0, len(%s))\n", n.keys, site.keyType, xSrc)
	fmt.Fprintf(&b, "for %s, %s := range %s {\n", keyVarSrc, valVarSrc, xSrc)
	b.WriteString(discardUnused(keyVarSrc, valVarSrc, usedByKey))
	fmt.Fprintf(&b, "%s = append(%s, %s)\n}\n", n.keys, n.keys, keySrc)
	// Phase: invoke the declared batch provider once.
	fmt.Fprintf(&b, "%s, %s := %s.%s(%s, %s)\n", n.values, n.err, recvSrc, site.batchName, ctxSrc, n.keys)
	// Phase: validate the global batch result, mirroring the scalar error return.
	fmt.Fprintf(&b, "if %s != nil {\n%s\n}\n", n.err, l.batchErrReturn(site, n.err))
	fmt.Fprintf(&b, "%s := 0\n", n.index)
	// Phase: replay original loop in source order with reconstructed outcomes.
	rest := site.rng.Body.List[1:]
	usedInBody := func(name string) bool { return identUsedInStmts(rest, name) }
	fmt.Fprintf(&b, "for %s, %s := range %s {\n", keyVarSrc, valVarSrc, xSrc)
	b.WriteString(discardUnused(keyVarSrc, valVarSrc, usedInBody))
	fmt.Fprintf(&b, "%s, %s := %s[%s], error(nil)\n%s++\n", site.valueName, site.errName, n.values, n.index, n.index)
	// Copy the remainder of the original body verbatim (from the err guard on).
	bodyStart := off(site.rng.Body.List[1].Pos())
	bodyEnd := off(site.rng.Body.Rbrace)
	b.Write(src[bodyStart:bodyEnd])
	b.WriteString("}\n")

	startOff = off(site.rng.Pos())
	endOff = off(site.rng.End())
	replacement = b.String()

	out := make([]byte, 0, len(src)+len(replacement))
	out = append(out, src[:startOff]...)
	out = append(out, replacement...)
	out = append(out, src[endOff:]...)
	return out, replacement, startOff, endOff
}

// discardUnused emits `_ = name` statements for non-blank range variables that
// the loop body no longer references, preventing "declared and not used" errors.
func discardUnused(keyVarSrc, valVarSrc string, used func(name string) bool) string {
	var b strings.Builder
	for _, v := range []string{keyVarSrc, valVarSrc} {
		if v == "" || v == "_" {
			continue
		}
		if !used(v) {
			fmt.Fprintf(&b, "_ = %s\n", v)
		}
	}
	return b.String()
}

// identUsed reports whether expr references an identifier named name.
func identUsed(expr ast.Node, name string) bool {
	used := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == name {
			used = true
		}
		return true
	})
	return used
}

// identUsedInStmts reports whether any statement references an identifier named
// name.
func identUsedInStmts(stmts []ast.Stmt, name string) bool {
	for _, s := range stmts {
		if identUsed(s, name) {
			return true
		}
	}
	return false
}

// batchErrReturn renders the batch-error return statement, mirroring the scalar
// error branch but substituting the batch error variable.
func (l *loaded) batchErrReturn(site *loopSite, batchErr string) string {
	off := func(p token.Pos) int { return l.fset.Position(p).Offset }
	var parts []string
	for _, e := range site.errReturn.Results {
		if id, ok := e.(*ast.Ident); ok && id.Name == site.errName {
			parts = append(parts, batchErr)
			continue
		}
		parts = append(parts, srcOf(site.fi.src, off, e))
	}
	return "return " + strings.Join(parts, ", ")
}

// srcOf returns the source bytes of a node.
func srcOf(src []byte, off func(token.Pos) int, n ast.Node) string {
	if n == nil {
		return ""
	}
	return string(src[off(n.Pos()):off(n.End())])
}

// parseLocation splits "file:line:col".
func parseLocation(loc string) (file string, line, col int, ok bool) {
	i := strings.LastIndexByte(loc, ':')
	if i < 0 {
		return "", 0, 0, false
	}
	j := strings.LastIndexByte(loc[:i], ':')
	if j < 0 {
		return "", 0, 0, false
	}
	col, err1 := strconv.Atoi(loc[i+1:])
	line, err2 := strconv.Atoi(loc[j+1 : i])
	if err1 != nil || err2 != nil {
		return "", 0, 0, false
	}
	return loc[:j], line, col, true
}

// symbolMethodName returns the method name from an operation.Symbol string.
func symbolMethodName(sym string) string {
	if i := strings.LastIndexByte(sym, '.'); i >= 0 {
		return sym[i+1:]
	}
	return sym
}

// isSliceOrArray reports whether the range expression is a slice, array, or
// pointer to array.
func (l *loaded) isSliceOrArray(fi *fileInfo, x ast.Expr) bool {
	if fi.pkg.TypesInfo == nil {
		return false
	}
	t := fi.pkg.TypesInfo.TypeOf(x)
	if t == nil {
		return false
	}
	switch u := t.Underlying().(type) {
	case *types.Slice, *types.Array:
		return true
	case *types.Pointer:
		_, ok := u.Elem().Underlying().(*types.Array)
		return ok
	default:
		return false
	}
}

// isContextType reports whether an argument has type context.Context.
func (l *loaded) isContextType(fi *fileInfo, arg ast.Expr) bool {
	if fi.pkg.TypesInfo == nil {
		return false
	}
	t := fi.pkg.TypesInfo.TypeOf(arg)
	return t != nil && t.String() == "context.Context"
}

// selectorResolvesTo reports whether a method selector resolves to the given
// fully qualified scalar symbol (pkgpath.(recv).Method or pkgpath.Func).
func (l *loaded) selectorResolvesTo(fi *fileInfo, sel *ast.SelectorExpr, symbol string) bool {
	if fi.pkg.TypesInfo == nil {
		return false
	}
	obj := fi.pkg.TypesInfo.ObjectOf(sel.Sel)
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	return symbolOfFunc(fn) == symbol
}

// symbolOfFunc renders a *types.Func as an operation.Symbol string:
// "importpath.(*Recv).Method", "importpath.(Recv).Method", or "importpath.Func".
func symbolOfFunc(fn *types.Func) string {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return ""
	}
	if recv := sig.Recv(); recv != nil {
		rt := recv.Type()
		ptr := ""
		if p, isPtr := rt.(*types.Pointer); isPtr {
			ptr = "*"
			rt = p.Elem()
		}
		named, ok := rt.(*types.Named)
		if !ok || named.Obj().Pkg() == nil {
			return ""
		}
		return named.Obj().Pkg().Path() + ".(" + ptr + named.Obj().Name() + ")." + fn.Name()
	}
	if fn.Pkg() == nil {
		return ""
	}
	return fn.Pkg().Path() + "." + fn.Name()
}

// batchKeyType resolves the batch method on the receiver type and validates the
// supported ordered/global-error signature func(context, []K) ([]V, error),
// returning a renderable source string for K.
func (l *loaded) batchKeyType(fi *fileInfo, recv ast.Expr, batchName string) (string, bool) {
	if fi.pkg.TypesInfo == nil {
		return "", false
	}
	rt := fi.pkg.TypesInfo.TypeOf(recv)
	if rt == nil {
		return "", false
	}
	obj, _, _ := types.LookupFieldOrMethod(rt, true, fi.pkg.Types, batchName)
	fn, ok := obj.(*types.Func)
	if !ok {
		return "", false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() != 2 || sig.Results().Len() != 2 {
		return "", false
	}
	// Params: (context.Context, []K).
	if sig.Params().At(0).Type().String() != "context.Context" {
		return "", false
	}
	sliceParam, ok := sig.Params().At(1).Type().(*types.Slice)
	if !ok {
		return "", false
	}
	// Results: ([]V, error).
	if _, ok := sig.Results().At(0).Type().(*types.Slice); !ok {
		return "", false
	}
	if sig.Results().At(1).Type().String() != "error" {
		return "", false
	}
	keyType := types.TypeString(sliceParam.Elem(), types.RelativeTo(fi.pkg.Types))
	return keyType, true
}

// matchErrGuard matches `if <err> != nil { return ... }` with no else and a
// single return, returning that return statement.
func matchErrGuard(stmt ast.Stmt, errName string) (*ast.ReturnStmt, bool) {
	ifs, ok := stmt.(*ast.IfStmt)
	if !ok || ifs.Init != nil || ifs.Else != nil {
		return nil, false
	}
	bin, ok := ifs.Cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return nil, false
	}
	lhs, ok := bin.X.(*ast.Ident)
	if !ok || lhs.Name != errName {
		return nil, false
	}
	if rhs, ok := bin.Y.(*ast.Ident); !ok || rhs.Name != "nil" {
		return nil, false
	}
	if len(ifs.Body.List) != 1 {
		return nil, false
	}
	ret, ok := ifs.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		return nil, false
	}
	return ret, true
}

// allocNames allocates collision-free deterministic generated identifiers within
// the enclosing function.
func (l *loaded) allocNames(fn *ast.FuncDecl) genNames {
	existing := map[string]bool{}
	ast.Inspect(fn, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			existing[id.Name] = true
		}
		return true
	})
	alloc := func(base string) string {
		if !existing[base] {
			existing[base] = true
			return base
		}
		for i := 2; ; i++ {
			c := base + strconv.Itoa(i)
			if !existing[c] {
				existing[c] = true
				return c
			}
		}
	}
	return genNames{
		keys:   alloc("bwKeys"),
		values: alloc("bwValues"),
		err:    alloc("bwErr"),
		index:  alloc("bwIndex"),
	}
}
