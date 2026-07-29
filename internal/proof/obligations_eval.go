package proof

import (
	"strings"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/operation"
)

// evidence builds an evidence node with a stable ID.
func (cc *candidateContext) evidence(kind, detail string) Evidence {
	return Evidence{
		ID:       shortID("bwev", cc.cand.ID, kind, detail),
		Kind:     kind,
		Detail:   detail,
		Location: cc.location,
	}
}

func result(id string, status ObligationStatus, summary string, ev ...Evidence) ObligationResult {
	return ObligationResult{ID: id, Status: status, Summary: summary, Evidence: ev}
}

// oblSignatureCompatible maps the analysis compatibility verdict.
func (cc *candidateContext) oblSignatureCompatible() ObligationResult {
	switch cc.op.Compatibility {
	case "valid":
		return result(OblSignatureCompatible, ObligationSatisfied,
			"scalar and batch signatures are compatible",
			cc.evidence("compatibility", "analysis resolved both symbols and verified error results"))
	case "unresolved":
		return result(OblSignatureCompatible, ObligationUnknown,
			"scalar or batch symbol could not be resolved in the loaded program")
	default:
		return result(OblSignatureCompatible, ObligationViolated,
			"scalar and batch signatures are not compatible")
	}
}

func (cc *candidateContext) oblOperationEnabled() ObligationResult {
	if cc.op.Disabled {
		return result(OblOperationEnabled, ObligationDeferred, "operation is disabled")
	}
	return result(OblOperationEnabled, ObligationSatisfied, "operation is enabled for transformation")
}

func (cc *candidateContext) oblResultContract() ObligationResult {
	if !cc.hasSpec {
		return result(OblResultContract, ObligationUnknown,
			"no validated result contract is available for this operation")
	}
	mode := cc.spec.ResultContract().Mode()
	return result(OblResultContract, ObligationSatisfied,
		"result contract ("+mode.String()+") is sufficient for scalar reconstruction",
		cc.evidence("contract", "result mode "+mode.String()+", missing policy "+cc.spec.ResultContract().Missing().String()))
}

func (cc *candidateContext) oblReadCategory() ObligationResult {
	effect, kindName := cc.operationEffect()
	switch effect {
	case operation.EffectRead:
		return result(OblReadCategory, ObligationSatisfied,
			"operation is declared "+kindName,
			cc.evidence("operation-kind", kindName))
	case operation.EffectUnknown:
		return result(OblReadCategory, ObligationUnknown,
			"operation kind is unknown; a read/write classification is required")
	default:
		w := cc.addWitness(OblReadCategory,
			"operation is a "+kindName+"; core static and runtime coalescing do not batch writes or aggregations",
			[]string{"declared kind: " + kindName})
		r := result(OblReadCategory, ObligationViolated,
			"operation is a "+kindName+"; the strategy does not support it")
		r.Witness = w
		return r
	}
}

// operationEffect returns the operation's read/write/aggregate effect and a
// human-readable kind name, from the contract when present or the analysis kind
// string otherwise.
func (cc *candidateContext) operationEffect() (operation.Effect, string) {
	if cc.hasSpec {
		k := cc.spec.Semantics().Kind()
		return k.Effect(), k.String()
	}
	switch cc.op.Kind {
	case "read-only", "freshness-sensitive-read":
		return operation.EffectRead, cc.op.Kind
	case "":
		return operation.EffectUnknown, "unknown"
	case "commutative-aggregation", "ordered-aggregation":
		return operation.EffectAggregate, cc.op.Kind
	default:
		return operation.EffectWrite, cc.op.Kind
	}
}

func (cc *candidateContext) oblFreshnessNeutral() ObligationResult {
	if cc.hasSpec && cc.spec.Semantics().FreshnessDependent() {
		return result(OblFreshnessNeutral, ObligationUnknown,
			"operation is freshness-sensitive; reordering across observations is not proven safe")
	}
	effect, _ := cc.operationEffect()
	if effect != operation.EffectRead {
		return result(OblFreshnessNeutral, ObligationNotApplicable,
			"freshness does not apply to a non-read operation")
	}
	return result(OblFreshnessNeutral, ObligationSatisfied, "operation is not freshness-sensitive")
}

func (cc *candidateContext) oblTargetResolved() ObligationResult {
	ambiguous := false
	for _, s := range cc.sites {
		if s.Targets != 1 || s.Dispatch == analysis.DispatchInterface || s.Dispatch == analysis.DispatchUnknown {
			ambiguous = true
		}
	}
	if ambiguous {
		w := cc.addWitness(OblTargetResolved,
			"call target includes an unresolved interface implementation",
			[]string{"dispatch: " + strings.Join(cc.dispatch, ", ")})
		r := result(OblTargetResolved, ObligationUnknown,
			"call target does not resolve to a single implementation")
		r.Witness = w
		return r
	}
	return result(OblTargetResolved, ObligationSatisfied,
		"call target resolves to a single implementation",
		cc.evidence("dispatch", "direct call with a single static target"))
}

