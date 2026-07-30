package transform

// SchemaVersion identifies the transformation IR schema. It is independent from
// the analysis and proof schema versions and is alpha because the model is not
// yet a stable public contract.
const SchemaVersion = "batchweaver.transform/v1alpha1"

// StrategyVersion versions the strategy implementations. It contributes to plan
// identity so regenerated plans invalidate when the generator changes.
const StrategyVersion = "1"

// StrategyID names a transformation strategy.
type StrategyID string

const (
	// StrategyStaticLoopPrefetch is the only strategy implemented in this stage:
	// hoist proven-safe key collection and a single batch call out of a certified
	// read-only slice/array loop, then replay results in source order.
	StrategyStaticLoopPrefetch StrategyID = "static-loop-prefetch"
)

// Phase names a generated transformation phase. Only phases actually required by
// a transformation appear in its IR.
type Phase string

// Transformation phases.
const (
	PhaseBindInvariants    Phase = "bind-invariants"
	PhaseCollectKeys       Phase = "collect-keys"
	PhaseInvokeBatch       Phase = "invoke-batch-provider"
	PhaseValidateGlobal    Phase = "validate-global-result"
	PhaseMapResults        Phase = "map-results"
	PhaseReplayScalarOrder Phase = "replay-scalar-order"
	PhaseExecuteOriginal   Phase = "execute-original-body"
	PhaseFinalize          Phase = "finalize"
)

// EditKind is the kind of a source edit. Only implemented kinds are exposed.
type EditKind string

// Edit kinds.
const (
	EditReplaceRange EditKind = "replace-range"
	EditInsertBefore EditKind = "insert-before"
)

// ValidationState is the outcome of a validation phase.
type ValidationState string

// Validation states.
const (
	ValidationNotRun  ValidationState = "not-run"
	ValidationPassed  ValidationState = "passed"
	ValidationFailed  ValidationState = "failed"
	ValidationSkipped ValidationState = "skipped-with-reason"
)

// MaterializationState is the lifecycle state of a materialization.
type MaterializationState string

// Materialization states.
const (
	MatPlanned        MaterializationState = "planned"
	MatWriting        MaterializationState = "writing"
	MatCommitted      MaterializationState = "committed"
	MatReverted       MaterializationState = "reverted"
	MatRevertConflict MaterializationState = "revert-conflict"
	MatRecoveryReq    MaterializationState = "recovery-required"
)

// GeneratedRole classifies a source-map segment.
type GeneratedRole string

// Generated roles.
const (
	RoleInvariantBinding GeneratedRole = "invariant-binding"
	RoleKeyCollection    GeneratedRole = "key-collection"
	RoleBatchCall        GeneratedRole = "batch-call"
	RoleGlobalErrorCheck GeneratedRole = "global-error-check"
	RoleResultRecon      GeneratedRole = "result-reconstruction"
	RoleScalarReplay     GeneratedRole = "scalar-order-replay"
)

// Plan is the deterministic, versioned transformation plan for a workspace.
type Plan struct {
	SchemaVersion   string             `json:"schema_version"`
	ID              string             `json:"id"`
	Workspace       string             `json:"workspace"`
	Toolchain       string             `json:"toolchain"`
	BuildConfig     BuildConfig        `json:"build_config"`
	AnalysisDigest  string             `json:"analysis_digest"`
	ProofSchema     string             `json:"proof_schema"`
	ContractDigest  string             `json:"contract_digest,omitempty"`
	StrategyVersion string             `json:"strategy_version"`
	Transformations []Transformation   `json:"transformations"`
	Files           []FilePlan         `json:"files"`
	Skipped         []SkippedCandidate `json:"skipped,omitempty"`
	Diagnostics     []Diagnostic       `json:"diagnostics,omitempty"`
	Validation      ValidationSummary  `json:"validation"`
	Digest          string             `json:"digest"`
}

