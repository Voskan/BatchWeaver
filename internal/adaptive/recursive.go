package adaptive

import (
	"context"
	"errors"
	"fmt"
)

// CyclePolicy selects how a recursive traversal treats a node it has already
// visited. The policy is explicit; there is no implicit default that silently
// changes semantics.
type CyclePolicy string

const (
	// CycleError fails the traversal when a cycle is detected.
	CycleError CyclePolicy = "error"
	// CycleSkipSeen skips already-visited nodes.
	CycleSkipSeen CyclePolicy = "skip-seen"
	// CycleReturnMarker records a cycle marker and does not re-expand.
	CycleReturnMarker CyclePolicy = "return-cycle-marker"
)

// RecursiveErrorPolicy selects how per-node errors are handled.
type RecursiveErrorPolicy string

const (
	// ErrFailFast returns on the first node error.
	ErrFailFast RecursiveErrorPolicy = "fail-fast"
	// ErrCollectPerNode records each node error and continues.
	ErrCollectPerNode RecursiveErrorPolicy = "collect-per-node"
	// ErrPartialGraph stops expanding on error but returns the partial graph.
	ErrPartialGraph RecursiveErrorPolicy = "partial-graph"
)

// RecursiveLimits bound a recursive traversal. Every limit is hard; exceeding
// one returns a typed limit error rather than continuing.
type RecursiveLimits struct {
	MaxDepth    int
	MaxNodes    int
	MaxEdges    int
	MaxFrontier int
}

func (l RecursiveLimits) withDefaults() RecursiveLimits {
	if l.MaxDepth <= 0 {
		l.MaxDepth = 32
	}
	if l.MaxNodes <= 0 {
		l.MaxNodes = 100000
	}
	if l.MaxEdges <= 0 {
		l.MaxEdges = 1000000
	}
	if l.MaxFrontier <= 0 {
		l.MaxFrontier = 65536
	}
	return l
}

// RecursiveContract declares the semantics of a proven recursive traversal.
// BatchWeaver only batches traversals whose contract is explicit and whose
// semantic proof is valid; ProofValid must be true, and it is the caller's
// responsibility (via the proof engine) to set it only for proven-eligible
// forms.
type RecursiveContract[K comparable, V any] struct {
	// Children extracts the ordered child keys of a resolved node.
	Children func(key K, value V) []K
	// Terminal reports whether a node is terminal (no expansion).
	Terminal func(key K, value V) bool
	// Limits bound the traversal.
	Limits RecursiveLimits
	// Cycle selects cycle handling.
	Cycle CyclePolicy
	// ErrorPolicy selects error handling.
	ErrorPolicy RecursiveErrorPolicy
	// ProofValid must be true; a false value causes a BW8103 stale-proof error.
	ProofValid bool
}

// NodeResult is one resolved node from a batch load.
type NodeResult[K comparable, V any] struct {
	Key   K
	Value V
	Found bool
	Err   error
	Depth int
}

// RecursiveLoader batch-loads a frontier of node keys. It is the batched backend
// call: one invocation per breadth-first frontier.
type RecursiveLoader[K comparable, V any] func(ctx context.Context, keys []K) ([]NodeResult[K, V], error)

// RecursiveResult is the outcome of a breadth-first traversal.
type RecursiveResult[K comparable, V any] struct {
	// Order lists resolved nodes in breadth-first, source-child order.
	Order []NodeResult[K, V]
	// FrontierSizes records the size of each frontier level.
	FrontierSizes []int
	// DepthReached is the deepest level expanded.
	DepthReached int
	// Nodes is the number of distinct nodes visited.
	Nodes int
	// Edges is the number of parent-child edges traversed.
	Edges int
	// CycleMarkers lists keys that closed a cycle under CycleReturnMarker.
	CycleMarkers []K
	// Diagnostics carries any limit or cycle diagnostics.
	Diagnostics []Diagnostic
}

// ErrRecursiveLimit is returned when a hard recursive limit is exceeded.
var ErrRecursiveLimit = errors.New("adaptive: recursive traversal limit exceeded")

// ErrRecursiveCycle is returned under CycleError when a cycle is detected.
var ErrRecursiveCycle = errors.New("adaptive: recursive traversal cycle detected")

// ErrRecursiveProofStale is returned when the contract's proof is not valid.
var ErrRecursiveProofStale = errors.New("adaptive: recursive traversal proof is stale")

