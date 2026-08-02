// Package release implements BatchWeaver's non-publishing release assurance
// pipeline. It deliberately keeps artifact construction and verification local:
// publishing is a separate, maintainer-authorized operation.
package release

import "time"

const (
	ManifestSchema = "batchweaver.release/v1alpha1"
	RuntimeABI     = "batchweaver.runtime/v1alpha1"
)

// Target identifies a supported release build target.
type Target struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

func (t Target) String() string { return t.GOOS + "/" + t.GOARCH }

// DefaultTargets is intentionally narrower than Go's complete cross-compile
// matrix. Every entry is covered by the repository's compile matrix.
var DefaultTargets = []Target{
	{GOOS: "linux", GOARCH: "amd64"},
	{GOOS: "linux", GOARCH: "arm64"},
	{GOOS: "darwin", GOARCH: "amd64"},
	{GOOS: "darwin", GOARCH: "arm64"},
	{GOOS: "windows", GOARCH: "amd64"},
}

type Digest struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type Artifact struct {
	Path            string  `json:"path"`
	Kind            string  `json:"kind"`
	Platform        string  `json:"platform,omitempty"`
	Size            int64   `json:"size"`
	Digest          Digest  `json:"digest"`
	SBOM            string  `json:"sbom,omitempty"`
	Provenance      string  `json:"provenance,omitempty"`
	Signature       *string `json:"signature,omitempty"`
	Reproducibility string  `json:"reproducibility"`
}

type Toolchain struct {
	GoVersion string `json:"go_version"`
	CGO       string `json:"cgo"`
	Trimpath  bool   `json:"trimpath"`
	BuildVCS  bool   `json:"build_vcs"`
}

type References struct {
	Compatibility string `json:"compatibility"`
	Security      string `json:"security"`
	Performance   string `json:"performance"`
	KnownIssues   string `json:"known_issues"`
}

// Manifest is deterministic. GeneratedAt is omitted on purpose; time-varying
// run metadata belongs in the human release report, never in signed content.
type Manifest struct {
	Schema          string     `json:"schema"`
	Version         string     `json:"version"`
	Snapshot        bool       `json:"snapshot"`
	Publication     string     `json:"publication"`
	Commit          string     `json:"commit"`
	SourceTree      Digest     `json:"source_tree"`
	Toolchain       Toolchain  `json:"toolchain"`
	RuntimeABI      string     `json:"runtime_abi"`
	ConfigSchema    int        `json:"config_schema"`
	Artifacts       []Artifact `json:"artifacts"`
	References      References `json:"references"`
	Signing         string     `json:"signing"`
	ProvenanceModel string     `json:"provenance_model"`
}

type CheckStatus string

const (
	StatusPass    CheckStatus = "PASS"
	StatusFail    CheckStatus = "FAIL"
	StatusNotRun  CheckStatus = "NOT-RUN"
	StatusSkipped CheckStatus = "SKIPPED"
)

type CheckResult struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Status      CheckStatus `json:"status"`
	Detail      string      `json:"detail"`
	Remediation string      `json:"remediation,omitempty"`
}

type ReadinessReport struct {
	Schema      string        `json:"schema"`
	Commit      string        `json:"commit"`
	Version     string        `json:"version_recommendation"`
	GeneratedAt time.Time     `json:"generated_at"`
	Checks      []CheckResult `json:"checks"`
	Ready       bool          `json:"ready"`
	Publication string        `json:"publication"`
}

type BuildOptions struct {
	Root       string
	Output     string
	Version    string
	Snapshot   bool
	Targets    []Target
	SourceDate time.Time
}