func (cc *candidateContext) oblNoObservableBarrier() ObligationResult {
	if !cc.effectsComplete {
		return result(OblNoObservableBarrier, ObligationUnknown,
			"the enclosing function has unresolved effects; a barrier cannot be excluded")
	}
	if b := cc.barrierEffects(); len(b) > 0 {
		w := cc.addWitness(OblNoObservableBarrier,
			"the enclosing function performs observable effects that may occur between scalar calls",
			[]string{"effects: " + strings.Join(b, ", ")})
		r := result(OblNoObservableBarrier, ObligationUnknown,
			"an observable effect may occur between scalar calls: "+strings.Join(b, ", "))
		r.Witness = w
		return r
	}
	return result(OblNoObservableBarrier, ObligationSatisfied,
		"the enclosing function is observably effect-free around the scalar calls",
		cc.evidence("effects", "no observable effects in the enclosing function"))
}

func (cc *candidateContext) oblEarlyExitReplayable() ObligationResult {
	if !cc.hasSpec {
		return result(OblEarlyExitReplayable, ObligationUnknown,
			"no result contract is available to reconstruct early control flow")
	}
	if cc.spec.ResultContract().ErrorMode() == operation.ErrorPerItem {
		return result(OblEarlyExitReplayable, ObligationSatisfied,
			"per-item errors allow early control flow to be reconstructed in source order")
	}
	return result(OblEarlyExitReplayable, ObligationUnknown,
		"a global error contract does not preserve per-iteration early exit without further proof")
}

func (cc *candidateContext) oblKeyIndependent() ObligationResult {
	if has(cc.keyDeps, analysis.KeyResultDependent) {
		w := cc.addWitness(OblKeyIndependent,
			"the key expression depends on the result of a prior scalar-operation call",
			[]string{"key dependency: " + analysis.KeyResultDependent})
		r := result(OblKeyIndependent, ObligationViolated,
			"a key depends on a prior scalar result and cannot be collected ahead of the operation")
		r.Witness = w
		return r
	}
	if has(cc.keyDeps, analysis.KeyCallDerived) {
		// A call-derived key is independent only if the called function is
		// side-effect-free. That cannot be inferred here; it is accepted only
		// through an explicit, scoped assumption.
		symbol := "key:" + cc.cand.Operation
		if cc.assumptions.has(symbol, FactSideEffectFreeRead) {
			r := result(OblKeyIndependent, ObligationSatisfied,
				"key independence accepted by an explicit assumption")
			r.Assumptions = []string{shortID("BW-A", symbol, FactSideEffectFreeRead)}
			return r
		}
		id := cc.assumptions.request(symbol, FactSideEffectFreeRead,
			"key expression for "+cc.cand.Operation+" is side-effect-free and independent of prior results")
		r := result(OblKeyIndependent, ObligationNeedsAssumption,
			"a key is produced by a call whose independence requires an explicit assumption")
		r.Assumptions = []string{id}
		r.Witness = cc.addWitness(OblKeyIndependent,
			"the key expression calls another function or reads mutable global state",
			[]string{"key dependency: " + analysis.KeyCallDerived})
		return r
	}
	if has(cc.keyDeps, analysis.KeyUnknown) || len(cc.keyDeps) == 0 {
		return result(OblKeyIndependent, ObligationUnknown,
			"the key expression could not be classified")
	}
	return result(OblKeyIndependent, ObligationSatisfied,
		"keys derive only from parameters, constants, and induction values",
		cc.evidence("key", "structural key dependency"))
}

func (cc *candidateContext) oblNoLoopCarried() ObligationResult {
	if cc.cand.State != analysis.StatePotentialLoop {
		return result(OblNoLoopCarried, ObligationNotApplicable, "candidate is not a loop")
	}
	if has(cc.keyDeps, analysis.KeyResultDependent) {
		r := result(OblNoLoopCarried, ObligationViolated,
			"a loop-carried dependency crosses the operation result")
		r.Witness = cc.addWitness(OblNoLoopCarried,
			"a later iteration key depends on an earlier iteration operation result",
			[]string{"key dependency: " + analysis.KeyResultDependent})
		return r
	}
	if has(cc.keyDeps, analysis.KeyCallDerived) || has(cc.keyDeps, analysis.KeyUnknown) {
		return result(OblNoLoopCarried, ObligationUnknown,
			"a loop-carried dependency cannot be excluded for this key")
	}
	return result(OblNoLoopCarried, ObligationSatisfied,
		"no loop-carried dependency crosses the operation result")
}

func (cc *candidateContext) oblReceiverInvariant() ObligationResult {
	if len(cc.receivers) == 0 {
		return result(OblReceiverInvariant, ObligationSatisfied,
			"operation has no receiver to vary")
	}
	if len(cc.receivers) > 1 || has(cc.receivers, "unknown") {
		r := result(OblReceiverInvariant, ObligationUnknown,
			"receiver identity is not proven invariant across the region")
		r.Witness = cc.addWitness(OblReceiverInvariant,
			"the receiver expression is not proven to evaluate to the same instance",
			[]string{"receivers: " + strings.Join(cc.receivers, ", ")})
		return r
	}
	return result(OblReceiverInvariant, ObligationSatisfied,
		"receiver identity is invariant across the region",
		cc.evidence("receiver", cc.receivers[0]))
}

