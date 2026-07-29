package proof

import (
	"errors"
	"sort"
	"strings"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/operation"
)

var errNilSnapshot = errors.New("proof: nil analysis snapshot")

// staticBarrierEffects are the effect categories that block static movement of a
// scalar call. Panic, defer, recover, and heap-alloc are handled by dedicated
// obligations and are intentionally excluded here.
var staticBarrierEffects = map[string]bool{
	"global-write":    true,
	"channel":         true,
	"synchronization": true,
	"network":         true,
	"process":         true,
	"filesystem":      true,
	"unsafe":          true,
	"reflection":      true,
	"unknown-call":    true,
	"randomness":      true,
	"time":            true,
	"logging":         true,
	"goroutine":       true,
}

// candidateContext holds the normalized facts and working state for proving one
// candidate. It never retains SSA pointers.
type candidateContext struct {
	cand    analysis.Candidate
	op      analysis.Operation
	spec    operation.Spec
	hasSpec bool
	sites   []analysis.CallSite

	contractDigest string
	assumptions    *assumptionSet
	limits         Limits

	// derived facts
	effects         map[string]bool
	effectsComplete bool
	dispatch        []string
	receivers       []string
	contexts        []string
	keyDeps         []string
	inGoroutine     bool
	location        string

	witnesses []Witness
}

// newCandidateContext gathers the facts for a candidate from the snapshot
// indexes.
func newCandidateContext(cand analysis.Candidate, op analysis.Operation, specs map[string]operation.Spec, siteByID map[string]analysis.CallSite, effByFn map[string]analysis.EffectSummary, assumptions *assumptionSet, contractDigest string, limits Limits) *candidateContext {
	cc := &candidateContext{
		cand: cand, op: op, contractDigest: contractDigest,
		assumptions: assumptions, limits: limits,
		effects: map[string]bool{}, effectsComplete: true,
	}
	if specs != nil {
		if sp, ok := specs[cand.Operation]; ok {
			cc.spec, cc.hasSpec = sp, true
		}
	}
	recvSet := map[string]bool{}
	ctxSet := map[string]bool{}
	dispSet := map[string]bool{}
	keySet := map[string]bool{}
	seenFn := map[string]bool{}
	for _, id := range cand.CallSites {
		s, ok := siteByID[id]
		if !ok {
			continue
		}
		cc.sites = append(cc.sites, s)
		if cc.location == "" {
			cc.location = s.Location
		}
		dispSet[s.Dispatch] = true
		if s.Targets != 1 || s.Dispatch == analysis.DispatchInterface || s.Dispatch == analysis.DispatchUnknown {
			dispSet["ambiguous"] = true
		}
		if s.Receiver != "" {
			recvSet[s.Receiver] = true
		}
		if s.ContextArg != "" {
			ctxSet[s.ContextArg] = true
		}
		if s.KeyDependency != "" {
			keySet[s.KeyDependency] = true
		}
		if s.InGoroutine {
			cc.inGoroutine = true
		}
		// Fold in the enclosing function's effect summary once per function.
		if s.EnclosingFunctionID != "" && !seenFn[s.EnclosingFunctionID] {
			seenFn[s.EnclosingFunctionID] = true
			if e, ok := effByFn[s.EnclosingFunctionID]; ok {
				for _, eff := range e.Effects {
					cc.effects[eff] = true
				}
				if !e.Complete {
					cc.effectsComplete = false
				}
			} else {
				// No effect summary for the enclosing function: treat as
				// incomplete rather than assuming absence of effects.
				cc.effectsComplete = false
			}
		}
	}
	cc.dispatch = keys(dispSet)
	cc.receivers = keys(recvSet)
	cc.contexts = keys(ctxSet)
	cc.keyDeps = keys(keySet)
	return cc
}

