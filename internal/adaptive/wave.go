package adaptive

import (
	"fmt"
	"sort"
	"strings"
)

// NodeKind classifies a node in an operation dependency graph.
type NodeKind string

const (
	// NodeOperation is a batchable operation execution.
	NodeOperation NodeKind = "operation"
	// NodeComputation is pure local computation between operations.
	NodeComputation NodeKind = "computation"
	// NodeBarrier forces all prior work to complete before later work starts.
	NodeBarrier NodeKind = "barrier"
	// NodeAdapterCompound is a single adapter request fusing multiple operations
	// where the adapter explicitly declares that capability.
	NodeAdapterCompound NodeKind = "adapter-compound"
	// NodeRecursiveFrontier is one breadth-first frontier of a recursive traversal.
	NodeRecursiveFrontier NodeKind = "recursive-frontier"
)

// EdgeKind classifies a dependency edge.
type EdgeKind string

const (
	// EdgeData is a value dependency (consumer needs producer's result).
	EdgeData EdgeKind = "data"
	// EdgeControl is a control dependency.
	EdgeControl EdgeKind = "control"
	// EdgeBarrier is an ordering edge imposed by a barrier.
	EdgeBarrier EdgeKind = "barrier"
	// EdgePartition is a partition/session ordering dependency.
	EdgePartition EdgeKind = "partition"
	// EdgeTransaction is a transaction/session identity dependency.
	EdgeTransaction EdgeKind = "transaction"
	// EdgeErrorOrder preserves caller-visible error ordering.
	EdgeErrorOrder EdgeKind = "error-order"
)

// Node is a vertex in the operation dependency graph.
type Node struct {
	ID          string   `json:"id"`
	Kind        NodeKind `json:"kind"`
	Operation   string   `json:"operation,omitempty"`
	Cost        float64  `json:"cost"`
	Priority    int      `json:"priority"`
	FusionGroup string   `json:"fusion_group,omitempty"`
	Partition   string   `json:"partition,omitempty"`
}

// Edge is a directed dependency from producer to consumer.
type Edge struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Kind EdgeKind `json:"kind"`
}

// WaveGraph is a versioned multi-operation dependency DAG. Dispatch groups
// independent nodes into waves so compatible operations run in parallel without
// sharing a batch. It is not safe for concurrent modification.
type WaveGraph struct {
	Version string `json:"version"`
	nodes   map[string]Node
	order   []string
	edges   []Edge
	adj     map[string][]string
	indeg   map[string]int
}

// NewWaveGraph returns an empty graph.
func NewWaveGraph() *WaveGraph {
	return &WaveGraph{
		Version: WaveSchemaVersion,
		nodes:   map[string]Node{},
		adj:     map[string][]string{},
		indeg:   map[string]int{},
	}
}

// AddNode adds or replaces a node. An empty ID is rejected.
func (g *WaveGraph) AddNode(n Node) error {
	if n.ID == "" {
		return fmt.Errorf("adaptive: wave node requires an ID")
	}
	if _, ok := g.nodes[n.ID]; !ok {
		g.order = append(g.order, n.ID)
		g.indeg[n.ID] = 0
	}
	g.nodes[n.ID] = n
	return nil
}

// AddEdge adds a dependency edge. Both endpoints must already exist.
func (g *WaveGraph) AddEdge(e Edge) error {
	if _, ok := g.nodes[e.From]; !ok {
		return fmt.Errorf("adaptive: wave edge from unknown node %q", e.From)
	}
	if _, ok := g.nodes[e.To]; !ok {
		return fmt.Errorf("adaptive: wave edge to unknown node %q", e.To)
	}
	g.edges = append(g.edges, e)
	g.adj[e.From] = append(g.adj[e.From], e.To)
	g.indeg[e.To]++
	return nil
}

// Nodes returns the nodes in insertion order.
func (g *WaveGraph) Nodes() []Node {
	out := make([]Node, 0, len(g.order))
	for _, id := range g.order {
		out = append(out, g.nodes[id])
	}
	return out
}

// Edges returns the edges in insertion order.
func (g *WaveGraph) Edges() []Edge { return append([]Edge(nil), g.edges...) }

// Validate reports the first unsupported cycle, if any, as a diagnostic. A DAG
// is required: a cycle means the dependency graph cannot be scheduled in waves.
func (g *WaveGraph) Validate() *Diagnostic {
	if _, ok := g.topoLevels(); !ok {
		d := newDiag(CodeUnsupportedCycle, "error", "", "operation dependency graph is not acyclic")
		return &d
	}
	return nil
}

// Wave is one dispatch wave: a set of nodes with no unsatisfied dependencies on
// each other, runnable together.
type Wave struct {
	Level int      `json:"level"`
	Nodes []string `json:"nodes"`
	// FusionGroups maps a fusion group to its member nodes within this wave, so a
	// caller can issue one compound adapter request per group.
	FusionGroups map[string][]string `json:"fusion_groups,omitempty"`
}