func (cc *candidateContext) oblContextInvariant() ObligationResult {
	if len(cc.contexts) == 0 {
		return result(OblContextInvariant, ObligationUnknown, "no context argument was identified")
	}
	if len(cc.contexts) == 1 && strings.HasPrefix(cc.contexts[0], "parameter ") {
		return result(OblContextInvariant, ObligationSatisfied,
			"context expression is a loop-invariant parameter",
			cc.evidence("context", cc.contexts[0]))
	}
	r := result(OblContextInvariant, ObligationUnknown,
		"context expression is not proven invariant across the region")
	r.Witness = cc.addWitness(OblContextInvariant,
		"the context argument is derived or varies across the region",
		[]string{"contexts: " + strings.Join(cc.contexts, ", ")})
	return r
}

func (cc *candidateContext) oblPartitionStable() ObligationResult {
	if !cc.hasSpec {
		return result(OblPartitionStable, ObligationUnknown,
			"no partition contract is available to verify isolation")
	}
	receiverInvariant := len(cc.receivers) <= 1 && !has(cc.receivers, "unknown")
	for _, dim := range cc.spec.PartitionContract().Required() {
		name := dim.String()
		if name == "receiver" {
			if !receiverInvariant {
				return result(OblPartitionStable, ObligationUnknown,
					"receiver partition is not proven invariant")
			}
			continue
		}
		// Tenant, authorization, region, consistency, transaction, and session
		// partitions are carried through context values or external state that
		// the core engine does not resolve.
		r := result(OblPartitionStable, ObligationUnknown,
			"partition dimension "+name+" cannot be verified from static facts")
		r.Witness = cc.addWitness(OblPartitionStable,
			"partition dimension "+name+" is not resolvable without a contract",
			[]string{"required dimension: " + name})
		return r
	}
	return result(OblPartitionStable, ObligationSatisfied,
		"required partitions are invariant across the region",
		cc.evidence("partition", "scope "+cc.spec.PartitionContract().Scope().String()))
}

func (cc *candidateContext) oblDuplicateMapping() ObligationResult {
	if !cc.hasSpec {
		return result(OblDuplicateMapping, ObligationUnknown,
			"no result contract is available to reconstruct duplicates and missing results")
	}
	return result(OblDuplicateMapping, ObligationSatisfied,
		"duplicate keys and missing results reconstruct scalar outcomes per the result contract",
		cc.evidence("contract", "missing policy "+cc.spec.ResultContract().Missing().String()))
}

func (cc *candidateContext) oblFirstErrorOrder() ObligationResult {
	if !cc.hasSpec {
		return result(OblFirstErrorOrder, ObligationUnknown,
			"no error contract is available to reconstruct first-error order")
	}
	switch cc.spec.ResultContract().ErrorMode() {
	case operation.ErrorPerItem:
		return result(OblFirstErrorOrder, ObligationSatisfied,
			"per-item errors allow source-order first-error reconstruction",
			cc.evidence("contract", "error mode per-item"))
	default:
		return result(OblFirstErrorOrder, ObligationUnknown,
			"a global or mixed error contract does not preserve source-order first-error without further proof")
	}
}

func (cc *candidateContext) oblNoDeferBarrier() ObligationResult {
	if cc.effects["defer"] {
		r := result(OblNoDeferBarrier, ObligationUnknown,
			"a defer registration in the enclosing function may fall within the candidate region")
		r.Witness = cc.addWitness(OblNoDeferBarrier,
			"the enclosing function registers a defer whose timing could move",
			[]string{"effect: defer"})
		return r
	}
	return result(OblNoDeferBarrier, ObligationSatisfied,
		"no defer registration timing would move")
}

func (cc *candidateContext) oblNoRecoverBoundary() ObligationResult {
	if cc.effects["recover"] {
		r := result(OblNoRecoverBoundary, ObligationUnknown,
			"the enclosing function crosses a recover boundary")
		r.Witness = cc.addWitness(OblNoRecoverBoundary,
			"a recover in the enclosing function may cross the candidate region",
			[]string{"effect: recover"})
		return r
	}
	return result(OblNoRecoverBoundary, ObligationSatisfied, "no recover boundary is crossed")
}

func (cc *candidateContext) oblNoNewConcurrency() ObligationResult {
	// Static prefetch and runtime coalescing never introduce new concurrency by
	// construction; the engine only proves batching of sequential or already
	// concurrent calls.
	return result(OblNoNewConcurrency, ObligationSatisfied,
		"the strategy introduces no new concurrency")
}

func (cc *candidateContext) oblFanoutEnvelope() ObligationResult {
	if !cc.inGoroutine {
		return result(OblFanoutEnvelope, ObligationNotApplicable,
			"candidate is not existing fan-out")
	}
	return result(OblFanoutEnvelope, ObligationSatisfied,
		"calls are already concurrent; coalescing preserves the existing concurrency envelope",
		cc.evidence("concurrency", "calls occur inside launched goroutines"))
}

// has reports whether s contains v.
func has(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
