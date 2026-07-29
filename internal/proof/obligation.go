package proof

// Obligation families group related proof obligations. Families are stable and
// appear in reports and diagnostics.
const (
	FamilyDeclaration = "declaration"
	FamilyOrder       = "order"
	FamilyDependency  = "dependency"
	FamilyEffect      = "effect"
	FamilyReceiver    = "receiver"
	FamilyKey         = "key"
	FamilyContext     = "context"
	FamilyResult      = "result"
	FamilyError       = "error"
	FamilyPanic       = "panic"
	FamilyTransaction = "transaction"
	FamilyConcurrency = "concurrency"
)

// Obligation identifiers. Each ID is stable and is never reused for a different
// meaning. New obligations receive new IDs.
const (
	OblSignatureCompatible = "BW-PROOF-DECL-001"
	OblOperationEnabled    = "BW-PROOF-DECL-002"
	OblResultContract      = "BW-PROOF-DECL-003"

	OblReadCategory     = "BW-PROOF-OP-001"
	OblFreshnessNeutral = "BW-PROOF-OP-002"

	OblTargetResolved = "BW-PROOF-TARGET-001"

	OblNoObservableBarrier = "BW-PROOF-ORDER-001"
	OblEarlyExitReplayable = "BW-PROOF-ORDER-002"

	OblKeyIndependent = "BW-PROOF-DEP-001"
	OblNoLoopCarried  = "BW-PROOF-DEP-002"

	OblReceiverInvariant = "BW-PROOF-RECV-001"
	OblContextInvariant  = "BW-PROOF-CTX-001"
	OblPartitionStable   = "BW-PROOF-PART-001"

	OblDuplicateMapping = "BW-PROOF-RESULT-001"
	OblFirstErrorOrder  = "BW-PROOF-ERROR-001"

	OblNoDeferBarrier    = "BW-PROOF-PANIC-001"
	OblNoRecoverBoundary = "BW-PROOF-PANIC-002"

	OblNoNewConcurrency = "BW-PROOF-CONC-001"
	OblFanoutEnvelope   = "BW-PROOF-CONC-002"
)

// ObligationSpec describes a registered obligation.
type ObligationSpec struct {
	ID     string
	Family string
	Title  string
}

// obligationRegistry is the closed, ordered registry of obligations. The order
// is the canonical evaluation and reporting order.
var obligationRegistry = []ObligationSpec{
	{OblSignatureCompatible, FamilyDeclaration, "scalar and batch signatures are compatible"},
	{OblOperationEnabled, FamilyDeclaration, "operation is enabled for transformation"},
	{OblResultContract, FamilyDeclaration, "result contract is sufficient for scalar reconstruction"},
	{OblReadCategory, FamilyEffect, "operation category permits the strategy"},
	{OblFreshnessNeutral, FamilyEffect, "operation is not freshness-sensitive in a way that blocks reordering"},
	{OblTargetResolved, FamilyDependency, "call target resolves to a single implementation"},
	{OblNoObservableBarrier, FamilyOrder, "no observable barrier occurs between scalar calls"},
	{OblEarlyExitReplayable, FamilyOrder, "early control flow can be reconstructed"},
	{OblKeyIndependent, FamilyDependency, "no key depends on a prior scalar result"},
	{OblNoLoopCarried, FamilyDependency, "no loop-carried dependency crosses the operation result"},
	{OblReceiverInvariant, FamilyReceiver, "receiver identity is invariant across the region"},
	{OblContextInvariant, FamilyContext, "context expression is invariant across the region"},
	{OblPartitionStable, FamilyTransaction, "receiver, tenant, and transaction partitions are invariant"},
	{OblDuplicateMapping, FamilyResult, "duplicate keys and missing results reconstruct scalar outcomes"},
	{OblFirstErrorOrder, FamilyError, "source-order first error can be reconstructed"},
	{OblNoDeferBarrier, FamilyPanic, "no defer registration timing would move"},
	{OblNoRecoverBoundary, FamilyPanic, "no recover boundary is crossed"},
	{OblNoNewConcurrency, FamilyConcurrency, "strategy introduces no unsupported concurrency"},
	{OblFanoutEnvelope, FamilyConcurrency, "existing fan-out concurrency envelope is preserved"},
}

var obligationByID = func() map[string]ObligationSpec {
	m := make(map[string]ObligationSpec, len(obligationRegistry))
	for _, o := range obligationRegistry {
		m[o.ID] = o
	}
	return m
}()

// Obligations returns the registered obligations in canonical order.
func Obligations() []ObligationSpec {
	out := make([]ObligationSpec, len(obligationRegistry))
	copy(out, obligationRegistry)
	return out
}

// Obligation returns the spec for an obligation ID, if registered.
func Obligation(id string) (ObligationSpec, bool) {
	o, ok := obligationByID[id]
	return o, ok
}

// obligationRank returns the canonical index of an obligation ID for sorting.
func obligationRank(id string) int {
	for i, o := range obligationRegistry {
		if o.ID == id {
			return i
		}
	}
	return len(obligationRegistry)
}