// prove evaluates all obligations, computes per-strategy eligibility, and builds
// the candidate proof record.
func (cc *candidateContext) prove() CandidateProof {
	facts := cc.facts()
	cd := candidateDigest(facts)

	// Special terminal cases derived directly from declaration state.
	if cc.op.Compatibility == "invalid" {
		return cc.terminal(cd, DecisionProvenIneligible, ReasonInvalidDeclaration,
			"operation declaration is invalid; the scalar and batch signatures are not compatible")
	}
	if cc.op.Disabled {
		return cc.terminal(cd, DecisionDeferred, ReasonDisabledOperation,
			"operation is disabled; no transformation strategy is offered")
	}

	results := cc.evaluateObligations()
	strategies := cc.applicableStrategies()

	var elig []StrategyEligibility
	for _, sid := range strategies {
		spec, _ := Strategy(sid)
		status, blocking := strategyStatus(spec.Required, results)
		elig = append(elig, StrategyEligibility{
			Strategy:            sid,
			Status:              status,
			Reason:              strategyReason(status, blocking),
			BlockingObligations: blocking,
		})
	}
	sort.Slice(elig, func(i, j int) bool {
		return strategyRank(elig[i].Strategy) < strategyRank(elig[j].Strategy)
	})

	decision := aggregateDecision(elig)
	ordered := orderedResults(results)

	assumptionIDs := cc.collectAssumptions(ordered)
	proof := CandidateProof{
		ID:                cc.cand.ID,
		Operation:         cc.cand.Operation,
		Structure:         structureLabel(cc.cand.State),
		Location:          cc.location,
		CandidateDigest:   cd,
		Decision:          decision,
		ReasonCode:        decisionReason(decision, elig),
		AllowedStrategies: elig,
		Obligations:       ordered,
		Assumptions:       assumptionIDs,
		Witnesses:         cc.witnesses,
		Limitations:       cc.limitations(),
		Invalidation: InvalidationSet{
			AnalysisDigest:   cc.contractDigest,
			ContractDigest:   cc.contractDigest,
			AssumptionDigest: cc.assumptions.digest(),
			CandidateDigest:  cd,
			ProofSchema:      SchemaVersion,
			StrategyRegistry: StrategyRegistryVersion,
		},
	}
	proof.ProofID = proofID(cd, ordered, elig, cc.contractDigest, cc.assumptions.digest())
	return proof
}

// terminal builds a proof record for a decision reached before strategy
// evaluation (invalid or disabled declarations).
func (cc *candidateContext) terminal(cd string, decision Decision, reason, summary string) CandidateProof {
	res := []ObligationResult{}
	if decision == DecisionProvenIneligible {
		res = append(res, ObligationResult{
			ID: OblSignatureCompatible, Family: FamilyDeclaration,
			Status: ObligationViolated, Summary: summary,
		})
	} else {
		res = append(res, ObligationResult{
			ID: OblOperationEnabled, Family: FamilyDeclaration,
			Status: ObligationDeferred, Summary: summary,
		})
	}
	cp := CandidateProof{
		ID: cc.cand.ID, Operation: cc.cand.Operation,
		Structure: structureLabel(cc.cand.State), Location: cc.location,
		CandidateDigest: cd, Decision: decision, ReasonCode: reason,
		AllowedStrategies: []StrategyEligibility{}, Obligations: res,
		Invalidation: InvalidationSet{
			AnalysisDigest: cc.contractDigest, ContractDigest: cc.contractDigest,
			AssumptionDigest: cc.assumptions.digest(), CandidateDigest: cd,
			ProofSchema: SchemaVersion, StrategyRegistry: StrategyRegistryVersion,
		},
	}
	cp.ProofID = proofID(cd, res, nil, cc.contractDigest, cc.assumptions.digest())
	return cp
}

// facts projects the candidate context into the digest fact bundle.
func (cc *candidateContext) facts() candidateFacts {
	return candidateFacts{
		id: cc.cand.ID, operation: cc.cand.Operation, structure: cc.cand.State,
		compatibility: cc.op.Compatibility, disabled: cc.op.Disabled, kind: cc.op.Kind,
		contractDigest: cc.contractDigest,
		dispatch:       cc.dispatch, effects: keys(effectSet(cc.effects)),
		effectsComplete: cc.effectsComplete, receivers: cc.receivers, contexts: cc.contexts,
	}
}

// applicableStrategies returns the strategies to evaluate for the candidate's
// structural class.
func (cc *candidateContext) applicableStrategies() []string {
	switch cc.cand.State {
	case analysis.StatePotentialLoop:
		return []string{StrategyStaticLoopPrefetch, StrategyStaticOrderedPrefetch, StrategyRuntimeScopeCoalescing}
	case analysis.StatePotentialSiblings:
		return []string{StrategyStaticSiblingFusion, StrategyRuntimeScopeCoalescing}
	case analysis.StatePotentialFanout:
		return []string{StrategyExistingFanoutCoalescing, StrategyRuntimeScopeCoalescing}
	case analysis.StateAmbiguousTarget, analysis.StateDirectIsolated:
		return []string{StrategyRuntimeScopeCoalescing}
	default:
		return []string{StrategyRuntimeScopeCoalescing}
	}
}

// evaluateObligations runs every obligation evaluator and returns the results by
// ID.
func (cc *candidateContext) evaluateObligations() map[string]ObligationResult {
	out := make(map[string]ObligationResult, len(obligationRegistry))
	add := func(r ObligationResult) {
		if spec, ok := Obligation(r.ID); ok && r.Family == "" {
			r.Family = spec.Family
		}
		out[r.ID] = r
	}
	add(cc.oblSignatureCompatible())
	add(cc.oblOperationEnabled())
	add(cc.oblResultContract())
	add(cc.oblReadCategory())
	add(cc.oblFreshnessNeutral())
	add(cc.oblTargetResolved())
	add(cc.oblNoObservableBarrier())
	add(cc.oblEarlyExitReplayable())
	add(cc.oblKeyIndependent())
	add(cc.oblNoLoopCarried())
	add(cc.oblReceiverInvariant())
	add(cc.oblContextInvariant())
	add(cc.oblPartitionStable())
	add(cc.oblDuplicateMapping())
	add(cc.oblFirstErrorOrder())
	add(cc.oblNoDeferBarrier())
	add(cc.oblNoRecoverBoundary())
	add(cc.oblNoNewConcurrency())
	add(cc.oblFanoutEnvelope())
	return out
}

