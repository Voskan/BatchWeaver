// Package multioperationwave demonstrates BatchWeaver's multi-operation wave
// planning. Independent operations are co-scheduled in the same wave (run in
// parallel without sharing a batch); dependent operations form later waves. The
// critical path identifies the operations that most constrain end-to-end
// latency.
package multioperationwave

import "github.com/Voskan/BatchWeaver/internal/adaptive"

// Plan is the demonstration result.
type Plan struct {
	Waves        [][]string
	CriticalPath []string
}

// Demo builds a small operation dependency graph and returns its dispatch waves
// and critical path. It is deterministic.
func Demo() Plan {
	g := adaptive.NewWaveGraph()
	_ = g.AddNode(adaptive.Node{ID: "load_user", Kind: adaptive.NodeOperation, Operation: "users.get", Cost: 3})
	_ = g.AddNode(adaptive.Node{ID: "load_org", Kind: adaptive.NodeOperation, Operation: "orgs.get", Cost: 2})
	_ = g.AddNode(adaptive.Node{ID: "load_perms", Kind: adaptive.NodeOperation, Operation: "perms.get", Cost: 4})
	_ = g.AddNode(adaptive.Node{ID: "render", Kind: adaptive.NodeComputation, Cost: 1})
	_ = g.AddEdge(adaptive.Edge{From: "load_user", To: "load_perms", Kind: adaptive.EdgeData})
	_ = g.AddEdge(adaptive.Edge{From: "load_org", To: "load_perms", Kind: adaptive.EdgeData})
	_ = g.AddEdge(adaptive.Edge{From: "load_perms", To: "render", Kind: adaptive.EdgeData})

	waves, _ := g.Waves()
	plan := Plan{CriticalPath: g.CriticalPath()}
	for _, w := range waves {
		plan.Waves = append(plan.Waves, w.Nodes)
	}
	return plan
}
