package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionCampaignPolicy(t *testing.T) {
	root := testRoot(t)
	workflowPath := filepath.Join(root, ".github", "workflows", "production-campaign.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, required := range []string{
		"workflow_dispatch:", "schedule:", `GO_VERSION: "1.26.5"`,
		`NODE_VERSION: "22.22.0"`, "Production campaign policy",
		"campaign-phase.go", "validate-campaign-evidence.go",
		"retention-days: 90", "persist-credentials: false",
		"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1",
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
		"actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e",
		"actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a",
		"actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("production campaign missing %q", required)
		}
	}
	for _, category := range []string{
		"config", "schemas", "proof", "transforms", "sql", "protocols", "lsp",
		"daemon", "profiles", "releases", "migrations", "compiler", "runtime",
		"adapters", "daemon-runtime", "adaptive", "editor", "soak", "leaks",
		"faults", "release",
	} {
		if !strings.Contains(workflow, "category: "+category) && !strings.Contains(workflow, "--category "+category) {
			t.Errorf("production campaign missing category %q", category)
		}
	}
	documentation, err := os.ReadFile(filepath.Join(root, "docs", "release", "production-campaigns.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, claimBoundary := range []string{"multiple successful hosted campaigns", "no long-term stability claim"} {
		if !strings.Contains(string(documentation), claimBoundary) {
			t.Errorf("campaign documentation missing claim boundary %q", claimBoundary)
		}
	}
}
