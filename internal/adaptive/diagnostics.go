package adaptive

// Diagnostic codes for the adaptive scheduling and production tuning layer. They
// occupy the BW8xxx range, kept deliberately distinct from analysis (BW3xxx),
// transform (BW34xx-38xx), runtime (BW4xxx), proof (BW5xxx), backend adapter
// (BW6xxx), and network adapter (BW7xxx) codes. The prompt's illustrative
// numbering reused the BW7xxx range that network adapters already own; this
// implementation renumbers into BW8xxx to preserve the repository's rule that
// diagnostic ranges are distinct per stage. The mapping is documented in
// docs/reference/tuning-diagnostics.md.
const (
	// CodeProfileIncompatible reports a workload profile that is incompatible with
	// the current build, ABI, or operation contract.
	CodeProfileIncompatible = "BW8001"
	// CodeProfileStale reports a workload profile that is too old for active use.
	CodeProfileStale = "BW8002"
	// CodeInsufficientEvidence reports too little evidence for active tuning.
	CodeInsufficientEvidence = "BW8003"
	// CodeRecommendationExceedsBound reports a recommendation clamped by a hard
	// bound.
	CodeRecommendationExceedsBound = "BW8004"
	// CodeGuardrailBreached reports an SLO guardrail breach.
	CodeGuardrailBreached = "BW8005"
	// CodePolicyRolledBack reports an automatic rollback.
	CodePolicyRolledBack = "BW8006"

	// CodeUnsupportedCycle reports an unsupported cycle in an operation DAG.
	CodeUnsupportedCycle = "BW8101"
	// CodeRecursiveDepthExceeded reports a recursive depth-limit breach.
	CodeRecursiveDepthExceeded = "BW8102"
	// CodeRecursiveProofStale reports a stale recursive traversal proof.
	CodeRecursiveProofStale = "BW8103"

	// CodeQuotaExceeded reports a tenant quota breach.
	CodeQuotaExceeded = "BW8201"
	// CodeStarvation reports detected starvation.
	CodeStarvation = "BW8202"

	// CodeOverload reports scheduler overload.
	CodeOverload = "BW8301"
	// CodeRequestShed reports a request shed by policy.
	CodeRequestShed = "BW8302"
	// CodeAdmissionRejected reports an admission-control rejection.
	CodeAdmissionRejected = "BW8303"

	// CodeReplayIncomplete reports incomplete replay input.
	CodeReplayIncomplete = "BW8401"
	// CodeCostConfidenceLow reports low cost-model confidence.
	CodeCostConfidenceLow = "BW8402"
)

// DiagnosticMessages maps every adaptive diagnostic code to its stable message.
var DiagnosticMessages = map[string]string{
	CodeProfileIncompatible:        "workload profile is incompatible",
	CodeProfileStale:               "workload profile is stale",
	CodeInsufficientEvidence:       "insufficient evidence for active tuning",
	CodeRecommendationExceedsBound: "adaptive recommendation exceeds hard bound",
	CodeGuardrailBreached:          "SLO guardrail breached",
	CodePolicyRolledBack:           "adaptive policy rolled back",
	CodeUnsupportedCycle:           "operation dependency graph contains unsupported cycle",
	CodeRecursiveDepthExceeded:     "recursive batching depth limit exceeded",
	CodeRecursiveProofStale:        "recursive traversal proof is stale",
	CodeQuotaExceeded:              "tenant quota exceeded",
	CodeStarvation:                 "starvation detected",
	CodeOverload:                   "scheduler overload detected",
	CodeRequestShed:                "request shed by policy",
	CodeAdmissionRejected:          "admission control rejected request",
	CodeReplayIncomplete:           "replay input is incomplete",
	CodeCostConfidenceLow:          "cost model confidence is low",
}

// Diagnostic is a single adaptive-layer diagnostic.
type Diagnostic struct {
	Code      string `json:"code"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Operation string `json:"operation,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// newDiag builds a diagnostic from a code, using the stable message unless a
// detail overrides it.
func newDiag(code, severity, operation, detail string) Diagnostic {
	return Diagnostic{
		Code:      code,
		Severity:  severity,
		Message:   DiagnosticMessages[code],
		Operation: operation,
		Detail:    detail,
	}
}