// BuildConfig records the build configuration a plan is valid for.
type BuildConfig struct {
	GOOS   string   `json:"goos,omitempty"`
	GOARCH string   `json:"goarch,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Tests  bool     `json:"tests,omitempty"`
}

// Transformation is one certified, planned rewrite.
type Transformation struct {
	ID               string       `json:"id"`
	CandidateID      string       `json:"candidate_id"`
	CertificateID    string       `json:"certificate_id"`
	Strategy         StrategyID   `json:"strategy"`
	Operation        string       `json:"operation"`
	Source           SourceAnchor `json:"source"`
	Phases           []Phase      `json:"phases"`
	GeneratedSymbols []string     `json:"generated_symbols"`
	Edits            []string     `json:"edit_ids"`
	Assumptions      []string     `json:"assumptions,omitempty"`
	NonGuarantees    []string     `json:"non_guarantees,omitempty"`
	Digest           string       `json:"digest"`
}

// SourceAnchor locates a candidate in a way tolerant of unrelated edits.
type SourceAnchor struct {
	File           string `json:"file"`
	Package        string `json:"package"`
	Function       string `json:"function"`
	StartLine      int    `json:"start_line"`
	StartCol       int    `json:"start_col"`
	EndLine        int    `json:"end_line"`
	EndCol         int    `json:"end_col"`
	StructuralHash string `json:"structural_hash"`
	Resolution     string `json:"resolution"`
}

// Anchor resolution outcomes.
const (
	AnchorExact             = "exact"
	AnchorRelocatedUnique   = "relocated-unambiguous"
	AnchorAmbiguous         = "ambiguous"
	AnchorMissing           = "missing"
	AnchorStructuralChanged = "structurally-changed"
	AnchorDigestMismatch    = "digest-mismatch"
)

// FilePlan describes the transformed content of one file.
type FilePlan struct {
	Path              string `json:"path"`
	OriginalDigest    string `json:"original_digest"`
	TransformedDigest string `json:"transformed_digest"`
	InsertedLines     int    `json:"inserted_lines"`
	RemovedLines      int    `json:"removed_lines"`
	// transformed holds the generated bytes in memory; it is never serialized in
	// the deterministic plan JSON (only its digest is).
	transformed []byte
	original    []byte
}

// SkippedCandidate records a proven candidate not transformed and why.
type SkippedCandidate struct {
	CandidateID string `json:"candidate_id"`
	Operation   string `json:"operation"`
	Reason      string `json:"reason"`
	Detail      string `json:"detail,omitempty"`
}

// Skip reason codes.
const (
	SkipCertificateStale     = "certificate-stale"
	SkipUnsupportedLoopForm  = "unsupported-loop-form"
	SkipBindingUnavailable   = "generated-binding-unavailable"
	SkipAssumptionMissing    = "explicit-assumption-missing"
	SkipOverlappingRegion    = "overlapping-source-region"
	SkipStrategyNotRequested = "strategy-not-requested"
	SkipNotEligible          = "not-proven-eligible"
	SkipAnchorUnresolved     = "source-anchor-unresolved"
)

// Edit is an immutable source edit.
type Edit struct {
	ID             string   `json:"id"`
	File           string   `json:"file"`
	Kind           EditKind `json:"kind"`
	StartOffset    int      `json:"start_offset"`
	EndOffset      int      `json:"end_offset"`
	OriginalDigest string   `json:"original_digest"`
	Replacement    string   `json:"replacement"`
	Transformation string   `json:"transformation"`
	Order          int      `json:"order"`
}

// SourceMapSegment maps generated code back to a candidate and role.
type SourceMapSegment struct {
	ID             string        `json:"id"`
	File           string        `json:"file"`
	GeneratedStart int           `json:"generated_start_line"`
	GeneratedEnd   int           `json:"generated_end_line"`
	Role           GeneratedRole `json:"role"`
	Transformation string        `json:"transformation"`
	Candidate      string        `json:"candidate"`
	Certificate    string        `json:"certificate"`
}

// ValidationSummary records plan-level validation outcomes.
type ValidationSummary struct {
	Parse         ValidationState `json:"parse"`
	TypeCheck     ValidationState `json:"type_check"`
	Preconditions ValidationState `json:"proof_preconditions"`
	Structural    ValidationState `json:"structural_verification"`
	Detail        string          `json:"detail,omitempty"`
}

// Diagnostic is a transformation diagnostic (BW3xxx range documented in
// docs/reference/diagnostics.md).
type Diagnostic struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Location    string `json:"location,omitempty"`
	Candidate   string `json:"candidate,omitempty"`
	Plan        string `json:"plan,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

// Transformed returns the generated bytes for a file plan (in-memory only).
func (f FilePlan) Transformed() []byte { return f.transformed }

// Original returns the original bytes for a file plan (in-memory only).
func (f FilePlan) Original() []byte { return f.original }
