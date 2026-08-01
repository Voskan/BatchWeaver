// Package recursivebatching demonstrates BatchWeaver's recursive breadth-first
// batching. A tree is loaded one batched call per frontier level rather than one
// call per node, preserving breadth-first and source-child order. Only proven
// traversal contracts are batched this way.
package recursivebatching

import (
	"context"

	"github.com/Voskan/BatchWeaver/internal/adaptive"
)

// Result is the demonstration result.
type Result struct {
	FrontierSizes []int
	DepthReached  int
	Nodes         int
	BackendCalls  int
}

// Demo traverses a small tree breadth-first, counting one batched backend call
// per frontier. It is deterministic.
func Demo() Result {
	calls := 0
	loader := func(_ context.Context, keys []int) ([]adaptive.NodeResult[int, int], error) {
		calls++ // one batched call per frontier level
		out := make([]adaptive.NodeResult[int, int], len(keys))
		for i, k := range keys {
			out[i] = adaptive.NodeResult[int, int]{Key: k, Value: k, Found: true}
		}
		return out, nil
	}
	contract := adaptive.RecursiveContract[int, int]{
		Children: func(_ int, v int) []int {
			if v >= 15 {
				return nil
			}
			return []int{v*2 + 1, v*2 + 2}
		},
		Limits:      adaptive.RecursiveLimits{MaxDepth: 8},
		Cycle:       adaptive.CycleSkipSeen,
		ErrorPolicy: adaptive.ErrCollectPerNode,
		ProofValid:  true,
	}
	res, _ := adaptive.Traverse(context.Background(), []int{0}, contract, loader)
	return Result{
		FrontierSizes: res.FrontierSizes,
		DepthReached:  res.DepthReached,
		Nodes:         res.Nodes,
		BackendCalls:  calls,
	}
}
