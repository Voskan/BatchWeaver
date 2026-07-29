package proof

import (
	"context"
	"runtime"
	"sort"
	"time"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/operation"
)

// DefaultMaxCandidates bounds how many candidates the engine proves before it
// reports the remainder as unknown due to a resource limit.
const DefaultMaxCandidates = 5000

// Limits bounds proof-engine work so a pathological input cannot exhaust
// resources. A zero value selects conservative defaults.
type Limits struct {
	MaxCandidates   int
	MaxWitnessSteps int
}

func (l Limits) withDefaults() Limits {
	if l.MaxCandidates <= 0 {
		l.MaxCandidates = DefaultMaxCandidates
	}
	if l.MaxWitnessSteps <= 0 {
		l.MaxWitnessSteps = 32
	}
	return l
}

// Input is the immutable input to a proof run.
type Input struct {
	// Snapshot is the analysis snapshot to prove. Required.
	Snapshot *analysis.Snapshot
	// Specs maps operation ID to its validated contract, when available.
	Specs map[string]operation.Spec
	// ContractDigest binds the report to the operation contracts in force.
	ContractDigest string
	// Assumptions are explicit, scoped assumptions supplied by the caller.
	Assumptions []Assumption
	// Reproducible omits volatile fields for byte-stable output.
	Reproducible bool
	// ToolVersion is recorded in the report.
	ToolVersion string
	// Limits bounds engine work.
	Limits Limits
}

// Prove evaluates every candidate in the snapshot and returns a deterministic
// proof report. It honors context cancellation between candidates. It returns a
// non-nil error only for an unusable input; ordinary proof outcomes are recorded
// in the report.
func Prove(ctx context.Context, in Input) (*Report, error) {
	if in.Snapshot == nil {
		return nil, errNilSnapshot
	}
	limits := in.Limits.withDefaults()
	snap := in.Snapshot

	siteByID := make(map[string]analysis.CallSite, len(snap.CallSites))
	for _, s := range snap.CallSites {
		siteByID[s.ID] = s
	}
	opByID := make(map[string]analysis.Operation, len(snap.Operations))
	for _, o := range snap.Operations {
		opByID[o.ID] = o
	}
	effByFn := make(map[string]analysis.EffectSummary, len(snap.Effects))
	for _, e := range snap.Effects {
		effByFn[e.Function] = e
	}
	assumptions := newAssumptionSet(in.Assumptions)

	report := &Report{
		SchemaVersion:      SchemaVersion,
		ToolVersion:        in.ToolVersion,
		GoVersion:          runtime.Version(),
		Workspace:          snap.Workspace,
		AnalysisSchema:     snap.SchemaVersion,
		AnalysisDigest:     snap.BuildDigest,
		ContractDigest:     in.ContractDigest,
		AssumptionDigest:   assumptions.digest(),
		DeclaredOperations: len(snap.Operations),
		OperationCallSites: len(snap.CallSites),
		Candidates:         len(snap.Candidates),
		DecisionCounts:     map[string]int{},
		StrategyCounts:     map[string]int{},
	}

	var diags []Diag
	for i, cand := range snap.Candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if i >= limits.MaxCandidates {
			cp := limitedProof(cand, in.ContractDigest, assumptions.digest())
			report.CandidateProofs = append(report.CandidateProofs, cp)
			report.DecisionCounts[string(cp.Decision)]++
			continue
		}
		cc := newCandidateContext(cand, opByID[cand.Operation], in.Specs, siteByID, effByFn, assumptions, in.ContractDigest, limits)
		cp := cc.prove()
		report.CandidateProofs = append(report.CandidateProofs, cp)
		report.DecisionCounts[string(cp.Decision)]++
		for _, s := range cp.AllowedStrategies {
			if s.Status == DecisionProvenEligible {
				report.StrategyCounts[s.Strategy]++
			}
		}
		diags = append(diags, cc.diagnostics(cp)...)
	}

	report.Assumptions = assumptions.requestedRefs()
	sortDiags(diags)
	report.Diagnostics = diags
	if !in.Reproducible {
		report.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	return report, nil
}

// limitedProof builds a resource-limited unknown proof for a candidate the
// engine did not evaluate.
func limitedProof(cand analysis.Candidate, contractDigest, assumptionDigest string) CandidateProof {
	facts := candidateFacts{id: cand.ID, operation: cand.Operation, structure: cand.State, contractDigest: contractDigest}
	cd := candidateDigest(facts)
	return CandidateProof{
		ID:              cand.ID,
		Operation:       cand.Operation,
		Structure:       structureLabel(cand.State),
		CandidateDigest: cd,
		Decision:        DecisionUnknown,
		ReasonCode:      "resource_limit",
		ProofID:         proofID(cd, nil, nil, contractDigest, assumptionDigest),
		Limitations:     []string{"candidate exceeded the configured proof budget and was not evaluated"},
		Invalidation: InvalidationSet{
			AnalysisDigest: contractDigest, ContractDigest: contractDigest,
			AssumptionDigest: assumptionDigest, CandidateDigest: cd,
			ProofSchema: SchemaVersion, StrategyRegistry: StrategyRegistryVersion,
		},
	}
}

// sortDiags orders diagnostics deterministically.
func sortDiags(d []Diag) {
	sort.Slice(d, func(i, j int) bool {
		if d[i].Location != d[j].Location {
			return d[i].Location < d[j].Location
		}
		if d[i].Code != d[j].Code {
			return d[i].Code < d[j].Code
		}
		return d[i].Fingerprint < d[j].Fingerprint
	})
}
