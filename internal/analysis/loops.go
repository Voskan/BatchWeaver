package analysis

import "golang.org/x/tools/go/ssa"

// naturalLoopDepths returns the loop-nesting depth of each basic block in fn,
// computed from natural loops over the SSA dominator tree. A block's depth is
// the number of natural loops that contain it.
func naturalLoopDepths(fn *ssa.Function) map[*ssa.BasicBlock]int {
	depths := make(map[*ssa.BasicBlock]int)
	if len(fn.Blocks) == 0 {
		return depths
	}
	for _, b := range fn.Blocks {
		for _, s := range b.Succs {
			// A back edge b -> s exists when the successor s dominates b; s is a
			// loop header.
			if s.Dominates(b) {
				for n := range naturalLoop(b, s) {
					depths[n]++
				}
			}
		}
	}
	return depths
}

// naturalLoop returns the set of blocks in the natural loop of the back edge
// latch -> header.
func naturalLoop(latch, header *ssa.BasicBlock) map[*ssa.BasicBlock]bool {
	body := map[*ssa.BasicBlock]bool{header: true}
	if latch == header {
		return body
	}
	body[latch] = true
	stack := []*ssa.BasicBlock{latch}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, p := range n.Preds {
			if !body[p] {
				body[p] = true
				stack = append(stack, p)
			}
		}
	}
	return body
}

// errgroupPath is the import path of the recognized goroutine-launch helper.
const errgroupPath = "golang.org/x/sync/errgroup"

// computeAsyncFunctions returns the set of closures launched asynchronously,
// either directly by a go statement or by a recognized errgroup.Group.Go call.
func computeAsyncFunctions(funcs []*ssa.Function) map[*ssa.Function]bool {
	async := make(map[*ssa.Function]bool)
	mark := func(v ssa.Value) {
		if mc, ok := v.(*ssa.MakeClosure); ok {
			if fn, ok := mc.Fn.(*ssa.Function); ok {
				async[fn] = true
			}
		}
	}
	for _, fn := range funcs {
		for _, b := range fn.Blocks {
			for _, instr := range b.Instrs {
				switch t := instr.(type) {
				case *ssa.Go:
					mark(t.Common().Value)
				case *ssa.Call:
					if isErrgroupGo(t.Common()) {
						for _, a := range t.Common().Args {
							mark(a)
						}
					}
				}
			}
		}
	}
	return async
}

// isErrgroupGo reports whether a call is errgroup.(*Group).Go.
func isErrgroupGo(common *ssa.CallCommon) bool {
	fn := common.StaticCallee()
	if fn == nil || fn.Name() != "Go" || fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return false
	}
	return fn.Pkg.Pkg.Path() == errgroupPath
}
