package release

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const modulePath = "github.com/Voskan/BatchWeaver"

type archiveEntry struct {
	name string
	data []byte
	mode os.FileMode
}

// Build constructs a deterministic, unsigned snapshot release. It does not
// access publishing APIs, credentials, registries, or package managers.
func Build(opts BuildOptions) (*Manifest, error) {
	root, err := repositoryRoot(opts.Root)
	if err != nil {
		return nil, err
	}
	if !opts.Snapshot {
		return nil, fmt.Errorf("BW9018: publication builds require explicit maintainer authorization; use --snapshot")
	}
	status, err := git(root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	if status != "" {
		return nil, fmt.Errorf("BW9001: release build requires a clean checkout; no artifacts were written")
	}
	if opts.Version == "" {
		opts.Version, err = RecommendedVersion(root)
		if err != nil {
			return nil, err
		}
	}
	if opts.SourceDate.IsZero() {
		opts.SourceDate = time.Unix(0, 0).UTC()
	}
	if len(opts.Targets) == 0 {
		opts.Targets = append([]Target(nil), DefaultTargets...)
	}
	out, err := cleanOutput(root, opts.Output)
	if err != nil {
		return nil, err
	}
	commit, err := git(root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	tree, err := sourceDigest(root)
	if err != nil {
		return nil, err
	}
	manifest := &Manifest{
		Schema:          ManifestSchema,
		Version:         opts.Version,
		Snapshot:        true,
		Publication:     "disabled",
		Commit:          commit,
		SourceTree:      tree,
		Toolchain:       Toolchain{GoVersion: runtime.Version(), CGO: "disabled", Trimpath: true, BuildVCS: false},
		RuntimeABI:      RuntimeABI,
		ConfigSchema:    1,
		Signing:         "disabled-for-unsigned-snapshot",
		ProvenanceModel: "local-build-statement; not a hosted-builder attestation",
		References: References{
			Compatibility: "reports/compatibility.json",
			Security:      "reports/security.md",
			Performance:   "reports/performance.json",
			KnownIssues:   "KNOWN-ISSUES.md",
		},
	}

	work, err := os.MkdirTemp("", "batchweaver-release-build-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(work) }()

	common, err := commonEntries(root)
	if err != nil {
		return nil, err
	}
	for _, target := range opts.Targets {
		if err := validateTarget(target); err != nil {
			return nil, err
		}
		binaryName := "batchweaver"
		if target.GOOS == "windows" {
			binaryName += ".exe"
		}
		binaryPath := filepath.Join(work, target.GOOS+"_"+target.GOARCH+"_"+binaryName)
		ldflags := strings.Join([]string{
			"-s", "-w",
			"-X", modulePath + "/internal/buildinfo.version=" + opts.Version,
			"-X", modulePath + "/internal/buildinfo.commit=" + commit,
			"-X", modulePath + "/internal/buildinfo.buildDate=unknown",
		}, " ")
		_, err = run(root, []string{"GOOS=" + target.GOOS, "GOARCH=" + target.GOARCH, "CGO_ENABLED=0"},
			"go", "build", "-trimpath", "-buildvcs=false", "-ldflags", ldflags, "-o", binaryPath, "./cmd/batchweaver")
		if err != nil {
			return nil, err
		}
		binary, err := os.ReadFile(binaryPath)
		if err != nil {
			return nil, err
		}
		entries := append([]archiveEntry{{name: binaryName, data: binary, mode: 0o755}}, common...)
		base := fmt.Sprintf("batchweaver_%s_%s", target.GOOS, target.GOARCH)
		archivePath := filepath.Join(out, base+".tar.gz")
		if target.GOOS == "windows" {
			archivePath = filepath.Join(out, base+".zip")
			err = writeZip(archivePath, entries, opts.SourceDate)
		} else {
			err = writeTarGz(archivePath, entries, opts.SourceDate)
		}
		if err != nil {
			return nil, err
		}
		artifact, err := describeArtifact(out, archivePath, "binary-archive", target.String())
		if err != nil {
			return nil, err
		}
		artifact.Reproducibility = "byte-reproducible-under-declared-toolchain"
		manifest.Artifacts = append(manifest.Artifacts, artifact)
	}

	sourcePath := filepath.Join(out, "batchweaver_"+opts.Version+"_source.tar.gz")
	if err := writeSourceArchive(root, sourcePath, opts.SourceDate); err != nil {
		return nil, err
	}
	sourceArtifact, err := describeArtifact(out, sourcePath, "source-archive", "")
	if err != nil {
		return nil, err
	}
	sourceArtifact.Reproducibility = "byte-reproducible-from-identical-index"
	manifest.Artifacts = append(manifest.Artifacts, sourceArtifact)

	vsixPath := filepath.Join(out, "batchweaver-vscode.vsix")
	if err := buildVSIX(root, vsixPath, opts.SourceDate); err != nil {
		return nil, fmt.Errorf("VSIX packaging: %w", err)
	}
	vsixArtifact, err := describeArtifact(out, vsixPath, "vscode-vsix", "vscode>=1.85")
	if err != nil {
		return nil, err
	}
	vsixArtifact.Reproducibility = "byte-reproducible-under-declared-node-and-lockfile"
	manifest.Artifacts = append(manifest.Artifacts, vsixArtifact)

	reportDir := filepath.Join(out, "reports")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return nil, err
	}
	reports := []struct{ src, dst, kind string }{
		{"release/compatibility.json", "reports/compatibility.json", "compatibility-report"},
		{"docs/release/readiness-report.md", "reports/readiness.md", "readiness-report"},
		{"docs/release/security-report.md", "reports/security.md", "security-report"},
		{"release/performance-budgets.json", "reports/performance.json", "performance-policy"},
		{"docs/release/reproducibility-report.md", "reports/reproducibility.md", "reproducibility-report"},
		{"KNOWN-ISSUES.md", "KNOWN-ISSUES.md", "known-issues"},
		{"docs/release/release-notes-0.1.0-rc.1.md", "RELEASE-NOTES.md", "release-notes"},
		{"docs/release/release-checklist.md", "RELEASE-CHECKLIST.md", "release-checklist"},
	}
	for _, report := range reports {
		dst := filepath.Join(out, filepath.FromSlash(report.dst))
		if err := copyFile(dst, filepath.Join(root, filepath.FromSlash(report.src)), 0o644); err != nil {
			return nil, err
		}
		artifact, descErr := describeArtifact(out, dst, report.kind, "")
		if descErr != nil {
			return nil, descErr
		}
		artifact.Reproducibility = "byte-reproducible-from-source"
		manifest.Artifacts = append(manifest.Artifacts, artifact)
	}

	spdxPath := filepath.Join(out, "batchweaver.spdx.json")
	cyclonePath := filepath.Join(out, "batchweaver.cdx.json")
	if err := writeSBOMs(root, manifest, spdxPath, cyclonePath); err != nil {
		return nil, fmt.Errorf("BW9012: %w", err)
	}
	provenancePath := filepath.Join(out, "batchweaver.provenance.json")
	if err := writeProvenance(manifest, provenancePath); err != nil {
		return nil, fmt.Errorf("BW9013: %w", err)
	}
	for i := range manifest.Artifacts {
		manifest.Artifacts[i].SBOM = "batchweaver.spdx.json,batchweaver.cdx.json"
		manifest.Artifacts[i].Provenance = "batchweaver.provenance.json"
	}
	for _, item := range []struct{ path, kind string }{{spdxPath, "sbom-spdx"}, {cyclonePath, "sbom-cyclonedx"}, {provenancePath, "provenance"}} {
		a, descErr := describeArtifact(out, item.path, item.kind, "")
		if descErr != nil {
			return nil, descErr
		}
		a.Reproducibility = "byte-reproducible-under-declared-toolchain"
		manifest.Artifacts = append(manifest.Artifacts, a)
	}
	sort.Slice(manifest.Artifacts, func(i, j int) bool { return manifest.Artifacts[i].Path < manifest.Artifacts[j].Path })
	manifestPath := filepath.Join(out, "release-manifest.json")
	if err := writeJSON(manifestPath, manifest); err != nil {
		return nil, err
	}
	if err := writeChecksums(out, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func validateTarget(target Target) error {
	for _, allowed := range DefaultTargets {
		if target == allowed {
			return nil
		}
	}
	return fmt.Errorf("unsupported release target %s", target.String())
}

func commonEntries(root string) ([]archiveEntry, error) {
	var entries []archiveEntry
	for _, name := range []string{"LICENSE", "NOTICE", "README.md", "CHANGELOG.md", "SECURITY.md", "THIRD_PARTY_NOTICES.md", "KNOWN-ISSUES.md"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return nil, err
		}
		entries = append(entries, archiveEntry{name: name, data: data, mode: 0o644})
	}
	generated := map[string]string{
		"completions/batchweaver.bash":     completionBash,
		"completions/_batchweaver":         completionZsh,
		"completions/batchweaver.fish":     completionFish,
		"completions/batchweaver.ps1":      completionPowerShell,
		"man/batchweaver.1":                manPage,
		"schemas/release-manifest-v1.json": releaseManifestSchema,
		"schemas/catalog.json":             schemaCatalog,
	}
	for name, body := range generated {
		entries = append(entries, archiveEntry{name: name, data: []byte(body), mode: 0o644})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries, nil
}

func writeTarGz(path string, entries []archiveEntry, stamp time.Time) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	gz, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	gz.ModTime = stamp
	gz.Name = ""
	gz.Comment = ""
	tw := tar.NewWriter(gz)
	for _, entry := range entries {
		h := &tar.Header{Name: filepath.ToSlash(entry.name), Mode: int64(entry.mode.Perm()), Size: int64(len(entry.data)), ModTime: stamp, Uid: 0, Gid: 0, Format: tar.FormatUSTAR}
		if err := tw.WriteHeader(h); err != nil {
			_ = f.Close()
			return err
		}
		if _, err := tw.Write(entry.data); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := tw.Close(); err != nil {
		_ = f.Close()
		return err
	}
	if err := gz.Close(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeZip(path string, entries []archiveEntry, stamp time.Time) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	for _, entry := range entries {
		h := &zip.FileHeader{Name: filepath.ToSlash(entry.name), Method: zip.Deflate}
		h.SetMode(entry.mode)
		h.Modified = stamp
		w, createErr := zw.CreateHeader(h)
		if createErr != nil {
			_ = f.Close()
			return createErr
		}
		if _, createErr = w.Write(entry.data); createErr != nil {
			_ = f.Close()
			return createErr
		}
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func buildVSIX(root, output string, stamp time.Time) error {
	versionOut, err := run(root, nil, "node", "--version")
	if err != nil {
		return fmt.Errorf("node 22 is required: %w", err)
	}
	majorText := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(string(versionOut)), "v"), ".", 2)[0]
	major, parseErr := strconv.Atoi(majorText)
	if parseErr != nil || major < 22 {
		return fmt.Errorf("node 22 or newer is required, found %s", strings.TrimSpace(string(versionOut)))
	}
	extension := filepath.Join(root, "editors", "vscode")
	for _, command := range [][]string{{"ci"}, {"audit", "--audit-level=high"}, {"run", "lint"}, {"run", "typecheck"}, {"run", "compile"}, {"test"}, {"run", "package"}} {
		if _, err := run(extension, nil, "npm", command...); err != nil {
			return err
		}
	}
	input := filepath.Join(extension, "batchweaver-vscode.vsix")
	defer func() { _ = os.Remove(input) }()
	return normalizeZip(input, output, stamp)
}

func normalizeZip(input, output string, stamp time.Time) error {
	zr, err := zip.OpenReader(input)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()
	entries := make([]archiveEntry, 0, len(zr.File))
	for _, f := range zr.File {
		if err := validateArchiveName(f.Name); err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			continue
		}
		r, openErr := f.Open()
		if openErr != nil {
			return openErr
		}
		data, readErr := io.ReadAll(io.LimitReader(r, 100<<20))
		_ = r.Close()
		if readErr != nil {
			return readErr
		}
		entries = append(entries, archiveEntry{name: f.Name, data: data, mode: f.Mode()})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return writeZip(output, entries, stamp)
}

func writeSourceArchive(root, path string, stamp time.Time) error {
	listedBytes, err := run(root, nil, "git", "ls-files", "-z")
	if err != nil {
		return err
	}
	paths := strings.Split(string(listedBytes), "\x00")
	sort.Strings(paths)
	entries := make([]archiveEntry, 0, len(paths))
	for _, rel := range paths {
		if rel == "" || strings.HasPrefix(rel, ".agent/") || strings.HasPrefix(rel, "dist/") {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(rel))
		info, statErr := os.Lstat(full)
		if statErr != nil {
			return statErr
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("source archive rejects non-regular tracked path %s", rel)
		}
		data, readErr := os.ReadFile(full)
		if readErr != nil {
			return readErr
		}
		mode := os.FileMode(0o644)
		if info.Mode()&0o111 != 0 {
			mode = 0o755
		}
		entries = append(entries, archiveEntry{name: filepath.ToSlash(rel), data: data, mode: mode})
	}
	return writeTarGz(path, entries, stamp)
}

func describeArtifact(root, path, kind, platform string) (Artifact, error) {
	digest, size, err := fileDigest(path)
	if err != nil {
		return Artifact{}, err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: filepath.ToSlash(rel), Kind: kind, Platform: platform, Size: size, Digest: digest}, nil
}

func writeChecksums(out string, manifest *Manifest) error {
	var b strings.Builder
	for _, artifact := range manifest.Artifacts {
		fmt.Fprintf(&b, "%s  %s\n", artifact.Digest.Value, artifact.Path)
	}
	manifestDigest, _, err := fileDigest(filepath.Join(out, "release-manifest.json"))
	if err != nil {
		return err
	}
	fmt.Fprintf(&b, "%s  release-manifest.json\n", manifestDigest.Value)
	return os.WriteFile(filepath.Join(out, "SHA256SUMS"), []byte(b.String()), 0o644)
}

type module struct {
	Path, Version, Sum string
	Main               bool
	Replace            *module
}

type npmPackage struct{ Path, Name, Version, License string }

func npmPackages(root string) ([]npmPackage, error) {
	data, err := os.ReadFile(filepath.Join(root, "editors", "vscode", "package-lock.json"))
	if err != nil {
		return nil, err
	}
	var lock struct {
		Packages map[string]struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			License string `json:"license"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}
	result := make([]npmPackage, 0, len(lock.Packages))
	for path, item := range lock.Packages {
		if path == "" || item.Version == "" {
			continue
		}
		name := item.Name
		if name == "" {
			name = path[strings.LastIndex(path, "node_modules/")+len("node_modules/"):]
		}
		license := item.License
		if license == "" {
			var installed struct {
				License any `json:"license"`
			}
			if installedData, readErr := os.ReadFile(filepath.Join(root, "editors", "vscode", filepath.FromSlash(path), "package.json")); readErr == nil && json.Unmarshal(installedData, &installed) == nil {
				switch value := installed.License.(type) {
				case string:
					license = value
				case map[string]any:
					if text, ok := value["type"].(string); ok {
						license = text
					}
				}
			}
		}
		result = append(result, npmPackage{Path: path, Name: name, Version: item.Version, License: license})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func modules(root string) ([]module, error) {
	out, err := run(root, nil, "go", "list", "-m", "-json", "all")
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var result []module
	for {
		var item module
		if err := dec.Decode(&item); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

func writeSBOMs(root string, manifest *Manifest, spdxPath, cyclonePath string) error {
	mods, err := modules(root)
	if err != nil {
		return err
	}
	npm, err := npmPackages(root)
	if err != nil {
		return err
	}
	if err := validateNPMLicenses(npm); err != nil {
		return err
	}
	type spdxPackage struct {
		Name             string `json:"name"`
		SPDXID           string `json:"SPDXID"`
		Version          string `json:"versionInfo,omitempty"`
		Download         string `json:"downloadLocation"`
		FilesAnalyzed    bool   `json:"filesAnalyzed"`
		LicenseConcluded string `json:"licenseConcluded"`
		LicenseDeclared  string `json:"licenseDeclared"`
		CopyrightText    string `json:"copyrightText"`
	}
	type spdxRelationship struct {
		SPDXElementID string `json:"spdxElementId"`
		Type          string `json:"relationshipType"`
		Related       string `json:"relatedSpdxElement"`
	}
	spdx := struct {
		SPDXVersion   string             `json:"spdxVersion"`
		DataLicense   string             `json:"dataLicense"`
		SPDXID        string             `json:"SPDXID"`
		Name          string             `json:"name"`
		Namespace     string             `json:"documentNamespace"`
		Creation      map[string]any     `json:"creationInfo"`
		Packages      []spdxPackage      `json:"packages"`
		Relationships []spdxRelationship `json:"relationships"`
	}{"SPDX-2.3", "CC0-1.0", "SPDXRef-DOCUMENT", "BatchWeaver " + manifest.Version, "https://github.com/Voskan/BatchWeaver/sbom/" + manifest.Commit, map[string]any{"creators": []string{"Tool: batchweaver-release"}, "created": "1970-01-01T00:00:00Z"}, nil, nil}
	type cdxComponent struct {
		Type    string `json:"type"`
		Ref     string `json:"bom-ref"`
		Name    string `json:"name"`
		Version string `json:"version,omitempty"`
		PURL    string `json:"purl,omitempty"`
	}
	cdx := struct {
		BOMFormat    string           `json:"bomFormat"`
		SpecVersion  string           `json:"specVersion"`
		SerialNumber string           `json:"serialNumber"`
		Version      int              `json:"version"`
		Metadata     map[string]any   `json:"metadata"`
		Components   []cdxComponent   `json:"components"`
		Dependencies []map[string]any `json:"dependencies"`
	}{"CycloneDX", "1.5", "urn:uuid:" + manifest.Commit[:8] + "-0000-4000-8000-" + manifest.Commit[8:20], 1, map[string]any{"timestamp": "1970-01-01T00:00:00Z", "component": map[string]any{"type": "application", "bom-ref": "pkg:golang/" + modulePath + "@" + manifest.Version, "name": "BatchWeaver", "version": manifest.Version, "hashes": []map[string]string{{"alg": "SHA-256", "content": manifest.SourceTree.Value}}}}, nil, nil}
	mainID := "SPDXRef-Package-BatchWeaver"
	spdx.Relationships = append(spdx.Relationships, spdxRelationship{"SPDXRef-DOCUMENT", "DESCRIBES", mainID})
	var cdxDependsOn []string
	for i, mod := range mods {
		version := mod.Version
		if mod.Main {
			version = manifest.Version
		}
		id := fmt.Sprintf("SPDXRef-Package-%d", i)
		license, copyrightText := goModuleLicense(mod.Path), "NOASSERTION"
		if mod.Main {
			id, license, copyrightText = mainID, "Apache-2.0", "Copyright 2026 Voskan Voskanyan"
		}
		spdx.Packages = append(spdx.Packages, spdxPackage{Name: mod.Path, SPDXID: id, Version: version, Download: "NOASSERTION", FilesAnalyzed: false, LicenseConcluded: license, LicenseDeclared: license, CopyrightText: copyrightText})
		ref := "pkg:golang/" + mod.Path + "@" + version
		cdx.Components = append(cdx.Components, cdxComponent{Type: "library", Ref: ref, Name: mod.Path, Version: version, PURL: ref})
		if !mod.Main {
			spdx.Relationships = append(spdx.Relationships, spdxRelationship{mainID, "DEPENDS_ON", id})
			cdxDependsOn = append(cdxDependsOn, ref)
		}
	}
	for i, pkg := range npm {
		license := pkg.License
		if license == "" {
			license = "NOASSERTION"
		}
		id := fmt.Sprintf("SPDXRef-NPM-%d", i)
		spdx.Packages = append(spdx.Packages, spdxPackage{Name: pkg.Name, SPDXID: id, Version: pkg.Version, Download: "NOASSERTION", FilesAnalyzed: false, LicenseConcluded: license, LicenseDeclared: license, CopyrightText: "NOASSERTION"})
		spdx.Relationships = append(spdx.Relationships, spdxRelationship{mainID, "CONTAINS", id})
		ref := "npm:" + pkg.Path + "@" + pkg.Version
		cdx.Components = append(cdx.Components, cdxComponent{Type: "library", Ref: ref, Name: pkg.Name, Version: pkg.Version})
		cdxDependsOn = append(cdxDependsOn, ref)
	}
	cdx.Dependencies = append(cdx.Dependencies, map[string]any{"ref": "pkg:golang/" + modulePath + "@" + manifest.Version, "dependsOn": cdxDependsOn})
	if err := writeJSON(spdxPath, spdx); err != nil {
		return err
	}
	return writeJSON(cyclonePath, cdx)
}

func goModuleLicense(path string) string {
	switch {
	case path == modulePath:
		return "Apache-2.0"
	case path == "github.com/goccy/go-yaml":
		return "MIT"
	case strings.HasPrefix(path, "golang.org/x/"):
		return "BSD-3-Clause"
	default:
		return "NOASSERTION"
	}
}

func validateNPMLicenses(packages []npmPackage) error {
	for _, pkg := range packages {
		if pkg.License == "" {
			if strings.Contains(pkg.Path, "node_modules/@esbuild/") || strings.Contains(pkg.Path, "node_modules/@vscode/vsce-sign-") {
				continue
			}
			return fmt.Errorf("npm dependency %s@%s has no reviewable license metadata", pkg.Name, pkg.Version)
		}
		upper := strings.ToUpper(pkg.License)
		if strings.Contains(upper, "AGPL") || strings.Contains(upper, "SSPL") || strings.Contains(upper, "GPL-") || strings.HasPrefix(upper, "GPL") {
			return fmt.Errorf("npm dependency %s@%s has incompatible license %s", pkg.Name, pkg.Version, pkg.License)
		}
	}
	return nil
}

func writeProvenance(manifest *Manifest, path string) error {
	type subject struct {
		Name   string            `json:"name"`
		Digest map[string]string `json:"digest"`
	}
	statement := struct {
		Type          string         `json:"_type"`
		Subject       []subject      `json:"subject"`
		PredicateType string         `json:"predicateType"`
		Predicate     map[string]any `json:"predicate"`
	}{Type: "https://in-toto.io/Statement/v1", PredicateType: "https://slsa.dev/provenance/v1", Predicate: map[string]any{"buildDefinition": map[string]any{"buildType": "https://github.com/Voskan/BatchWeaver/release/snapshot/v1", "externalParameters": map[string]any{"version": manifest.Version, "snapshot": true}, "resolvedDependencies": []map[string]any{{"uri": "git+https://github.com/Voskan/BatchWeaver", "digest": map[string]string{"gitCommit": manifest.Commit}}}}, "runDetails": map[string]any{"builder": map[string]string{"id": "local:batchweaver-release"}, "metadata": map[string]bool{"invocationId": false}}}}
	for _, artifact := range manifest.Artifacts {
		statement.Subject = append(statement.Subject, subject{Name: artifact.Path, Digest: map[string]string{"sha256": artifact.Digest.Value}})
	}
	return writeJSON(path, statement)
}

const completionBash = `complete -W "version config operation scan prove candidate proof assumption strategy transform build test run runtime barrier tool-exec toolexec adapter graphql grpc http openapi profile tune fairness overload wave recursive lsp daemon editor verify compatibility security release help" batchweaver
`
const completionZsh = `#compdef batchweaver
_arguments '1:command:(version config operation scan prove candidate proof assumption strategy transform build test run runtime barrier tool-exec toolexec adapter graphql grpc http openapi profile tune fairness overload wave recursive lsp daemon editor verify compatibility security release help)'
`
const completionFish = `complete -c batchweaver -f -a 'version config operation scan prove candidate proof assumption strategy transform build test run runtime barrier tool-exec toolexec adapter graphql grpc http openapi profile tune fairness overload wave recursive lsp daemon editor verify compatibility security release help'
`
const completionPowerShell = `Register-ArgumentCompleter -Native -CommandName batchweaver -ScriptBlock { param($wordToComplete) 'version','config','operation','scan','prove','candidate','proof','assumption','strategy','transform','build','test','run','runtime','barrier','tool-exec','toolexec','adapter','graphql','grpc','http','openapi','profile','tune','fairness','overload','wave','recursive','lsp','daemon','editor','verify','compatibility','security','release','help' | Where-Object { $_ -like "$wordToComplete*" } }
`
const manPage = `.TH BATCHWEAVER 1 "" "BatchWeaver snapshot" "User Commands"
.SH NAME
batchweaver \- semantic batching compiler and runtime for Go
.SH SYNOPSIS
.B batchweaver
COMMAND [ARGUMENTS]
.SH RELEASE ASSURANCE
Use \fBbatchweaver release check\fR, \fBbuild --snapshot\fR, \fBverify\fR, and \fBreproduce\fR. No release command publishes artifacts.
`
const releaseManifestSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://batchweaver.dev/schemas/release-manifest-v1.json","title":"BatchWeaver release manifest","type":"object","required":["schema","version","snapshot","publication","commit","artifacts"],"properties":{"schema":{"const":"batchweaver.release/v1alpha1"},"version":{"type":"string"},"snapshot":{"const":true},"publication":{"const":"disabled"},"commit":{"type":"string","pattern":"^[0-9a-f]{40}$"},"artifacts":{"type":"array"}},"additionalProperties":true}
`
const schemaCatalog = `{"schema":"batchweaver.schema-catalog/v1alpha1","artifacts":[{"name":"analysis","id":"batchweaver.analysis/v1alpha1"},{"name":"proof","id":"batchweaver.proof/v1alpha1"},{"name":"transform","id":"batchweaver.transform/v1alpha1"},{"name":"runtime","id":"batchweaver.runtime/v1alpha1"},{"name":"release","id":"batchweaver.release/v1alpha1"}],"note":"The repository reference documentation is authoritative for artifacts without a standalone JSON Schema."}
`