// Traverse performs a breadth-first, level-batched recursive traversal. Each
// frontier is loaded with a single batched call, then its children form the next
// frontier. Breadth-first order and source child order are preserved; DFS is
// never substituted. Cancellation and deadlines are honored through ctx.
func Traverse[K comparable, V any](ctx context.Context, roots []K, c RecursiveContract[K, V], loader RecursiveLoader[K, V]) (RecursiveResult[K, V], error) {
	var res RecursiveResult[K, V]
	if !c.ProofValid {
		res.Diagnostics = append(res.Diagnostics, newDiag(CodeRecursiveProofStale, "error", "", "traversal proof invalid or stale"))
		return res, ErrRecursiveProofStale
	}
	if c.Children == nil {
		return res, fmt.Errorf("adaptive: recursive contract requires a Children function")
	}
	limits := c.Limits.withDefaults()

	visited := make(map[K]struct{})
	frontier := dedupKeys(roots, visited)

	for depth := 0; len(frontier) > 0; depth++ {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		if depth > limits.MaxDepth {
			res.Diagnostics = append(res.Diagnostics, newDiag(CodeRecursiveDepthExceeded, "error", "",
				fmt.Sprintf("depth %d exceeds limit %d", depth, limits.MaxDepth)))
			return res, ErrRecursiveLimit
		}
		if len(frontier) > limits.MaxFrontier {
			res.Diagnostics = append(res.Diagnostics, newDiag(CodeRecursiveDepthExceeded, "error", "",
				fmt.Sprintf("frontier size %d exceeds limit %d", len(frontier), limits.MaxFrontier)))
			return res, ErrRecursiveLimit
		}
		res.FrontierSizes = append(res.FrontierSizes, len(frontier))
		res.DepthReached = depth

		loaded, err := loader(ctx, frontier)
		if err != nil {
			return res, err
		}
		byKey := indexResults(loaded)

		var next []K
		nextSeen := make(map[K]struct{})
		for _, key := range frontier {
			nr, ok := byKey[key]
			if !ok {
				nr = NodeResult[K, V]{Key: key, Found: false}
			}
			nr.Depth = depth
			res.Order = append(res.Order, nr)
			res.Nodes++
			if res.Nodes > limits.MaxNodes {
				res.Diagnostics = append(res.Diagnostics, newDiag(CodeRecursiveDepthExceeded, "error", "",
					fmt.Sprintf("node count exceeds limit %d", limits.MaxNodes)))
				return res, ErrRecursiveLimit
			}
			if nr.Err != nil {
				switch c.ErrorPolicy {
				case ErrFailFast:
					return res, nr.Err
				case ErrPartialGraph:
					continue // do not expand this node
				default: // collect-per-node
					continue
				}
			}
			if !nr.Found {
				continue
			}
			if c.Terminal != nil && c.Terminal(key, nr.Value) {
				continue
			}
			for _, child := range c.Children(key, nr.Value) {
				res.Edges++
				if res.Edges > limits.MaxEdges {
					res.Diagnostics = append(res.Diagnostics, newDiag(CodeRecursiveDepthExceeded, "error", "",
						fmt.Sprintf("edge count exceeds limit %d", limits.MaxEdges)))
					return res, ErrRecursiveLimit
				}
				if _, seen := visited[child]; seen {
					switch c.Cycle {
					case CycleError:
						res.Diagnostics = append(res.Diagnostics, newDiag(CodeUnsupportedCycle, "error", "", "cycle detected in recursive traversal"))
						return res, ErrRecursiveCycle
					case CycleReturnMarker:
						res.CycleMarkers = append(res.CycleMarkers, child)
						continue
					default: // skip-seen
						continue
					}
				}
				if _, dup := nextSeen[child]; dup {
					continue // stable duplicate handling within a frontier
				}
				nextSeen[child] = struct{}{}
				next = append(next, child)
			}
		}
		for k := range nextSeen {
			visited[k] = struct{}{}
		}
		frontier = next
	}
	return res, nil
}

// dedupKeys returns the unique keys in source order, marking them visited.
func dedupKeys[K comparable](keys []K, visited map[K]struct{}) []K {
	var out []K
	for _, k := range keys {
		if _, ok := visited[k]; ok {
			continue
		}
		visited[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

// indexResults maps loaded node results by key.
func indexResults[K comparable, V any](rs []NodeResult[K, V]) map[K]NodeResult[K, V] {
	m := make(map[K]NodeResult[K, V], len(rs))
	for _, r := range rs {
		m[r.Key] = r
	}
	return m
}
