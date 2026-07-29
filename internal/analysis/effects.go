package analysis

import (
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/ssa"
)

// maxEffectIterations bounds the interprocedural fixed point so a pathological
// call graph cannot loop unbounded.
const maxEffectIterations = 32

// funcEffects holds a function's effect set and completeness.
type funcEffects struct {
	effects  map[string]bool
	complete bool
}

// computeEffects computes conservative effect summaries for application
// functions, propagating callee effects to callers through a bounded, monotone
// interprocedural fixed point. Unresolved dynamic calls mark a summary
// incomplete, and unknown never collapses into optimistic absence.
func computeEffects(sr *ssaResult, appFuncs map[*ssa.Function]bool) []EffectSummary {
	state := make(map[*ssa.Function]*funcEffects, len(sr.funcs))
	for _, fn := range sr.funcs {
		state[fn] = directEffects(fn)
	}

	cg := cha.CallGraph(sr.prog)
	for iter := 0; iter < maxEffectIterations; iter++ {
		changed := false
		for _, node := range cg.Nodes {
			caller := node.Func
			cs := state[caller]
			if cs == nil {
				continue
			}
			for _, e := range node.Out {
				callee := e.Callee.Func
				es := state[callee]
				if es == nil {
					if !cs.effects["unknown-call"] {
						cs.effects["unknown-call"] = true
						cs.complete = false
						changed = true
					}
					continue
				}
				for eff := range es.effects {
					if !cs.effects[eff] {
						cs.effects[eff] = true
						changed = true
					}
				}
				if !es.complete && cs.complete {
					cs.complete = false
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}

	var out []EffectSummary
	for fn := range appFuncs {
		fe := state[fn]
		if fe == nil {
			continue
		}
		effs := make([]string, 0, len(fe.effects))
		for e := range fe.effects {
			effs = append(effs, e)
		}
		sort.Strings(effs)
		out = append(out, EffectSummary{
			Function: shortDigest("fn", funcKey(fn)),
			Effects:  effs,
			Complete: fe.complete,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Function < out[j].Function })
	return out
}

// directEffects computes a function's directly observable effects.
func directEffects(fn *ssa.Function) *funcEffects {
	fe := &funcEffects{effects: make(map[string]bool), complete: true}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			switch t := instr.(type) {
			case *ssa.Go:
				fe.effects["goroutine"] = true
			case *ssa.Defer:
				fe.effects["defer"] = true
			case *ssa.Panic:
				fe.effects["panic"] = true
			case *ssa.Send:
				fe.effects["channel"] = true
			case *ssa.Select:
				fe.effects["channel"] = true
			case *ssa.UnOp:
				if t.Op == token.ARROW {
					fe.effects["channel"] = true
				}
			case *ssa.Store:
				if _, ok := t.Addr.(*ssa.Global); ok {
					fe.effects["global-write"] = true
				}
			case *ssa.Alloc:
				if t.Heap {
					fe.effects["heap-alloc"] = true
				}
			case ssa.CallInstruction:
				addCallEffect(fe, t.Common())
			}
		}
	}
	return fe
}

// addCallEffect records the direct effect implied by a call's static callee, or
// marks the summary incomplete for an unresolved dynamic call.
func addCallEffect(fe *funcEffects, common *ssa.CallCommon) {
	callee := common.StaticCallee()
	if callee == nil {
		// A builtin call is not an unresolved target: its semantics are fixed by
		// the language specification. Classify it precisely rather than treating
		// it as an unknown external effect.
		if b, ok := common.Value.(*ssa.Builtin); ok {
			if eff, observable := builtinEffect(b.Name()); observable {
				fe.effects[eff] = true
			}
			return
		}
		if common.IsInvoke() || common.Signature() != nil {
			fe.effects["unknown-call"] = true
			fe.complete = false
		}
		return
	}
	if callee.Pkg == nil || callee.Pkg.Pkg == nil {
		return
	}
	if eff, ok := stdlibEffect(callee.Pkg.Pkg.Path()); ok {
		fe.effects[eff] = true
	}
}

// builtinEffect returns the observable effect category implied by a Go builtin,
// if any. Most builtins operate on local memory only and are not observable
// reordering barriers; a few have specific semantics that are.
func builtinEffect(name string) (effect string, observable bool) {
	switch name {
	case "close":
		// Closing a channel is a communication/synchronization event.
		return "channel", true
	case "panic":
		return "panic", true
	case "recover":
		return "recover", true
	case "print", "println":
		// The debug builtins write to standard error.
		return "logging", true
	case "append", "len", "cap", "make", "new", "copy", "delete", "clear",
		"complex", "real", "imag", "min", "max":
		return "", false
	default:
		// An unrecognized builtin is treated conservatively as benign local
		// behavior; the set of Go builtins is fixed and fully enumerated above.
		return "", false
	}
}

// stdlibEffect returns the reviewed effect category for a standard-library
// package path, if one applies.
func stdlibEffect(path string) (string, bool) {
	switch {
	case path == "os/exec":
		return "process", true
	case path == "os", path == "io/ioutil":
		return "filesystem", true
	case strings.HasPrefix(path, "net"):
		return "network", true
	case path == "time":
		return "time", true
	case path == "math/rand", path == "math/rand/v2", path == "crypto/rand":
		return "randomness", true
	case strings.HasPrefix(path, "sync"):
		return "synchronization", true
	case path == "reflect":
		return "reflection", true
	case path == "unsafe":
		return "unsafe", true
	case path == "log", path == "log/slog":
		return "logging", true
	default:
		return "", false
	}
}
