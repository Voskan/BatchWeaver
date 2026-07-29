package proof

// StrategyRegistryVersion versions the strategy vocabulary. It contributes to
// certificate invalidation so that adding or changing a strategy invalidates
// prior proofs.
const StrategyRegistryVersion = "1"

// Strategy identifiers. Each names a precise future transformation shape. There
// is deliberately no generic "safe" strategy: eligibility is always attached to
// a named strategy.
const (
	StrategyStaticLoopPrefetch       = "static-loop-prefetch"
	StrategyStaticOrderedPrefetch    = "static-ordered-prefetch"
	StrategyStaticSiblingFusion      = "static-sibling-fusion"
	StrategyExistingFanoutCoalescing = "existing-fanout-coalescing"
	StrategyRuntimeScopeCoalescing   = "runtime-scope-coalescing"
	StrategyWaveCandidate            = "wave-candidate"
	StrategyDirectOnly               = "direct-only"
	StrategyAdapterDeferred          = "adapter-deferred"
)

// StrategySpec describes a transformation strategy and the obligations it
// requires.
type StrategySpec struct {
	ID      string
	Title   string
	Summary string
	// IntroducesConcurrency is true when the strategy would run scalar calls
	// concurrently; the core engine never proves these eligible from sequential
	// code.
	IntroducesConcurrency bool
	// Required lists the obligation IDs that must not be violated or unknown for
	// the strategy to be eligible.
	Required []string
}

// staticCore are the obligations shared by all static (non-runtime) strategies
// that hoist or fuse calls at compile time.
var staticCore = []string{
	OblSignatureCompatible,
	OblOperationEnabled,
	OblResultContract,
	OblReadCategory,
	OblFreshnessNeutral,
	OblTargetResolved,
	OblNoObservableBarrier,
	OblKeyIndependent,
	OblReceiverInvariant,
	OblContextInvariant,
	OblPartitionStable,
	OblDuplicateMapping,
	OblNoDeferBarrier,
	OblNoRecoverBoundary,
	OblNoNewConcurrency,
}

var strategyRegistry = []StrategySpec{
	{
		ID:      StrategyStaticLoopPrefetch,
		Title:   "static loop prefetch",
		Summary: "evaluate proven-safe keys, perform one batch call, and reconstruct scalar observations in original loop order",
		Required: append(append([]string{}, staticCore...),
			OblNoLoopCarried, OblEarlyExitReplayable, OblFirstErrorOrder),
	},
	{
		ID:      StrategyStaticOrderedPrefetch,
		Title:   "static ordered prefetch",
		Summary: "like loop prefetch but uses an ordered per-item contract to preserve source-order first-error semantics",
		Required: append(append([]string{}, staticCore...),
			OblNoLoopCarried, OblEarlyExitReplayable, OblFirstErrorOrder),
	},
	{
		ID:      StrategyStaticSiblingFusion,
		Title:   "static sibling-call fusion",
		Summary: "combine straight-line sibling calls while preserving lexical evaluation and observation order",
		Required: append(append([]string{}, staticCore...),
			OblEarlyExitReplayable, OblFirstErrorOrder),
	},
	{
		ID:      StrategyExistingFanoutCoalescing,
		Title:   "existing fan-out coalescing",
		Summary: "coalesce calls that are already concurrent without introducing additional concurrency",
		Required: []string{
			OblSignatureCompatible, OblOperationEnabled, OblResultContract,
			OblReadCategory, OblTargetResolved, OblPartitionStable,
			OblDuplicateMapping, OblFanoutEnvelope,
		},
	},
	{
		ID:      StrategyRuntimeScopeCoalescing,
		Title:   "runtime scope coalescing",
		Summary: "route scalar calls through the runtime without hoisting or creating concurrency",
		Required: []string{
			OblSignatureCompatible, OblOperationEnabled, OblResultContract,
			OblReadCategory, OblTargetResolved, OblPartitionStable, OblDuplicateMapping,
		},
	},
}

var strategyByID = func() map[string]StrategySpec {
	m := make(map[string]StrategySpec, len(strategyRegistry))
	for _, s := range strategyRegistry {
		m[s.ID] = s
	}
	return m
}()

// Strategies returns the registered strategies in canonical order.
func Strategies() []StrategySpec {
	out := make([]StrategySpec, len(strategyRegistry))
	copy(out, strategyRegistry)
	return out
}

// Strategy returns the spec for a strategy ID, if registered.
func Strategy(id string) (StrategySpec, bool) {
	s, ok := strategyByID[id]
	return s, ok
}

// strategyRank returns the canonical index of a strategy ID for sorting.
func strategyRank(id string) int {
	for i, s := range strategyRegistry {
		if s.ID == id {
			return i
		}
	}
	return len(strategyRegistry)
}
