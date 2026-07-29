package analysis

import (
	"golang.org/x/tools/go/ssa"
)

// Key dependency classifications. They describe how a call site's key argument
// is computed, which a later proof stage uses to reason about whether the key
// may be evaluated before the operation's scalar calls.
const (
	// KeyStructural means the key derives only from parameters, constants,
	// fields, induction/range values, and pure computations over them. Such a
	// key can be evaluated without observing mutable state or a prior result.
	KeyStructural = "structural"
	// KeyResultDependent means the key transitively depends on the result of a
	// scalar-operation call, i.e. it is loop-carried through the operation
	// result. Such a key must not be evaluated ahead of the operation.
	KeyResultDependent = "result-dependent"
	// KeyCallDerived means the key is produced by calling another function or by
	// reading mutable global state, so its evaluation is observable or its
	// independence cannot be established without a contract.
	KeyCallDerived = "call-derived"
	// KeyUnknown means the key is computed by SSA the classifier does not model
	// conservatively enough to trust.
	KeyUnknown = "unknown"
)

// classifyKeyDependency conservatively classifies how a key value is computed by
// walking the transitive SSA operand closure. It never treats an unmodeled node
// as pure: unmodeled computation yields KeyUnknown, and any function call or
// global read that is not a pure builtin degrades the classification.
func classifyKeyDependency(key ssa.Value, scalarFns map[*ssa.Function]bool) string {
	if key == nil {
		return KeyUnknown
	}
	seen := make(map[ssa.Value]bool)
	var stack []ssa.Value
	push := func(v ssa.Value) {
		if v != nil && !seen[v] {
			seen[v] = true
			stack = append(stack, v)
		}
	}
	push(key)

	var sawScalarResult, sawCall, sawGlobalRead, sawUnknown bool
	addOperands := func(instr ssa.Instruction) {
		var rands []*ssa.Value
		for _, r := range instr.Operands(rands) {
			if r != nil && *r != nil {
				push(*r)
			}
		}
	}

	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		switch t := v.(type) {
		case *ssa.Parameter, *ssa.Const, *ssa.FreeVar, *ssa.Function:
			// Pure leaves.
		case *ssa.Global:
			// The address of a global reached through the key closure implies a
			// read of mutable, externally observable state.
			sawGlobalRead = true
		case *ssa.Call:
			if callee := t.Common().StaticCallee(); callee != nil && scalarFns[callee] {
				sawScalarResult = true
			} else if _, isBuiltin := t.Common().Value.(*ssa.Builtin); isBuiltin {
				// A builtin such as len/cap is pure; fall through to its operands.
			} else {
				sawCall = true
			}
			addOperands(t)
		case *ssa.Phi, *ssa.BinOp, *ssa.UnOp, *ssa.Convert, *ssa.ChangeType,
			*ssa.Field, *ssa.FieldAddr, *ssa.Index, *ssa.IndexAddr, *ssa.Slice,
			*ssa.Extract, *ssa.MakeInterface, *ssa.ChangeInterface, *ssa.Lookup,
			*ssa.TypeAssert:
			addOperands(v.(ssa.Instruction))
		default:
			// Any other computed value (closures, ranges over functions, unsafe,
			// etc.) is not modeled; classify conservatively.
			sawUnknown = true
			if instr, ok := v.(ssa.Instruction); ok {
				addOperands(instr)
			}
		}
	}

	switch {
	case sawScalarResult:
		return KeyResultDependent
	case sawCall || sawGlobalRead:
		return KeyCallDerived
	case sawUnknown:
		return KeyUnknown
	default:
		return KeyStructural
	}
}
