package proof

import "testing"

// res builds an obligation result with a status.
func res(id string, status ObligationStatus) ObligationResult {
	return ObligationResult{ID: id, Status: status}
}

func TestStrategyStatusPrecedence(t *testing.T) {
	t.Parallel()
	required := []string{OblSignatureCompatible, OblReadCategory, OblTargetResolved}
	cases := []struct {
		name    string
		results map[string]ObligationResult
		want    Decision
	}{
		{
			name: "all satisfied",
			results: map[string]ObligationResult{
				OblSignatureCompatible: res(OblSignatureCompatible, ObligationSatisfied),
				OblReadCategory:        res(OblReadCategory, ObligationSatisfied),
				OblTargetResolved:      res(OblTargetResolved, ObligationSatisfied),
			},
			want: DecisionProvenEligible,
		},
		{
			name: "violation dominates unknown",
			results: map[string]ObligationResult{
				OblSignatureCompatible: res(OblSignatureCompatible, ObligationViolated),
				OblReadCategory:        res(OblReadCategory, ObligationUnknown),
				OblTargetResolved:      res(OblTargetResolved, ObligationSatisfied),
			},
			want: DecisionProvenIneligible,
		},
		{
			name: "unknown dominates needs-assumption",
			results: map[string]ObligationResult{
				OblSignatureCompatible: res(OblSignatureCompatible, ObligationSatisfied),
				OblReadCategory:        res(OblReadCategory, ObligationUnknown),
				OblTargetResolved:      res(OblTargetResolved, ObligationNeedsAssumption),
			},
			want: DecisionUnknown,
		},
		{
			name: "needs-assumption dominates deferred",
			results: map[string]ObligationResult{
				OblSignatureCompatible: res(OblSignatureCompatible, ObligationNeedsAssumption),
				OblReadCategory:        res(OblReadCategory, ObligationDeferred),
				OblTargetResolved:      res(OblTargetResolved, ObligationSatisfied),
			},
			want: DecisionRequiresAssumption,
		},
		{
			name: "not-applicable is inert",
			results: map[string]ObligationResult{
				OblSignatureCompatible: res(OblSignatureCompatible, ObligationSatisfied),
				OblReadCategory:        res(OblReadCategory, ObligationNotApplicable),
				OblTargetResolved:      res(OblTargetResolved, ObligationSatisfied),
			},
			want: DecisionProvenEligible,
		},
		{
			name:    "missing required obligation is unknown",
			results: map[string]ObligationResult{},
			want:    DecisionUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := strategyStatus(required, tc.results)
			if got != tc.want {
				t.Errorf("strategyStatus = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestAggregateDecision(t *testing.T) {
	t.Parallel()
	elig := func(ds ...Decision) []StrategyEligibility {
		var out []StrategyEligibility
		for _, d := range ds {
			out = append(out, StrategyEligibility{Status: d})
		}
		return out
	}
	cases := []struct {
		name string
		in   []StrategyEligibility
		want Decision
	}{
		{"eligible wins", elig(DecisionProvenIneligible, DecisionProvenEligible), DecisionProvenEligible},
		{"assumption over unknown", elig(DecisionUnknown, DecisionRequiresAssumption), DecisionRequiresAssumption},
		{"unknown over deferred", elig(DecisionDeferred, DecisionUnknown), DecisionUnknown},
		{"deferred when no ineligible", elig(DecisionDeferred), DecisionDeferred},
		{"all ineligible", elig(DecisionProvenIneligible, DecisionProvenIneligible), DecisionProvenIneligible},
		{"empty is deferred", nil, DecisionDeferred},
		{"deferred plus ineligible is ineligible", elig(DecisionDeferred, DecisionProvenIneligible), DecisionProvenIneligible},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := aggregateDecision(tc.in); got != tc.want {
				t.Errorf("aggregateDecision = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestSortObligationsCanonical(t *testing.T) {
	t.Parallel()
	in := []string{OblTargetResolved, OblSignatureCompatible, OblReadCategory}
	got := sortObligations(in)
	if got[0] != OblSignatureCompatible {
		t.Errorf("expected declaration obligation first, got %v", got)
	}
	// The input slice must not be mutated.
	if in[0] != OblTargetResolved {
		t.Error("sortObligations mutated its input")
	}
}
