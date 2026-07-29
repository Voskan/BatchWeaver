package analysis

import "sort"

// buildCandidates groups discovered call sites into candidate structures. It
// never asserts that a candidate is safe to transform; every candidate defers
// semantic safety to a later analysis stage.
func buildCandidates(ops []Operation, sites []CallSite) []Candidate {
	opByID := make(map[string]Operation, len(ops))
	for _, op := range ops {
		opByID[op.ID] = op
	}

	type groupKey struct{ op, fn string }
	groups := make(map[groupKey][]CallSite)
	var order []groupKey
	for _, s := range sites {
		k := groupKey{s.Operation, s.EnclosingFunction}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], s)
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i].op != order[j].op {
			return order[i].op < order[j].op
		}
		return order[i].fn < order[j].fn
	})

	var candidates []Candidate
	for _, k := range order {
		g := groups[k]
		op := opByID[k.op]
		state := groupState(op, g)
		ids := make([]string, 0, len(g))
		for _, s := range g {
			ids = append(ids, s.ID)
		}
		sort.Strings(ids)
		candidates = append(candidates, Candidate{
			ID:                "bwcand_" + shortDigest("candidate", k.op, k.fn, state),
			Operation:         k.op,
			State:             state,
			StructuralContext: k.fn,
			CallSites:         ids,
			Evidence:          []string{"semantic batching safety has not yet been proven"},
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	return candidates
}

// groupState derives a candidate state from an operation and a group of call
// sites in one enclosing function.
func groupState(op Operation, g []CallSite) string {
	if op.Disabled {
		return StateDisabledOperation
	}
	if op.Compatibility == "invalid" {
		return StateInvalidDeclaration
	}
	hasLoop, hasGoroutine, hasInterface := false, false, false
	for _, s := range g {
		if s.LoopDepth > 0 {
			hasLoop = true
		}
		if s.InGoroutine {
			hasGoroutine = true
		}
		if s.Dispatch == DispatchInterface {
			hasInterface = true
		}
	}
	switch {
	case hasInterface:
		return StateAmbiguousTarget
	case hasGoroutine:
		return StatePotentialFanout
	case hasLoop:
		return StatePotentialLoop
	case len(g) > 1:
		return StatePotentialSiblings
	default:
		return StateDirectIsolated
	}
}
