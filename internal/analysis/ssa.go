package analysis

import (
	"go/types"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// ssaResult holds the SSA program and derived indexes.
type ssaResult struct {
	prog        *ssa.Program
	funcs       []*ssa.Function
	scalarFuncs map[string]*ssa.Function // canonical scalar symbol -> function
	funcCount   int
}

// buildSSA builds SSA for the loaded packages. It degrades gracefully: packages
// with type errors are skipped by ssautil and reported elsewhere.
func buildSSA(l *loaded) *ssaResult {
	prog, _ := ssautil.AllPackages(l.pkgs, ssa.InstantiateGenerics)
	prog.Build()
	all := ssautil.AllFunctions(prog)
	funcs := make([]*ssa.Function, 0, len(all))
	for fn := range all {
		funcs = append(funcs, fn)
	}
	sort.Slice(funcs, func(i, j int) bool { return funcKey(funcs[i]) < funcKey(funcs[j]) })

	scalar := make(map[string]*ssa.Function)
	for _, fn := range funcs {
		if fn.Object() == nil {
			continue
		}
		tf, ok := fn.Object().(*types.Func)
		if !ok {
			continue
		}
		if sym, ok := symbolFromFunc(tf); ok {
			if _, exists := scalar[sym.String()]; !exists {
				scalar[sym.String()] = fn
			}
		}
	}
	return &ssaResult{prog: prog, funcs: funcs, scalarFuncs: scalar, funcCount: len(funcs)}
}

// funcKey returns a stable canonical key for an SSA function.
func funcKey(fn *ssa.Function) string {
	if fn.Object() != nil {
		if tf, ok := fn.Object().(*types.Func); ok {
			if sym, ok := symbolFromFunc(tf); ok {
				return sym.String()
			}
		}
	}
	return fn.String()
}

// funcDisplay returns a readable function name for reports.
func funcDisplay(fn *ssa.Function) string {
	if fn.Signature != nil && fn.Signature.Recv() != nil {
		recv := fn.Signature.Recv().Type().String()
		if i := strings.LastIndexByte(recv, '/'); i >= 0 {
			recv = recv[i+1:]
		}
		return "(" + recv + ")." + fn.Name()
	}
	return fn.Name()
}

// buildCallGraph builds a conservative CHA call graph and returns normalized,
// deterministic edges limited to the loaded (non-synthetic) functions.
func buildCallGraph(sr *ssaResult, appFuncs map[*ssa.Function]bool) []CallEdge {
	cg := cha.CallGraph(sr.prog)
	seen := make(map[string]struct{})
	var edges []CallEdge
	for _, node := range cg.Nodes {
		if node == nil || node.Func == nil || !appFuncs[node.Func] {
			continue
		}
		for _, e := range node.Out {
			if e.Callee == nil || e.Callee.Func == nil {
				continue
			}
			caller := shortDigest("fn", funcKey(node.Func))
			callee := shortDigest("fn", funcKey(e.Callee.Func))
			dispatch := DispatchDirect
			if e.Site != nil && e.Site.Common().IsInvoke() {
				dispatch = DispatchInterface
			}
			key := caller + callee + dispatch
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			edges = append(edges, CallEdge{Caller: caller, Callee: callee, Dispatch: dispatch, Algorithm: "cha"})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Caller != edges[j].Caller {
			return edges[i].Caller < edges[j].Caller
		}
		if edges[i].Callee != edges[j].Callee {
			return edges[i].Callee < edges[j].Callee
		}
		return edges[i].Dispatch < edges[j].Dispatch
	})
	return edges
}

// appFunctions returns the set of SSA functions defined in application packages,
// used to bound call-graph and call-site reporting.
func appFunctions(sr *ssaResult, appPaths map[string]bool) map[*ssa.Function]bool {
	out := make(map[*ssa.Function]bool)
	for _, fn := range sr.funcs {
		if fn.Pkg != nil && fn.Pkg.Pkg != nil && appPaths[fn.Pkg.Pkg.Path()] {
			out[fn] = true
		}
	}
	return out
}

// callSiteDiscovery finds calls to declared scalar operations and records their
// structural context. It reports direct static calls precisely and interface
// dispatches conservatively (as ambiguous).
func callSiteDiscovery(l *loaded, sr *ssaResult, ops []Operation, appFuncs, asyncFuncs map[*ssa.Function]bool) ([]CallSite, map[string][]string) {
	// Map scalar symbol -> operation ID and method name -> operations (for
	// conservative interface matching).
	scalarToOp := make(map[string]string)
	scalarFnByOp := make(map[string]*ssa.Function)
	for _, op := range ops {
		if op.ScalarSymbol == "" || op.Disabled {
			continue
		}
		scalarToOp[op.ScalarSymbol] = op.ID
		if fn, ok := sr.scalarFuncs[op.ScalarSymbol]; ok {
			scalarFnByOp[op.ID] = fn
		}
	}
	// The set of declared scalar-operation functions, used to detect keys that
	// are loop-carried through an operation result. It must contain only
	// declared operations, not every resolvable function.
	scalarFnSet := make(map[*ssa.Function]bool, len(scalarFnByOp))
	for _, fn := range scalarFnByOp {
		scalarFnSet[fn] = true
	}

	var sites []CallSite
	byOp := make(map[string][]string)

	for _, fn := range sr.funcs {
		if !appFuncs[fn] {
			continue
		}
		loops := naturalLoopDepths(fn)
		async := asyncFuncs[fn]
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				call, ok := instr.(ssa.CallInstruction)
				if !ok {
					continue
				}
				opID, dispatch, targets := matchOperation(call, scalarToOp, sr)
				if opID == "" {
					continue
				}
				_, isGo := instr.(*ssa.Go)
				pos := l.location(fn.Prog.Fset, call.Common().Pos())
				site := CallSite{
					Operation:           opID,
					Location:            pos,
					EnclosingFunction:   funcDisplay(fn),
					EnclosingFunctionID: shortDigest("fn", funcKey(fn)),
					Dispatch:            dispatch,
					Targets:             targets,
					InGoroutine:         isGo || async,
					LoopDepth:           loops[block],
				}
				classifyArgs(&site, call, scalarFnByOp[opID], scalarFnSet)
				site.Structural = classifyStructural(&site)
				site.ID = "bwcall_" + shortDigest("callsite", funcKey(fn), pos, opID, strconv.Itoa(len(sites)))
				sites = append(sites, site)
				byOp[opID] = append(byOp[opID], site.ID)
			}
		}
	}

	sort.Slice(sites, func(i, j int) bool {
		if sites[i].Location != sites[j].Location {
			return sites[i].Location < sites[j].Location
		}
		return sites[i].ID < sites[j].ID
	})
	return sites, byOp
}

