package transform

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
)

// pendingEdit is a byte-range replacement within one workspace-relative file.
type pendingEdit struct {
	rel   string
	start int
	end   int
	repl  string
}

// siteLoc is a call-site location parsed from an analysis snapshot.
type siteLoc struct {
	rel  string
	line int
}

// lowerRuntime lowers every call site of a candidate into a runtime bridge call
// and returns the edits plus the bridge to generate. Every call site must match
// the supported method shape recv.Scalar(ctx, key) with a compatible signature;
// otherwise the candidate is conservatively skipped.
func (l *loaded) lowerRuntime(op string, locs []siteLoc, scalarSym string) ([]pendingEdit, *bridgeReq, string) {
	if len(locs) == 0 {
		return nil, nil, SkipAnchorUnresolved
	}
	varName := bridgeVarName(op)
	var br *bridgeReq
	var edits []pendingEdit
	for _, loc := range locs {
		fi, call, _, ok := l.findScalarCall(loc.rel, loc.line, scalarSym)
		if !ok {
			return nil, nil, SkipAnchorUnresolved
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || len(call.Args) != 2 {
			return nil, nil, SkipUnsupportedLoopForm
		}
		if !l.isContextType(fi, call.Args[0]) {
			return nil, nil, SkipUnsupportedLoopForm
		}
		off := func(p token.Pos) int { return l.fset.Position(p).Offset }
		recvSrc := srcOf(fi.src, off, sel.X)
		ctxSrc := srcOf(fi.src, off, call.Args[0])
		keySrc := srcOf(fi.src, off, call.Args[1])
		edits = append(edits, pendingEdit{
			rel:   fi.rel,
			start: off(call.Pos()),
			end:   off(call.End()),
			repl:  fmt.Sprintf("%s.Call(%s, %s, %s)", varName, ctxSrc, recvSrc, keySrc),
		})
		if br == nil {
			recv, key, val, name, imports, valid := l.bridgeTypes(fi, sel)
			if !valid {
				return nil, nil, SkipBindingUnavailable
			}
			br = &bridgeReq{
				pkgDir: path.Dir(fi.rel), pkgName: fi.pkg.Name, op: op, varName: varName,
				recvType: recv, keyType: key, valueType: val, scalarName: name,
				file: path.Join(path.Dir(fi.rel), bridgeFileName(op)), imports: imports,
			}
		}
	}
	return edits, br, ""
}

// bridgeTypes extracts and renders the receiver, key, and value types of a
// scalar method and validates the supported func(context.Context, K) (V, error)
// shape. Types in the call site's own package render unqualified and are not
// imported.
func (l *loaded) bridgeTypes(fi *fileInfo, sel *ast.SelectorExpr) (recv, key, val, name string, imports map[string]string, ok bool) {
	if fi.pkg.TypesInfo == nil {
		return "", "", "", "", nil, false
	}
	obj := fi.pkg.TypesInfo.ObjectOf(sel.Sel)
	fn, isFunc := obj.(*types.Func)
	if !isFunc {
		return "", "", "", "", nil, false
	}
	sig, isSig := fn.Type().(*types.Signature)
	if !isSig || sig.Recv() == nil || sig.Params().Len() != 2 || sig.Results().Len() != 2 {
		return "", "", "", "", nil, false
	}
	if sig.Params().At(0).Type().String() != "context.Context" || sig.Results().At(1).Type().String() != "error" {
		return "", "", "", "", nil, false
	}
	ic := newImportCollector()
	ic.local = fi.pkg.Types
	recv = ic.render(sig.Recv().Type())
	key = ic.render(sig.Params().At(1).Type())
	val = ic.render(sig.Results().At(0).Type())
	return recv, key, val, fn.Name(), ic.pkgs, true
}