// topoLevels computes the longest-path level of each node using Kahn's
// algorithm. It returns (levels, true) for a DAG and (nil, false) if a cycle is
// present.
func (g *WaveGraph) topoLevels() (map[string]int, bool) {
	indeg := make(map[string]int, len(g.indeg))
	for k, v := range g.indeg {
		indeg[k] = v
	}
	level := map[string]int{}
	var queue []string
	for _, id := range g.order {
		if indeg[id] == 0 {
			queue = append(queue, id)
			level[id] = 0
		}
	}
	sort.Strings(queue)
	processed := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		processed++
		succ := append([]string(nil), g.adj[id]...)
		sort.Strings(succ)
		for _, to := range succ {
			if level[id]+1 > level[to] {
				level[to] = level[id] + 1
			}
			indeg[to]--
			if indeg[to] == 0 {
				queue = append(queue, to)
				sort.Strings(queue)
			}
		}
	}
	if processed != len(g.order) {
		return nil, false
	}
	return level, true
}

// Waves returns the dispatch waves in ascending level order. Nodes at the same
// level are co-scheduled: run in parallel but not merged into one batch unless
// they share a fusion group. It returns an error diagnostic if the graph has a
// cycle.
func (g *WaveGraph) Waves() ([]Wave, *Diagnostic) {
	levels, ok := g.topoLevels()
	if !ok {
		d := newDiag(CodeUnsupportedCycle, "error", "", "operation dependency graph is not acyclic")
		return nil, &d
	}
	byLevel := map[int][]string{}
	maxLevel := 0
	for _, id := range g.order {
		lv := levels[id]
		byLevel[lv] = append(byLevel[lv], id)
		if lv > maxLevel {
			maxLevel = lv
		}
	}
	var waves []Wave
	for lv := 0; lv <= maxLevel; lv++ {
		ids := byLevel[lv]
		if len(ids) == 0 {
			continue
		}
		sort.Strings(ids)
		w := Wave{Level: lv, Nodes: ids}
		for _, id := range ids {
			if grp := g.nodes[id].FusionGroup; grp != "" {
				if w.FusionGroups == nil {
					w.FusionGroups = map[string][]string{}
				}
				w.FusionGroups[grp] = append(w.FusionGroups[grp], id)
			}
		}
		waves = append(waves, w)
	}
	return waves, nil
}

// CriticalPath returns the node IDs on a maximum-cost path through the DAG, used
// to prioritize the operations that most constrain end-to-end latency. It
// returns nil for a cyclic graph.
func (g *WaveGraph) CriticalPath() []string {
	levels, ok := g.topoLevels()
	if !ok {
		return nil
	}
	// Process nodes in level order so every predecessor is finalized first.
	orderByLevel := append([]string(nil), g.order...)
	sort.SliceStable(orderByLevel, func(i, j int) bool {
		li, lj := levels[orderByLevel[i]], levels[orderByLevel[j]]
		if li != lj {
			return li < lj
		}
		return orderByLevel[i] < orderByLevel[j]
	})
	// incoming[id] is the max path cost arriving at id (excluding id's own cost);
	// dist[id] adds id's cost. This avoids double-counting a node's cost.
	incoming := map[string]float64{}
	dist := map[string]float64{}
	pred := map[string]string{}
	best := ""
	for _, id := range orderByLevel {
		dist[id] = incoming[id] + g.nodes[id].Cost
		for _, to := range g.adj[id] {
			if dist[id] > incoming[to] {
				incoming[to] = dist[id]
				pred[to] = id
			}
		}
		if best == "" || dist[id] > dist[best] {
			best = id
		}
	}
	if best == "" {
		return nil
	}
	var path []string
	for id := best; ; {
		path = append([]string{id}, path...)
		p, ok := pred[id]
		if !ok {
			break
		}
		id = p
	}
	return path
}

// DOT renders the graph as Graphviz DOT for the wave graph CLI.
func (g *WaveGraph) DOT() string {
	var b strings.Builder
	b.WriteString("digraph waves {\n  rankdir=LR;\n")
	for _, id := range g.order {
		n := g.nodes[id]
		label := id
		if n.Operation != "" {
			label = id + "\\n" + n.Operation
		}
		fmt.Fprintf(&b, "  %q [label=%q, shape=%s];\n", id, label, shapeFor(n.Kind))
	}
	for _, e := range g.edges {
		fmt.Fprintf(&b, "  %q -> %q [label=%q];\n", e.From, e.To, string(e.Kind))
	}
	b.WriteString("}\n")
	return b.String()
}

// shapeFor maps a node kind to a DOT shape.
func shapeFor(k NodeKind) string {
	switch k {
	case NodeBarrier:
		return "diamond"
	case NodeRecursiveFrontier:
		return "box3d"
	case NodeAdapterCompound:
		return "component"
	case NodeComputation:
		return "ellipse"
	default:
		return "box"
	}
}
