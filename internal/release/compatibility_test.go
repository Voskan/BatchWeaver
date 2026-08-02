package release

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

type compatibilityPolicy struct {
	Schema string `json:"schema"`
	AsOf   string `json:"as_of"`
	Policy struct {
		MinimumGo              string   `json:"minimum_go"`
		CurrentGo              string   `json:"current_go"`
		SupportedGoWindow      string   `json:"supported_go_window"`
		Workflow               string   `json:"workflow"`
		RequiredCheck          string   `json:"required_check"`
		HostedEvidenceArtifact string   `json:"hosted_evidence_artifact"`
		ReleaseTargets         []string `json:"release_targets"`
	} `json:"policy"`
	StatusVocabulary []string `json:"status_vocabulary"`
	Rows             []struct {
		Category    string `json:"category"`
		Environment string `json:"environment"`
		Status      string `json:"status"`
		Gate        string `json:"gate"`
		Evidence    string `json:"evidence"`
	} `json:"rows"`
}

func TestCompatibilityPolicy(t *testing.T) {
	root, err := Root(".")
	if err != nil {
		t.Fatal(err)
	}
	data := readCompatibilityFile(t, filepath.Join(root, "release", "compatibility.json"))
	var policy compatibilityPolicy
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		t.Fatal(err)
	}
	if policy.Schema != "batchweaver.compatibility/v1alpha2" || policy.AsOf == "" {
		t.Fatalf("invalid compatibility identity: %q %q", policy.Schema, policy.AsOf)
	}
	if policy.Policy.MinimumGo != "go1.26.0" || policy.Policy.CurrentGo != "go1.26.5" || policy.Policy.SupportedGoWindow != "go1.26.x" {
		t.Fatalf("unexpected Go policy: %+v", policy.Policy)
	}
	if policy.Policy.Workflow != ".github/workflows/compatibility.yml" || policy.Policy.RequiredCheck != "Compatibility policy" || policy.Policy.HostedEvidenceArtifact != "compatibility-run-<commit>" {
		t.Fatalf("hosted evidence contract is incomplete: %+v", policy.Policy)
	}
	allowed := make(map[string]bool, len(policy.StatusVocabulary))
	for _, status := range policy.StatusVocabulary {
		allowed[status] = true
	}
	requiredRows := map[string]bool{
		"Go/go1.26.0": false, "Go/go1.26.5": false,
		"build/CGO_ENABLED=0": false, "build/CGO_ENABLED=1": false,
		"build/module mode": false, "build/go.work mode": false, "build/vendor/offline module resolution": false,
		"gopls/v0.21.1": false, "VS Code/1.85.2 (minimum)": false, "VS Code/1.131.0 (current)": false,
		"adapter/pgx v5.10.0": false, "adapter/go-redis v9.21.0": false,
		"adapter/gqlgen v0.17.94": false, "adapter/grpc-go v1.83.0": false,
	}
	for _, row := range policy.Rows {
		if !allowed[row.Status] || row.Gate == "" || row.Evidence == "" {
			t.Fatalf("invalid compatibility row: %+v", row)
		}
		key := row.Category + "/" + row.Environment
		if _, ok := requiredRows[key]; ok {
			requiredRows[key] = true
		}
	}
	for row, present := range requiredRows {
		if !present {
			t.Errorf("required compatibility row missing: %s", row)
		}
	}

	goModData := readCompatibilityFile(t, filepath.Join(root, "go.mod"))
	parsed, err := modfile.Parse("go.mod", goModData, nil)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Go == nil || parsed.Go.Version != "1.26" || parsed.Toolchain == nil || parsed.Toolchain.Name != policy.Policy.CurrentGo {
		t.Fatalf("go.mod and compatibility Go policy drifted")
	}
	requirements := make(map[string]string)
	for _, requirement := range parsed.Require {
		requirements[requirement.Mod.Path] = requirement.Mod.Version
	}
	for module, version := range map[string]string{
		"github.com/jackc/pgx/v5": "v5.10.0", "github.com/redis/go-redis/v9": "v9.21.0",
		"github.com/99designs/gqlgen": "v0.17.94", "google.golang.org/grpc": "v1.83.0",
	} {
		if requirements[module] != version {
			t.Errorf("%s = %s, compatibility requires %s", module, requirements[module], version)
		}
	}

	var targets []string
	for _, target := range DefaultTargets {
		targets = append(targets, target.String())
	}
	if !slices.Equal(targets, policy.Policy.ReleaseTargets) {
		t.Fatalf("release targets drifted: code=%v policy=%v", targets, policy.Policy.ReleaseTargets)
	}
	workflow := string(readCompatibilityFile(t, filepath.Join(root, policy.Policy.Workflow)))
	for _, required := range []string{"1.26.0", "1.26.5", "module, vendor, go-work, cgo", "v0.21.1", "1.85.2", "1.131.0", "name: Compatibility policy", "-eq 18"} {
		if !strings.Contains(workflow, required) {
			t.Errorf("compatibility workflow does not enforce %q", required)
		}
	}
	documentation := string(readCompatibilityFile(t, filepath.Join(root, "docs", "release", "compatibility.md")))
	for _, required := range []string{policy.Policy.MinimumGo, policy.Policy.CurrentGo, policy.Policy.RequiredCheck, policy.Policy.HostedEvidenceArtifact} {
		if !strings.Contains(documentation, required) {
			t.Errorf("compatibility documentation omits %q", required)
		}
	}
}

func readCompatibilityFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
