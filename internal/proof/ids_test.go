package proof

import (
	"strings"
	"testing"
)

func TestCandidateDigestStableAndSensitive(t *testing.T) {
	t.Parallel()
	base := candidateFacts{
		id: "c", operation: "users.get", structure: "potential_loop",
		compatibility: "valid", kind: "read-only", contractDigest: "d",
		dispatch: []string{"direct"}, effects: []string{}, effectsComplete: true,
		receivers: []string{"field repo"}, contexts: []string{"parameter ctx"},
	}
	d1 := candidateDigest(base)
	// Reordering the normalized slices must not change the digest.
	base2 := base
	base2.dispatch = []string{"direct"}
	if candidateDigest(base2) != d1 {
		t.Error("digest is not order-stable")
	}
	// Changing a semantic fact must change the digest.
	base3 := base
	base3.kind = "non-idempotent-write"
	if candidateDigest(base3) == d1 {
		t.Error("digest did not change when the operation kind changed")
	}
}

func TestProofIDInputsMatter(t *testing.T) {
	t.Parallel()
	obl := []ObligationResult{{ID: OblSignatureCompatible, Status: ObligationSatisfied}}
	str := []StrategyEligibility{{Strategy: StrategyRuntimeScopeCoalescing, Status: DecisionProvenEligible}}
	a := proofID("cd", obl, str, "contract", "assume")
	if !strings.HasPrefix(a, "bwproof_") {
		t.Fatalf("unexpected proof ID %q", a)
	}
	if proofID("cd", obl, str, "contract2", "assume") == a {
		t.Error("contract digest must affect the proof ID")
	}
	obl2 := []ObligationResult{{ID: OblSignatureCompatible, Status: ObligationViolated}}
	if proofID("cd", obl2, str, "contract", "assume") == a {
		t.Error("obligation outcomes must affect the proof ID")
	}
}

func TestShortIDDeterministic(t *testing.T) {
	t.Parallel()
	first := shortID("bwx", "a", "b")
	second := shortID("bwx", "a", "b")
	if first != second {
		t.Error("shortID is not deterministic")
	}
	if first == shortID("bwx", "ab", "") {
		t.Error("shortID must separate parts")
	}
}
