package proof

// SchemaVersion identifies the proof report schema. It is deliberately an alpha
// version, independent from the analysis schema version, because the proof model
// is not yet a stable public contract. Readers must reject unsupported future
// major versions and tolerate compatible minor additions.
const SchemaVersion = "batchweaver.proof/v1alpha1"

// Decision is the closed set of per-candidate proof outcomes. A decision is
// always derived from named obligations, never from a probabilistic score.
type Decision string

const (
	// DecisionProvenEligible means at least one named strategy is proven safe
	// for the candidate under the recorded assumptions.
	DecisionProvenEligible Decision = "proven_eligible"
	// DecisionProvenIneligible means every applicable strategy has a concrete
	// semantic conflict; at least one obligation is violated.
	DecisionProvenIneligible Decision = "proven_ineligible"
	// DecisionRequiresAssumption means eligibility depends on an explicit
	// assumption that has not been supplied.
	DecisionRequiresAssumption Decision = "requires_assumption"
	// DecisionUnknown means analysis precision or resource limits were
	// insufficient to prove success or failure.
	DecisionUnknown Decision = "unknown"
	// DecisionDeferred means the candidate belongs to a proof or adapter class
	// intentionally reserved for a later stage.
	DecisionDeferred Decision = "deferred"
)

// ObligationStatus is the lattice value of a single proof obligation. Obligation
// evaluation uses this lattice rather than booleans so that unknown and
// assumption-required states propagate deterministically.
type ObligationStatus string

const (
	// ObligationSatisfied means the property is proven to hold.
	ObligationSatisfied ObligationStatus = "satisfied"
	// ObligationViolated means the property is proven not to hold.
	ObligationViolated ObligationStatus = "violated"
	// ObligationNeedsAssumption means the property holds only if a named
	// assumption is accepted.
	ObligationNeedsAssumption ObligationStatus = "needs_assumption"
	// ObligationUnknown means the property could not be decided within the
	// supported precision and limits.
	ObligationUnknown ObligationStatus = "unknown"
	// ObligationNotApplicable means the property does not constrain the
	// candidate or strategy under evaluation.
	ObligationNotApplicable ObligationStatus = "not_applicable"
	// ObligationDeferred means deciding the property is reserved for a later
	// stage.
	ObligationDeferred ObligationStatus = "deferred"
)

// Report is the deterministic, versioned result of proving a workspace. All
// slices are sorted by a stable key so the serialized form is byte-identical
// across machines for unchanged inputs. Volatile fields are omitted in
// reproducible mode.
type Report struct {
	SchemaVersion  string `json:"schema_version"`
	ToolVersion    string `json:"tool_version"`
	GoVersion      string `json:"go_version"`
	Timestamp      string `json:"timestamp,omitempty"`
	Workspace      string `json:"workspace"`
	AnalysisSchema string `json:"analysis_schema"`
	// AnalysisDigest binds the report to the exact analysis inputs it proved.
	AnalysisDigest string `json:"analysis_digest"`
	// ContractDigest binds the report to the operation contracts in force.
	ContractDigest string `json:"contract_digest,omitempty"`
	// AssumptionDigest binds the report to the applied assumption set.
	AssumptionDigest string `json:"assumption_digest,omitempty"`

	DeclaredOperations int `json:"declared_operations"`
	OperationCallSites int `json:"operation_call_sites"`
	Candidates         int `json:"candidates"`

	// DecisionCounts is a normalized histogram keyed by decision value.
	DecisionCounts map[string]int `json:"decision_counts"`
	// StrategyCounts is a normalized histogram of eligible strategies.
	StrategyCounts map[string]int `json:"strategy_counts"`

	CandidateProofs []CandidateProof `json:"candidate_proofs"`
	Assumptions     []AssumptionRef  `json:"assumptions,omitempty"`
	Diagnostics     []Diag           `json:"diagnostics,omitempty"`
}

