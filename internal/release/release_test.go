package release

import (
	"encoding/json"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestPublicAPIBaseline(t *testing.T) {
	root := testRoot(t)
	got := publicAPI(t, root)
	path := filepath.Join(root, "internal", "release", "testdata", "public-api.txt")
	if os.Getenv("UPDATE_API_BASELINE") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read API baseline: %v (run UPDATE_API_BASELINE=1 go test ./internal/release -run TestPublicAPIBaseline)", err)
	}
	if got != string(want) {
		t.Fatalf("public API differs from internal/release/testdata/public-api.txt; review compatibility and regenerate intentionally")
	}
}

func publicAPI(t *testing.T, root string) string {
	t.Helper()
	patterns := []string{".", "./bridge", "./config", "./diagnostics", "./operation", "./runtime"}
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedTypes, Dir: root}, patterns...)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	qualifier := func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	}
	for _, pkg := range pkgs {
		if len(pkg.Errors) != 0 {
			t.Fatalf("load %s: %v", pkg.PkgPath, pkg.Errors)
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			if !token.IsExported(name) {
				continue
			}
			obj := scope.Lookup(name)
			lines = append(lines, pkg.PkgPath+": "+types.ObjectString(obj, qualifier))
			if named, ok := obj.Type().(*types.Named); ok {
				for i := 0; i < named.NumMethods(); i++ {
					method := named.Method(i)
					if method.Exported() {
						lines = append(lines, pkg.PkgPath+": "+types.ObjectString(method, qualifier))
					}
				}
			}
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

func TestSchemaCompatibility(t *testing.T) {
	manifest := Manifest{Schema: ManifestSchema, Version: "0.1.0-beta.1", Snapshot: true, Publication: "disabled", Commit: strings.Repeat("a", 40), Artifacts: []Artifact{{Path: "x", Kind: "test", Size: 1, Digest: Digest{Algorithm: "sha256", Value: strings.Repeat("0", 64)}}}}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(path); err != nil {
		t.Fatalf("current schema rejected: %v", err)
	}
	manifest.Schema = "batchweaver.release/v0"
	data, _ = json.Marshal(manifest)
	_ = os.WriteFile(path, data, 0o644)
	if _, err := LoadManifest(path); err == nil || !strings.Contains(err.Error(), "unsupported release manifest schema") {
		t.Fatalf("legacy schema rejection = %v", err)
	}
	manifest.Schema = ManifestSchema
	data, _ = json.Marshal(manifest)
	data = append(data[:len(data)-1], []byte(`,"unknown":true}`)...)
	_ = os.WriteFile(path, data, 0o644)
	if _, err := LoadManifest(path); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestVersionConsistency(t *testing.T) {
	root := testRoot(t)
	version, err := RecommendedVersion(root)
	if err != nil {
		t.Fatal(err)
	}
	var extension struct {
		Version string `json:"version"`
	}
	data, err := os.ReadFile(filepath.Join(root, "editors", "vscode", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &extension); err != nil {
		t.Fatal(err)
	}
	if extension.Version != version {
		t.Fatalf("extension version %q != canonical release/VERSION %q", extension.Version, version)
	}
}

func TestGitHubReleaseLayout(t *testing.T) {
	t.Parallel()

	flat := &Manifest{Artifacts: []Artifact{{Path: "compatibility.json"}, {Path: "security.md"}}}
	if err := validateGitHubReleaseLayout(flat); err != nil {
		t.Fatalf("flat release layout rejected: %v", err)
	}

	for name, manifest := range map[string]*Manifest{
		"directory":         {Artifacts: []Artifact{{Path: "reports/security.md"}}},
		"windows-directory": {Artifacts: []Artifact{{Path: `reports\security.md`}}},
		"duplicate":         {Artifacts: []Artifact{{Path: "security.md"}, {Path: "security.md"}}},
		"empty":             {Artifacts: []Artifact{{Path: ""}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateGitHubReleaseLayout(manifest); err == nil {
				t.Fatal("invalid public release layout accepted")
			}
		})
	}
}

func FuzzManifest(f *testing.F) {
	seed := Manifest{Schema: ManifestSchema, Version: "0.1.0-beta.1", Snapshot: true, Publication: "disabled", Commit: strings.Repeat("a", 40), Artifacts: []Artifact{{Path: "x", Kind: "test", Size: 1, Digest: Digest{Algorithm: "sha256", Value: strings.Repeat("0", 64)}}}}
	data, _ := json.Marshal(seed)
	f.Add(data)
	f.Add([]byte(`{"schema":"batchweaver.release/v0"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		path := filepath.Join(t.TempDir(), "manifest.json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		_, _ = LoadManifest(path)
	})
}

func TestDocumentation(t *testing.T) {
	root := testRoot(t)
	link := regexp.MustCompile(`\[[^]]*\]\(([^)]+)\)`)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == ".agent" || entry.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		inFence := false
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}
			for _, match := range link.FindAllStringSubmatch(line, -1) {
				target := match[1]
				if strings.ContainsAny(target, " <>\t\n") {
					continue
				}
				if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "#") {
					continue
				}
				target = strings.Split(target, "#")[0]
				if target == "" {
					continue
				}
				candidate := filepath.Join(filepath.Dir(path), filepath.FromSlash(target))
				if _, statErr := os.Stat(candidate); statErr != nil {
					return fmt.Errorf("%s: broken relative link %s", path, target)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"docs/release/release-policy.md", "docs/release/readiness-report.md", "docs/release/compatibility.md", "docs/release/security-report.md", "docs/release/performance-report.md", "docs/release/reproducibility-report.md", "docs/release/release-checklist.md", "docs/release/rollback.md", "docs/release/0.1.0-beta.1/launch-decision.md", "docs/release/0.1.0-beta.1/publication-blockers.md", "docs/release/0.1.0-beta.2/launch-decision.md", "docs/release/0.1.0-beta.2/publication-blockers.md", "docs/release/0.1.0-beta.3/launch-decision.md", "docs/release/0.1.0-beta.3/publication-blockers.md", "docs/release/release-notes-0.1.0-beta.3.md", "docs/release/beta-exit-criteria.md", "docs/privacy.md", "docs/maintainers/beta-operations.md", "site/index.html", "site/docs.html", "site/examples.html", "site/api.html", "site/status.html", "site/sitemap.xml", "site/robots.txt", "site/llms.txt", "site/site.webmanifest", "site/social-card.png", "KNOWN-ISSUES.md"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(required))); err != nil {
			t.Errorf("required release document %s: %v", required, err)
		}
	}
}

func TestDocumentationSiteSEOContract(t *testing.T) {
	root := testRoot(t)
	pages := map[string]string{
		"index.html":    "https://voskan.github.io/BatchWeaver/",
		"docs.html":     "https://voskan.github.io/BatchWeaver/docs.html",
		"examples.html": "https://voskan.github.io/BatchWeaver/examples.html",
		"api.html":      "https://voskan.github.io/BatchWeaver/api.html",
		"status.html":   "https://voskan.github.io/BatchWeaver/status.html",
	}
	required := []string{
		`<html lang="en">`,
		"<title>",
		`<meta name="description"`,
		`<meta property="og:title"`,
		`<meta property="og:description"`,
		`<meta property="og:image"`,
		`<meta name="twitter:card"`,
		`<script type="application/ld+json">`,
		`<main id="main"`,
	}
	for page, canonical := range pages {
		t.Run(page, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, "site", page))
			if err != nil {
				t.Fatal(err)
			}
			content := string(data)
			for _, marker := range required {
				if !strings.Contains(content, marker) {
					t.Errorf("missing SEO/accessibility marker %q", marker)
				}
			}
			if marker := `<link rel="canonical" href="` + canonical + `">`; !strings.Contains(content, marker) {
				t.Errorf("missing canonical URL %q", canonical)
			}
			if count := strings.Count(content, "<h1"); count != 1 {
				t.Errorf("h1 count = %d, want 1", count)
			}
		})
	}

	sitemapData, err := os.ReadFile(filepath.Join(root, "site", "sitemap.xml"))
	if err != nil {
		t.Fatal(err)
	}
	sitemap := string(sitemapData)
	for _, canonical := range pages {
		if !strings.Contains(sitemap, "<loc>"+canonical+"</loc>") {
			t.Errorf("sitemap is missing %s", canonical)
		}
	}
	robotsData, err := os.ReadFile(filepath.Join(root, "site", "robots.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(robotsData), "Sitemap: https://voskan.github.io/BatchWeaver/sitemap.xml") {
		t.Error("robots.txt does not advertise the canonical sitemap")
	}
}

func TestStableDecisionRemainsBlockedWithoutEvidence(t *testing.T) {
	root := testRoot(t)
	required := []string{
		"docs/release/v1.0.0/api-freeze.md",
		"docs/release/v1.0.0/beta-evidence.md",
		"docs/release/v1.0.0/compatibility.md",
		"docs/release/v1.0.0/documentation.md",
		"docs/release/v1.0.0/known-issues.md",
		"docs/release/v1.0.0/launch-health.md",
		"docs/release/v1.0.0/migration.md",
		"docs/release/v1.0.0/performance.md",
		"docs/release/v1.0.0/project-completion.md",
		"docs/release/v1.0.0/reproducibility.md",
		"docs/release/v1.0.0/security.md",
		"docs/release/v1.0.0/stable-release-decision.md",
		"release/api-inventory-v1.json",
		"release/gates-v1.0.0.json",
	}
	for _, rel := range required {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("read required stable evidence %s: %v", rel, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("required stable evidence %s is empty", rel)
		}
	}

	data, err := os.ReadFile(filepath.Join(root, "release", "gates-v1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var gates struct {
		Decision string `json:"decision"`
	}
	if err := json.Unmarshal(data, &gates); err != nil {
		t.Fatalf("decode stable gates: %v", err)
	}
	if gates.Decision != "blocked" {
		t.Fatalf("stable decision = %q, want blocked until public evidence passes", gates.Decision)
	}
}

func TestLaunchGateReportDecisionMatchesRequiredGateState(t *testing.T) {
	root := testRoot(t)
	versionData, err := os.ReadFile(filepath.Join(root, "release", "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	gateName := "gates-v" + strings.TrimSpace(string(versionData)) + ".json"
	data, err := os.ReadFile(filepath.Join(root, "release", gateName))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Decision string `json:"decision"`
		Gates    []struct {
			ID         string   `json:"id"`
			Required   bool     `json:"required"`
			Status     string   `json:"status"`
			Evidence   []string `json:"evidence"`
			Exceptions []string `json:"exceptions"`
		} `json:"gates"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	want := []string{"source-integrity", "repository-identity", "git-cleanliness", "license", "third-party-notices", "secrets", "vulnerabilities", "dependency-policy", "compatibility", "unit-tests", "race-tests", "fuzz-smoke", "differential-tests", "mutation-thresholds", "integration-tests", "soak-tests", "performance-budgets", "reproducible-builds", "artifact-verification", "SBOM", "provenance", "signatures", "documentation", "links", "examples", "installation", "upgrade-downgrade", "security-reporting", "rollback", "known-issues", "community-health"}
	if len(report.Gates) != len(want) {
		t.Fatalf("gate count=%d want=%d", len(report.Gates), len(want))
	}
	blocked := false
	for i, gate := range report.Gates {
		if gate.ID != want[i] || !gate.Required || len(gate.Evidence) == 0 {
			t.Errorf("gate %d is incomplete: %+v", i, gate)
		}
		switch gate.Status {
		case "pass":
		case "blocked", "fail":
			blocked = true
		case "accepted-risk":
			if len(gate.Exceptions) == 0 {
				t.Errorf("accepted risk %s has no exception", gate.ID)
			}
		case "not-applicable":
		default:
			t.Errorf("gate %s has invalid status %q", gate.ID, gate.Status)
		}
	}
	if blocked && report.Decision != "blocked" {
		t.Fatalf("mandatory blocked gates must block publication: %+v", report)
	}
	if !blocked && report.Decision != "ready" {
		t.Fatalf("closed gates must produce a ready publication decision: %+v", report)
	}
}

func TestPublicLaunchContentHasNoPrivateCoordinationReferences(t *testing.T) {
	root := testRoot(t)
	forbidden := regexp.MustCompile(`(?i)(prompt (?:0[1-9]|1[0-4])|/Users/|localhost|\.agent/)`)
	for _, base := range []string{"README.md", "docs", "site"} {
		path := filepath.Join(root, base)
		err := filepath.WalkDir(path, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || (filepath.Ext(path) != ".md" && filepath.Ext(path) != ".html") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if forbidden.Match(data) {
				return fmt.Errorf("public content contains private coordination reference: %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func testRoot(t *testing.T) string {
	t.Helper()
	root, err := repositoryRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	return root
}