// matchOperation resolves a call to a declared scalar operation.
func matchOperation(call ssa.CallInstruction, scalarToOp map[string]string, sr *ssaResult) (opID, dispatch string, targets int) {
	common := call.Common()
	if fn := common.StaticCallee(); fn != nil {
		if fn.Object() != nil {
			if tf, ok := fn.Object().(*types.Func); ok {
				if sym, ok := symbolFromFunc(tf); ok {
					if id, found := scalarToOp[sym.String()]; found {
						return id, DispatchDirect, 1
					}
				}
			}
		}
		return "", "", 0
	}
	// Interface dispatch: conservatively match by method name and identical
	// signature (ignoring receiver) to a declared scalar function.
	if common.IsInvoke() && common.Method != nil {
		mname := common.Method.Name()
		msig, _ := common.Method.Type().(*types.Signature)
		for symStr, id := range scalarToOp {
			fn := sr.scalarFuncs[symStr]
			if fn == nil || fn.Object() == nil {
				continue
			}
			if tf, ok := fn.Object().(*types.Func); ok && tf.Name() == mname {
				if sameParamsResults(msig, fn.Signature) {
					return id, DispatchInterface, 0 // 0 = ambiguous target count
				}
			}
		}
	}
	return "", "", 0
}

// sameParamsResults compares parameter and result types of two signatures,
// ignoring receivers.
func sameParamsResults(a, b *types.Signature) bool {
	if a == nil || b == nil {
		return false
	}
	return types.Identical(types.NewSignatureType(nil, nil, nil, a.Params(), a.Results(), a.Variadic()),
		types.NewSignatureType(nil, nil, nil, b.Params(), b.Results(), b.Variadic()))
}

// classifyArgs records conservative context, key, and receiver classifications.
func classifyArgs(site *CallSite, call ssa.CallInstruction, scalarFn *ssa.Function, scalarFnSet map[*ssa.Function]bool) {
	args := call.Common().Args
	method := scalarFn != nil && scalarFn.Signature != nil && scalarFn.Signature.Recv() != nil
	// For a static method call, Args[0] is the receiver.
	idx := 0
	if method && call.Common().StaticCallee() != nil {
		if len(args) > 0 {
			site.Receiver = describeValue(args[0])
		}
		idx = 1
	}
	if len(args) > idx {
		site.ContextArg = classifyContext(args[idx])
	}
	if len(args) > idx+1 {
		site.KeyArg = describeValue(args[idx+1])
		site.KeyDependency = classifyKeyDependency(args[idx+1], scalarFnSet)
	}
}

// classifyContext classifies a context argument's source.
func classifyContext(v ssa.Value) string {
	switch t := v.(type) {
	case *ssa.Parameter:
		return "parameter " + t.Name()
	case *ssa.Call:
		if callee := t.Common().StaticCallee(); callee != nil && callee.Pkg != nil &&
			callee.Pkg.Pkg != nil && callee.Pkg.Pkg.Path() == "context" {
			return "derived"
		}
		return "call-result"
	default:
		return "unknown"
	}
}

// describeValue renders a conservative, non-source description of an SSA value.
func describeValue(v ssa.Value) string {
	if v == nil {
		return "unknown"
	}
	switch t := v.(type) {
	case *ssa.Parameter:
		return "parameter " + t.Name()
	case *ssa.Const:
		return "constant"
	case *ssa.FieldAddr:
		return "field " + fieldName(t.X.Type(), t.Field)
	case *ssa.Field:
		return "field " + fieldName(t.X.Type(), t.Field)
	default:
		return shortType(v.Type())
	}
}

func fieldName(t types.Type, i int) string {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		if st, ok := named.Underlying().(*types.Struct); ok && i < st.NumFields() {
			return st.Field(i).Name()
		}
	}
	return "?"
}

func shortType(t types.Type) string {
	s := t.String()
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		return s[i+1:]
	}
	return s
}

// classifyStructural derives a structural context label for a single call site.
func classifyStructural(site *CallSite) string {
	switch {
	case site.Dispatch == DispatchInterface:
		return StateAmbiguousTarget
	case site.InGoroutine:
		return StatePotentialFanout
	case site.LoopDepth > 0:
		return StatePotentialLoop
	default:
		return StateDirectIsolated
	}
}
