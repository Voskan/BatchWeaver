package transform

import "github.com/Voskan/BatchWeaver/internal/proof"

// Certificate is a typed, validated view of a proof certificate. The
// transformation strategy consumes this rather than raw proof JSON.
type Certificate struct {
	CandidateID     string
	ProofID         string
	Operation       string
	Location        string
	Decision        proof.Decision
	Strategies      map[string]proof.Decision // strategy -> per-strategy status
	Assumptions     []string
	NonGuarantees   []string
	CandidateDigest string
}

// newCertificate projects a proof candidate proof into a typed certificate.
func newCertificate(cp proof.CandidateProof) Certificate {
	strategies := make(map[string]proof.Decision, len(cp.AllowedStrategies))
	for _, s := range cp.AllowedStrategies {
		strategies[s.Strategy] = s.Status
	}
	return Certificate{
		CandidateID:     cp.ID,
		ProofID:         cp.ProofID,
		Operation:       cp.Operation,
		Location:        cp.Location,
		Decision:        cp.Decision,
		Strategies:      strategies,
		Assumptions:     cp.Assumptions,
		NonGuarantees:   cp.Limitations,
		CandidateDigest: cp.CandidateDigest,
	}
}

// eligibleForProof reports whether the certificate proves the candidate eligible
// for a proof-engine strategy identifier. It enforces the core safety rule: a
// candidate is transformable only when the overall decision is proven eligible
// and the exact strategy is itself proven eligible.
func (c Certificate) eligibleForProof(proofStrategy string) bool {
	return c.Decision == proof.DecisionProvenEligible && c.Strategies[proofStrategy] == proof.DecisionProvenEligible
}
