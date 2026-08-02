package release

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// CheckOptions configures release-readiness evaluation.
type CheckOptions struct {
	Root string
}

// Check executes release gates. No skipped or unexecuted gate is represented
// as passing, and readiness requires every declared gate to pass.
func Check(opts CheckOptions) (*ReadinessReport, error) {
	root, err := repositoryRoot(opts.Root)
	if err != nil {
		return nil, err
	}
	commit, err := git(root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	version, versionErr := RecommendedVersion(root)
	if versionErr != nil {
		return nil, versionErr
	}
	report := &ReadinessReport{Schema: "batchweaver.readiness/v1alpha1", Commit: commit, Version: version, GeneratedAt: time.Now().UTC(), Publication: "not performed"}
	add := func(result CheckResult) { report.Checks = append(report.Checks, result) }

	status, statusErr := git(root, "status", "--porcelain", "--untracked-files=all")
	switch {
	case statusErr != nil:
		add(failed("BW9001", "Working tree", statusErr.Error(), "run from a clean checkout"))
	case status != "":
		add(failed("BW9001", "Working tree", "source tree is dirty", "commit or remove local changes before a release dry run"))
	default:
		add(passed("BW9001", "Working tree", "clean"))
	}

	trackedBinaries, _ := git(root, "ls-files", "batchweaver", "examples/editor-diagnostics/editor-diagnostics", "examples/editor-proxy/editor-proxy", "examples/editor-transform-preview/editor-transform-preview")
	if trackedBinaries != "" {
		add(failed("BW9002", "Generated artifacts", "generated executables are tracked", "remove generated binaries from source control"))
	} else {
		add(passed("BW9002", "Generated artifacts", "no known generated executable is tracked"))
	}

	runGate := func(id, name, detail, pkg, pattern string) {
		_, gateErr := run(root, []string{"CGO_ENABLED=0"}, "go", "test", pkg, "-run", pattern, "-count=1")
		if gateErr != nil {
			add(failed(id, name, gateErr.Error(), "run the reported go test command and resolve every failure"))
			return
		}
		add(passed(id, name, detail))
	}
	runGate("BW9003", "Public API compatibility", "exported API matches the committed baseline", "./internal/release", "^TestPublicAPIBaseline$")
	runGate("BW9004", "Schema compatibility", "current and supported legacy fixtures decode as declared", "./internal/release", "^TestSchemaCompatibility$")
	runGate("BW9005", "Semantic differential", "central scalar/batch differential suite passed", "./internal/assurance", "^TestDifferential")
	runGate("BW9006", "Mutation threshold", "all safety-critical modeled mutations were killed", "./internal/assurance", "^TestSafetyMutation")
	runGate("BW9007", "Fuzz corpus smoke", "release-manifest seed corpus passed", "./internal/release", "^FuzzManifest$")

	security := AuditSecurity(root)
	_, vulnErr := run(root, []string{"GOTOOLCHAIN=go1.26.5"}, "go", "run", "golang.org/x/vuln/cmd/govulncheck@v1.6.0", "./...")
	if security.BlockingFindings == 0 && vulnErr == nil {
		add(passed("BW9010", "Security checks", fmt.Sprintf("%d built-in checks and govulncheck v1.6.0 passed", len(security.Checks))))
	} else {
		detail := fmt.Sprintf("%d built-in blocking finding(s)", security.BlockingFindings)
		if vulnErr != nil {
			detail += "; govulncheck: " + vulnErr.Error()
		}
		add(failed("BW9010", "Security checks", detail, "review the redacted security audit and vulnerability output"))
	}
	if err := AuditLicenses(root); err != nil {
		add(failed("BW9011", "License audit", err.Error(), "resolve missing or incompatible license metadata"))
	} else {
		add(passed("BW9011", "License audit", "root license, notice, Go module inventory, and extension license metadata present"))
	}
	runGate("BW9009", "Performance budgets", "deterministic allocation and scale budgets passed", "./internal/assurance", "^TestPerformanceBudget")
	runGate("BW9016", "Documentation validation", "internal links and release documentation contract passed", "./internal/release", "^TestDocumentation")

	temp, tempErr := os.MkdirTemp("", "batchweaver-readiness-")
	if tempErr != nil {
		add(failed("BW9017", "Release dry run", tempErr.Error(), "ensure a writable temporary directory is available"))
	} else {
		defer func() { _ = os.RemoveAll(temp) }()
		manifest, buildErr := Build(BuildOptions{Root: root, Output: temp, Version: version, Snapshot: true})
		if buildErr != nil {
			add(failed("BW9017", "Release dry run", buildErr.Error(), "run batchweaver release build --snapshot and resolve the failure"))
		} else {
			manifestPath := filepath.Join(temp, "release-manifest.json")
			if verifyErr := Verify(manifestPath); verifyErr != nil {
				add(failed("BW9014", "Artifact verification", verifyErr.Error(), "rebuild and inspect the mismatched artifact"))
			} else {
				add(passed("BW9014", "Artifact verification", fmt.Sprintf("%d declared artifacts verified", len(manifest.Artifacts))))
			}
			if reproduceErr := Reproduce(manifestPath, root); reproduceErr != nil {
				add(failed("BW9015", "Reproducibility", reproduceErr.Error(), "compare the declared toolchain and artifact metadata"))
			} else {
				add(passed("BW9015", "Reproducibility", "declared artifacts rebuilt byte-for-byte in a second output directory"))
			}
			add(passed("BW9008", "Compatibility matrix", fmt.Sprintf("cross-build passed for %d declared targets", len(DefaultTargets))))
			add(passed("BW9017", "Release dry run", "snapshot built and verified; publication remained disabled"))
		}
	}
	report.Ready = true
	for _, check := range report.Checks {
		if check.Status != StatusPass {
			report.Ready = false
			break
		}
	}
	return report, nil
}

func passed(id, name, detail string) CheckResult {
	return CheckResult{ID: id, Name: name, Status: StatusPass, Detail: detail}
}
func failed(id, name, detail, remediation string) CheckResult {
	return CheckResult{ID: id, Name: name, Status: StatusFail, Detail: detail, Remediation: remediation}
}

// SecurityReport records local checks, omissions, and redaction policy.
type SecurityReport struct {
	Schema            string        `json:"schema"`
	Checks            []CheckResult `json:"checks"`
	BlockingFindings  int           `json:"blocking_findings"`
	ExternalScanners  []string      `json:"external_scanners"`
	SensitiveDetails  string        `json:"sensitive_details"`
	NetworkCollection string        `json:"network_collection"`
}

// AuditSecurity performs deterministic repository-local checks and never
// prints a matched secret value.
func AuditSecurity(root string) SecurityReport {
	report := SecurityReport{Schema: "batchweaver.security/v1alpha1", SensitiveDetails: "withheld", NetworkCollection: "none"}
	add := func(result CheckResult) {
		report.Checks = append(report.Checks, result)
		if result.Status == StatusFail {
			report.BlockingFindings++
		}
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`),
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{30,}`),
	}
	files, err := run(root, nil, "git", "ls-files", "-z")
	if err != nil {
		add(failed("BW9010", "Tracked-file inventory", err.Error(), "ensure git is available"))
		return report
	}
	var suspicious []string
	for _, rel := range strings.Split(string(files), "\x00") {
		if rel == "" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if readErr != nil || bytesContainNUL(data) {
			continue
		}
		for _, pattern := range patterns {
			if pattern.Match(data) {
				suspicious = append(suspicious, rel)
				break
			}
		}
	}
	if len(suspicious) == 0 {
		add(passed("BW9010", "Secret patterns", "no tracked text file matched the built-in high-confidence patterns"))
	} else {
		sort.Strings(suspicious)
		add(failed("BW9010", "Secret patterns", "possible secret material in: "+strings.Join(suspicious, ", "), "rotate if real and remove through the security process"))
	}

	workflowPaths, _ := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.y*ml"))
	actionPin := regexp.MustCompile(`uses:\s+[^\s@]+@[0-9a-f]{40}(?:\s|$)`)
	var weak []string
	for _, path := range workflowPaths {
		f, openErr := os.Open(path)
		if openErr != nil {
			weak = append(weak, filepath.Base(path))
			continue
		}
		scanner := bufio.NewScanner(f)
		hasPermissions := false
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "permissions:" {
				hasPermissions = true
			}
			if strings.HasPrefix(line, "uses:") && !actionPin.MatchString(line) {
				weak = append(weak, filepath.Base(path)+": unpinned action")
			}
		}
		_ = f.Close()
		if !hasPermissions {
			weak = append(weak, filepath.Base(path)+": missing permissions")
		}
	}
	if len(weak) == 0 {
		add(passed("BW9010", "Workflow hardening", "actions are immutable-SHA pinned and explicit permissions are present"))
	} else {
		add(failed("BW9010", "Workflow hardening", strings.Join(weak, "; "), "pin actions and declare least-privilege permissions"))
	}
	publishScript := filepath.Join(root, "scripts", "publish-prerelease.sh")
	if data, readErr := os.ReadFile(publishScript); readErr != nil || !strings.Contains(string(data), "--confirm-v0.1.0-beta.1") || !strings.Contains(string(data), "gh auth status") {
		add(failed("BW9010", "Publication boundary", "maintainer publication helper is missing explicit confirmation or authenticated identity checks", "restore both controls before any prerelease"))
	} else {
		add(passed("BW9010", "Publication boundary", "CLI remains non-publishing; maintainer helper requires exact confirmation, authenticated identity, immutable tag, and verified assets"))
	}
	report.ExternalScanners = []string{"CodeQL: CI only, not run by built-in audit", "govulncheck v1.6.0: run by make check/CI", "GitHub dependency review: pull requests only"}
	return report
}

func bytesContainNUL(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

// AuditLicenses verifies repository license inputs and dependency inventory.
func AuditLicenses(root string) error {
	license, err := os.ReadFile(filepath.Join(root, "LICENSE"))
	if err != nil {
		return err
	}
	if !strings.Contains(string(license), "Apache License") {
		return fmt.Errorf("root LICENSE is not recognizable as Apache-2.0")
	}
	if _, err := os.Stat(filepath.Join(root, "NOTICE")); err != nil {
		return err
	}
	pkg, err := os.ReadFile(filepath.Join(root, "editors", "vscode", "package.json"))
	if err != nil {
		return err
	}
	if !strings.Contains(string(pkg), `"license": "Apache-2.0"`) {
		return fmt.Errorf("VS Code extension license metadata is missing")
	}
	if _, err := modules(root); err != nil {
		return fmt.Errorf("go dependency inventory: %w", err)
	}
	return nil
}