// CandidateProof is the proof record for a single candidate. Ordering of the
// nested slices is normalized (not semantic).
type CandidateProof struct {
	ID        string `json:"id"`
	ProofID   string `json:"proof_id"`
	Operation string `json:"operation"`
	Structure string `json:"structure"`
	Location  string `json:"location,omitempty"`
	// CandidateDigest captures the analyzed facts; it changes when the
	// candidate's structure, targets, effects, or contracts change.
	CandidateDigest   string                `json:"candidate_digest"`
	Decision          Decision              `json:"decision"`
	ReasonCode        string                `json:"reason_code,omitempty"`
	AllowedStrategies []StrategyEligibility `json:"allowed_strategies"`
	Obligations       []ObligationResult    `json:"obligations"`
	Assumptions       []string              `json:"assumptions,omitempty"`
	Witnesses         []Witness             `json:"witnesses,omitempty"`
	Limitations       []string              `json:"limitations,omitempty"`
	// Invalidation lists the digests whose change invalidates this proof.
	Invalidation InvalidationSet `json:"invalidation"`
}

// StrategyEligibility records the outcome for one named strategy.
type StrategyEligibility struct {
	Strategy string   `json:"strategy"`
	Status   Decision `json:"status"`
	Reason   string   `json:"reason,omitempty"`
	// BlockingObligations names the obligations that prevented eligibility.
	BlockingObligations []string `json:"blocking_obligations,omitempty"`
}

// ObligationResult is the evaluation of one proof obligation for a candidate.
type ObligationResult struct {
	ID       string           `json:"id"`
	Family   string           `json:"family"`
	Status   ObligationStatus `json:"status"`
	Summary  string           `json:"summary"`
	Evidence []Evidence       `json:"evidence,omitempty"`
	// Assumptions names the assumption IDs this result depends on.
	Assumptions []string `json:"assumptions,omitempty"`
	// Witness references a witness ID when the status is violated or unknown.
	Witness string `json:"witness,omitempty"`
}

// Evidence is a source-linked analysis fact supporting or refuting an
// obligation. It never contains raw secrets, keys, or pointer identities.
type Evidence struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Detail   string `json:"detail"`
	Location string `json:"location,omitempty"`
}

// Witness is a concrete trace demonstrating failure or uncertainty.
type Witness struct {
	ID         string   `json:"id"`
	Obligation string   `json:"obligation"`
	Summary    string   `json:"summary"`
	Steps      []string `json:"steps,omitempty"`
	Location   string   `json:"location,omitempty"`
	Truncated  bool     `json:"truncated,omitempty"`
}

// InvalidationSet lists the inputs whose change invalidates a certificate.
type InvalidationSet struct {
	AnalysisDigest   string `json:"analysis_digest"`
	ContractDigest   string `json:"contract_digest,omitempty"`
	AssumptionDigest string `json:"assumption_digest,omitempty"`
	CandidateDigest  string `json:"candidate_digest"`
	ProofSchema      string `json:"proof_schema"`
	StrategyRegistry string `json:"strategy_registry"`
}

// AssumptionRef records an assumption surfaced in a report.
type AssumptionRef struct {
	ID               string   `json:"id"`
	Text             string   `json:"text"`
	Origin           string   `json:"origin"`
	Symbol           string   `json:"symbol,omitempty"`
	Facts            []string `json:"facts,omitempty"`
	Digest           string   `json:"digest"`
	RequestedByCount int      `json:"requested_by_count"`
	Applied          bool     `json:"applied"`
}

// Diag is a proof-engine diagnostic (BW5xxx family).
type Diag struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Location    string `json:"location,omitempty"`
	Candidate   string `json:"candidate,omitempty"`
	Strategy    string `json:"strategy,omitempty"`
	Obligation  string `json:"obligation,omitempty"`
	Fingerprint string `json:"fingerprint"`
}
