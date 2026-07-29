// Package analysis implements BatchWeaver's static discovery and analysis
// foundation: it loads real Go programs, builds canonical identities, discovers
// operation declarations, constructs SSA and a conservative call graph,
// summarizes observable effects, indexes scalar-operation call sites, and
// produces a deterministic, versioned analysis snapshot.
//
// The analysis never modifies source, never executes analyzed application code,
// and never claims that a discovered call is safe to transform. Unknown facts
// are represented explicitly as unknown, ambiguous, or deferred. Everything here
// is internal; no unstable go/types or SSA value is exposed as a public API.
package analysis

// SchemaVersion identifies the analysis snapshot schema. It is deliberately an
// alpha version because the analysis model is not yet a stable public contract.
const SchemaVersion = "batchweaver.analysis/v1alpha1"

// Package classifications.
const (
	ClassApplication     = "application"
	ClassApplicationTest = "application-test"
	ClassDependency      = "dependency"
	ClassStandardLibrary = "standard-library"
	ClassCommand         = "command"
	ClassUnsupported     = "unsupported"
)

// Dispatch kinds for call sites and edges.
const (
	DispatchDirect    = "direct"
	DispatchInterface = "interface"
	DispatchClosure   = "closure"
	DispatchUnknown   = "unknown"
)

// Candidate states. Discovery never asserts transformation safety.
const (
	StateDirectIsolated     = "direct_isolated"
	StatePotentialLoop      = "potential_loop"
	StatePotentialSiblings  = "potential_siblings"
	StatePotentialFanout    = "potential_fanout"
	StateAmbiguousTarget    = "ambiguous_target"
	StateDisabledOperation  = "disabled_operation"
	StateInvalidDeclaration = "invalid_declaration"
)

// Package is a portable record of a loaded Go package.
type Package struct {
	ID         string   `json:"id"`
	ImportPath string   `json:"import_path"`
	Name       string   `json:"name"`
	Module     string   `json:"module,omitempty"`
	Class      string   `json:"class"`
	Files      []string `json:"files"`
	Generated  bool     `json:"generated,omitempty"`
	HasErrors  bool     `json:"has_errors,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

// DeclarationSource records where a declaration field came from.
type DeclarationSource struct {
	Kind     string `json:"kind"` // typed | configuration | directive
	Location string `json:"location"`
}

// Operation is a discovered, resolved BatchWeaver operation.
type Operation struct {
	ID            string              `json:"id"`
	Sources       []DeclarationSource `json:"sources"`
	ScalarSymbol  string              `json:"scalar_symbol,omitempty"`
	BatchSymbol   string              `json:"batch_symbol,omitempty"`
	Kind          string              `json:"kind,omitempty"`
	Compatibility string              `json:"compatibility"` // valid | unresolved | invalid
	Disabled      bool                `json:"disabled,omitempty"`
	CallSiteIDs   []string            `json:"call_site_ids,omitempty"`
}

// CallSite is a discovered call to a declared scalar operation.
type CallSite struct {
	ID                string `json:"id"`
	Operation         string `json:"operation"`
	Location          string `json:"location"`
	EnclosingFunction string `json:"enclosing_function"`
	Dispatch          string `json:"dispatch"`
	Structural        string `json:"structural"`
	LoopDepth         int    `json:"loop_depth,omitempty"`
	InGoroutine       bool   `json:"in_goroutine,omitempty"`
	Targets           int    `json:"targets"`
	ContextArg        string `json:"context_arg,omitempty"`
	KeyArg            string `json:"key_arg,omitempty"`
	Receiver          string `json:"receiver,omitempty"`
}

// CallEdge is a normalized call graph edge.
type CallEdge struct {
	Caller    string `json:"caller"`
	Callee    string `json:"callee"`
	Dispatch  string `json:"dispatch"`
	Algorithm string `json:"algorithm"`
}

// EffectSummary is a conservative summary of a function's observable effects.
type EffectSummary struct {
	Function string   `json:"function"`
	Effects  []string `json:"effects"`
	Complete bool     `json:"complete"`
}

// Candidate is a discovered potential batching structure. It never asserts
// safety.
type Candidate struct {
	ID                string   `json:"id"`
	Operation         string   `json:"operation"`
	State             string   `json:"state"`
	StructuralContext string   `json:"structural_context"`
	CallSites         []string `json:"call_sites"`
	Evidence          []string `json:"evidence,omitempty"`
}

// Diag is an analysis diagnostic (a portable projection of the diagnostics model).
type Diag struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Location    string `json:"location,omitempty"`
	Fingerprint string `json:"fingerprint"`
	Phase       string `json:"phase,omitempty"`
}

// Snapshot is the immutable, deterministic result of an analysis. Volatile
// fields (Timestamp) are omitted in reproducible mode.
type Snapshot struct {
	SchemaVersion       string          `json:"schema_version"`
	ToolVersion         string          `json:"tool_version"`
	GoVersion           string          `json:"go_version"`
	Timestamp           string          `json:"timestamp,omitempty"`
	Workspace           string          `json:"workspace"`
	ConfigDigest        string          `json:"config_digest,omitempty"`
	BuildDigest         string          `json:"build_digest"`
	PackagesLoaded      int             `json:"packages_loaded"`
	ApplicationPackages int             `json:"application_packages"`
	TestVariants        int             `json:"test_variants"`
	SSAFunctions        int             `json:"ssa_functions"`
	CallGraphEdges      int             `json:"call_graph_edges"`
	Incomplete          bool            `json:"incomplete,omitempty"`
	Packages            []Package       `json:"packages"`
	Operations          []Operation     `json:"operations"`
	CallSites           []CallSite      `json:"call_sites"`
	Edges               []CallEdge      `json:"call_edges"`
	Effects             []EffectSummary `json:"effect_summaries"`
	Candidates          []Candidate     `json:"candidates"`
	Diagnostics         []Diag          `json:"diagnostics"`
}