// orderedResults returns the obligation results in canonical registry order.
func orderedResults(m map[string]ObligationResult) []ObligationResult {
	out := make([]ObligationResult, 0, len(m))
	for _, o := range obligationRegistry {
		if r, ok := m[o.ID]; ok {
			out = append(out, r)
		}
	}
	return out
}

// --- helpers ---

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// effectSet copies an effect membership map.
func effectSet(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		if v {
			out[k] = true
		}
	}
	return out
}

// structureLabel renders a human-readable structural context label.
func structureLabel(state string) string {
	switch state {
	case analysis.StatePotentialLoop:
		return "range loop"
	case analysis.StatePotentialSiblings:
		return "sibling calls"
	case analysis.StatePotentialFanout:
		return "goroutine fan-out"
	case analysis.StateAmbiguousTarget:
		return "ambiguous target"
	case analysis.StateDirectIsolated:
		return "direct call"
	case analysis.StateDisabledOperation:
		return "disabled operation"
	case analysis.StateInvalidDeclaration:
		return "invalid declaration"
	default:
		return state
	}
}

// barrierEffects returns the sorted static-barrier effects present.
func (cc *candidateContext) barrierEffects() []string {
	var out []string
	for e := range cc.effects {
		if staticBarrierEffects[e] {
			out = append(out, e)
		}
	}
	sort.Strings(out)
	return out
}

// addWitness records a witness and returns its ID.
func (cc *candidateContext) addWitness(obl, summary string, steps []string) string {
	id := shortID("bwwit", cc.cand.ID, obl)
	truncated := false
	if len(steps) > cc.limits.MaxWitnessSteps {
		steps = steps[:cc.limits.MaxWitnessSteps]
		truncated = true
	}
	cc.witnesses = append(cc.witnesses, Witness{
		ID: id, Obligation: obl, Summary: summary, Steps: steps,
		Location: cc.location, Truncated: truncated,
	})
	return id
}

// collectAssumptions gathers the assumption IDs referenced by obligations and
// records the built-in race-free assumption when concurrency is involved.
func (cc *candidateContext) collectAssumptions(results []ObligationResult) []string {
	set := map[string]bool{}
	for _, r := range results {
		for _, a := range r.Assumptions {
			set[a] = true
		}
	}
	if cc.inGoroutine {
		set[builtinRaceFree] = true
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// limitations returns the non-guarantees recorded on every certificate.
func (cc *candidateContext) limitations() []string {
	lims := []string{
		"provider performance is not guaranteed",
		"provider side effects not declared in the operation contract are not modeled",
		"behavior of code reached only through unresolved reflection is not modeled",
	}
	if cc.inGoroutine {
		lims = append(lims, "correctness assumes the analyzed program is free of data races")
	}
	if !cc.effectsComplete {
		lims = append(lims, "the enclosing function has unresolved effects; some obligations are unknown")
	}
	sort.Strings(lims)
	return lims
}

// strategyReason renders a short human-readable reason for a strategy status.
func strategyReason(status Decision, blocking []string) string {
	switch status {
	case DecisionProvenEligible:
		return "all required obligations are satisfied"
	case DecisionProvenIneligible:
		return "a required obligation is violated: " + strings.Join(blocking, ", ")
	case DecisionUnknown:
		return "a required obligation is unknown: " + strings.Join(blocking, ", ")
	case DecisionRequiresAssumption:
		return "a required obligation needs an assumption: " + strings.Join(blocking, ", ")
	case DecisionDeferred:
		return "a required obligation is deferred: " + strings.Join(blocking, ", ")
	default:
		return ""
	}
}

// decisionReason derives a reason code for the aggregate decision.
func decisionReason(decision Decision, elig []StrategyEligibility) string {
	if decision == DecisionProvenEligible {
		return ReasonNone
	}
	// Report the reason of the most informative blocking strategy.
	for _, s := range elig {
		if s.Status == decision && len(s.BlockingObligations) > 0 {
			blocking := s.BlockingObligations[0]
			if blocking == OblReadCategory {
				return ReasonWriteCategory
			}
			if blocking == OblTargetResolved {
				return ReasonAmbiguousTarget
			}
			spec, _ := Obligation(blocking)
			switch spec.Family {
			case FamilyDependency:
				return ReasonAmbiguousTarget
			case FamilyOrder, FamilyEffect:
				return ReasonObservableBarrier
			case FamilyResult, FamilyError, FamilyReceiver, FamilyContext, FamilyTransaction:
				return ReasonMissingContract
			}
		}
	}
	return ReasonNone
}
