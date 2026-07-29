package proof

// Reason codes attached to unknown and deferred outcomes so downstream tools can
// branch without parsing prose.
const (
	ReasonNone               = ""
	ReasonAmbiguousTarget    = "ambiguous_target"
	ReasonIncompleteEffects  = "incomplete_effects"
	ReasonObservableBarrier  = "observable_barrier"
	ReasonMissingContract    = "missing_contract"
	ReasonWriteCategory      = "write_category"
	ReasonInvalidDeclaration = "invalid_declaration"
	ReasonDisabledOperation  = "disabled_operation"
	ReasonNeedsAssumption    = "needs_assumption"
	ReasonRecursiveWave      = "recursive_wave"
	ReasonUnsupportedShape   = "unsupported_shape"
)

// strategyStatus reduces a strategy's required obligations to a single decision.
// The reduction is a fixed precedence over the obligation lattice: any violation
// makes the strategy ineligible; otherwise any unknown makes it unknown;
// otherwise any needed-but-unapplied assumption makes it assumption-required;
// otherwise any deferred obligation defers it; otherwise it is eligible.
//
// It also returns the obligation IDs that blocked eligibility, in canonical
// order, for explainability.
func strategyStatus(required []string, results map[string]ObligationResult) (Decision, []string) {
	var violated, unknown, needs, deferred []string
	for _, id := range required {
		r, ok := results[id]
		if !ok {
			// A required obligation that was not evaluated is treated as unknown;
			// the engine must evaluate every required obligation, so this is a
			// conservative guard rather than an expected path.
			unknown = append(unknown, id)
			continue
		}
		switch r.Status {
		case ObligationViolated:
			violated = append(violated, id)
		case ObligationUnknown:
			unknown = append(unknown, id)
		case ObligationNeedsAssumption:
			needs = append(needs, id)
		case ObligationDeferred:
			deferred = append(deferred, id)
		case ObligationSatisfied, ObligationNotApplicable:
			// contributes nothing
		}
	}
	switch {
	case len(violated) > 0:
		return DecisionProvenIneligible, sortObligations(violated)
	case len(unknown) > 0:
		return DecisionUnknown, sortObligations(unknown)
	case len(needs) > 0:
		return DecisionRequiresAssumption, sortObligations(needs)
	case len(deferred) > 0:
		return DecisionDeferred, sortObligations(deferred)
	default:
		return DecisionProvenEligible, nil
	}
}

// aggregateDecision reduces per-strategy statuses to the candidate decision. The
// precedence favors the most informative positive result: any eligible strategy
// makes the candidate proven eligible. Otherwise the candidate takes the best
// remaining status in the order requires-assumption, unknown, deferred, and
// finally proven-ineligible only when every applicable strategy is ineligible.
func aggregateDecision(strategies []StrategyEligibility) Decision {
	if len(strategies) == 0 {
		return DecisionDeferred
	}
	var anyEligible, anyAssumption, anyUnknown, anyDeferred, anyIneligible bool
	for _, s := range strategies {
		switch s.Status {
		case DecisionProvenEligible:
			anyEligible = true
		case DecisionRequiresAssumption:
			anyAssumption = true
		case DecisionUnknown:
			anyUnknown = true
		case DecisionDeferred:
			anyDeferred = true
		case DecisionProvenIneligible:
			anyIneligible = true
		}
	}
	switch {
	case anyEligible:
		return DecisionProvenEligible
	case anyAssumption:
		return DecisionRequiresAssumption
	case anyUnknown:
		return DecisionUnknown
	case anyDeferred && !anyIneligible:
		return DecisionDeferred
	default:
		return DecisionProvenIneligible
	}
}

// sortObligations orders obligation IDs by canonical registry rank.
func sortObligations(ids []string) []string {
	out := append([]string(nil), ids...)
	// insertion sort keeps the tiny slice deterministic without importing sort
	// for a hot path; obligation lists are short.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && obligationRank(out[j]) < obligationRank(out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
