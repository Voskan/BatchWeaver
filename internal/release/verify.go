package release

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var requiredArchiveFiles = map[string]bool{
	"LICENSE": true, "NOTICE": true, "README.md": true, "CHANGELOG.md": true, "SECURITY.md": true,
	"THIRD_PARTY_NOTICES.md": true, "KNOWN-ISSUES.md": true,
	"completions/batchweaver.bash": true, "completions/_batchweaver": true,
	"completions/batchweaver.fish": true, "completions/batchweaver.ps1": true,
	"man/batchweaver.1": true, "schemas/release-manifest-v1.json": true, "schemas/catalog.json": true,
}

// LoadManifest strictly decodes the versioned release manifest.
func LoadManifest(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(io.LimitReader(f, 8<<20))
	dec.DisallowUnknownFields()
	var manifest Manifest
	if err := dec.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode release manifest: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("release manifest contains trailing data")
	}
	if manifest.Schema != ManifestSchema {
		return nil, fmt.Errorf("unsupported release manifest schema %q; supported: %s", manifest.Schema, ManifestSchema)
	}
	if !manifest.Snapshot || manifest.Publication != "disabled" {
		return nil, fmt.Errorf("BW9018: manifest is not an unpublished snapshot")
	}
	if len(manifest.Commit) != 40 {
		return nil, fmt.Errorf("invalid source commit %q", manifest.Commit)
	}
	if _, err := hex.DecodeString(manifest.Commit); err != nil {
		return nil, fmt.Errorf("invalid source commit: %w", err)
	}
	if manifest.Version == "" || len(manifest.Artifacts) == 0 {
		return nil, fmt.Errorf("release manifest is incomplete")
	}
	return &manifest, nil
}

// Verify checks artifact digests, sizes, references, checksums, and archive
// allowlists without network access.
func Verify(manifestPath string) error {
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}
	root := filepath.Dir(manifestPath)
	seen := make(map[string]bool)
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == "" || filepath.IsAbs(artifact.Path) || strings.Contains(filepath.ToSlash(artifact.Path), "../") {
			return fmt.Errorf("unexpected artifact path %q", artifact.Path)
		}
		if seen[artifact.Path] {
			return fmt.Errorf("duplicate artifact %q", artifact.Path)
		}
		seen[artifact.Path] = true
		path := filepath.Join(root, filepath.FromSlash(artifact.Path))
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return fmt.Errorf("artifact %s: %w", artifact.Path, statErr)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("artifact %s is not a regular file", artifact.Path)
		}
		digest, size, digestErr := fileDigest(path)
		if digestErr != nil {
			return fmt.Errorf("artifact %s: %w", artifact.Path, digestErr)
		}
		if size != artifact.Size || digest != artifact.Digest {
			return fmt.Errorf("BW9014: artifact %s checksum or size mismatch", artifact.Path)
		}
		if artifact.Kind == "binary-archive" {
			if err := verifyBinaryArchive(path, artifact.Platform, manifest.Version, manifest.Commit); err != nil {
				return err
			}
		} else if strings.HasPrefix(artifact.Kind, "sbom-") {
			if err := verifySBOM(path, artifact.Kind); err != nil {
				return err
			}
		} else if artifact.Kind == "provenance" {
			if err := verifyProvenance(path, manifest); err != nil {
				return err
			}
		} else if artifact.Kind == "vscode-vsix" {
			if err := verifyVSIX(path); err != nil {
				return err
			}
		}
	}
	for _, reference := range []string{manifest.References.Compatibility, manifest.References.Security, manifest.References.Performance, manifest.References.KnownIssues} {
		if reference == "" || !seen[reference] {
			return fmt.Errorf("manifest reference %q is not a declared artifact", reference)
		}
	}
	if err := verifyChecksums(filepath.Join(root, "SHA256SUMS"), manifest); err != nil {
		return err
	}
	return nil
}

func verifyVSIX(path string) error {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer zr.Close()
	required := map[string]bool{"extension/package.json": false, "extension/out/extension.js": false, "extension/NOTICE": false, "extension/THIRD_PARTY_NOTICES.md": false, "extension.vsixmanifest": false, "[Content_Types].xml": false}
	licensePresent := false
	for _, f := range zr.File {
		if err := validateArchiveName(f.Name); err != nil {
			return err
		}
		if strings.HasSuffix(f.Name, ".map") || strings.HasSuffix(f.Name, ".ts") || strings.Contains(f.Name, "node_modules/") {
			return fmt.Errorf("VSIX contains unexpected source/dependency path %s", f.Name)
		}
		if _, ok := required[f.Name]; ok {
			required[f.Name] = true
		}
		if f.Name == "extension/LICENSE" || f.Name == "extension/LICENSE.txt" {
			licensePresent = true
		}
	}
	for name, present := range required {
		if !present {
			return fmt.Errorf("VSIX missing %s", name)
		}
	}
	if !licensePresent {
		return fmt.Errorf("VSIX missing LICENSE")
	}
	return nil
}

func verifySBOM(path, kind string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: invalid JSON: %w", kind, err)
	}
	if kind == "sbom-spdx" {
		if doc["spdxVersion"] != "SPDX-2.3" || doc["SPDXID"] != "SPDXRef-DOCUMENT" {
			return fmt.Errorf("SBOM is not SPDX 2.3")
		}
		if packages, ok := doc["packages"].([]any); !ok || len(packages) == 0 {
			return fmt.Errorf("SPDX SBOM has no packages")
		}
		if relationships, ok := doc["relationships"].([]any); !ok || len(relationships) == 0 {
			return fmt.Errorf("SPDX SBOM has no relationships")
		}
	} else {
		if doc["bomFormat"] != "CycloneDX" || doc["specVersion"] != "1.5" {
			return fmt.Errorf("SBOM is not CycloneDX 1.5")
		}
		if components, ok := doc["components"].([]any); !ok || len(components) == 0 {
			return fmt.Errorf("CycloneDX SBOM has no components")
		}
	}
	return nil
}

func verifyProvenance(path string, manifest *Manifest) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var statement struct {
		Type          string `json:"_type"`
		PredicateType string `json:"predicateType"`
		Subject       []struct {
			Name   string            `json:"name"`
			Digest map[string]string `json:"digest"`
		} `json:"subject"`
	}
	if err := json.Unmarshal(data, &statement); err != nil {
		return err
	}
	if statement.Type != "https://in-toto.io/Statement/v1" || statement.PredicateType != "https://slsa.dev/provenance/v1" {
		return fmt.Errorf("invalid provenance statement type")
	}
	if len(statement.Subject) == 0 || len(statement.Subject) > len(manifest.Artifacts) {
		return fmt.Errorf("invalid provenance subject set")
	}
	for _, subject := range statement.Subject {
		if subject.Name == "" || subject.Digest["sha256"] == "" {
			return fmt.Errorf("invalid provenance subject")
		}
	}
	return nil
}

func verifyChecksums(path string, manifest *Manifest) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("checksums: %w", err)
	}
	defer f.Close()
	got := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 {
			return fmt.Errorf("invalid SHA256SUMS line")
		}
		got[parts[1]] = parts[0]
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	for _, artifact := range manifest.Artifacts {
		if got[artifact.Path] != artifact.Digest.Value {
			return fmt.Errorf("BW9014: SHA256SUMS mismatch for %s", artifact.Path)
		}
	}
	manifestDigest, _, err := fileDigest(filepath.Join(filepath.Dir(path), "release-manifest.json"))
	if err != nil {
		return err
	}
	if got["release-manifest.json"] != manifestDigest.Value {
		return fmt.Errorf("BW9014: SHA256SUMS mismatch for release-manifest.json")
	}
	if len(got) != len(manifest.Artifacts)+1 {
		return fmt.Errorf("SHA256SUMS contains unexpected entries")
	}
	return nil
}

func verifyBinaryArchive(path, platform, version, commit string) error {
	wantExecutable := "batchweaver"
	if strings.HasPrefix(platform, "windows/") {
		wantExecutable += ".exe"
	}
	want := make(map[string]bool, len(requiredArchiveFiles)+1)
	for name := range requiredArchiveFiles {
		want[name] = true
	}
	want[wantExecutable] = true
	got := make(map[string]bool)
	var executable []byte
	if strings.HasSuffix(path, ".zip") {
		zr, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer zr.Close()
		for _, f := range zr.File {
			if err := validateArchiveName(f.Name); err != nil {
				return err
			}
			if got[f.Name] {
				return fmt.Errorf("archive contains duplicate file %s", f.Name)
			}
			got[f.Name] = true
			if f.Name == wantExecutable && f.Mode()&0o111 == 0 {
				return fmt.Errorf("archive executable %s lacks executable metadata", f.Name)
			}
			if f.Name == wantExecutable {
				r, openErr := f.Open()
				if openErr != nil {
					return openErr
				}
				executable, openErr = io.ReadAll(io.LimitReader(r, 100<<20))
				r.Close()
				if openErr != nil {
					return openErr
				}
			}
		}
	} else {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		tr := tar.NewReader(gz)
		for {
			h, nextErr := tr.Next()
			if errors.Is(nextErr, io.EOF) {
				break
			}
			if nextErr != nil {
				return nextErr
			}
			if err := validateArchiveName(h.Name); err != nil {
				return err
			}
			if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA {
				return fmt.Errorf("archive contains non-regular entry %s", h.Name)
			}
			if got[h.Name] {
				return fmt.Errorf("archive contains duplicate file %s", h.Name)
			}
			got[h.Name] = true
			if h.Name == wantExecutable && h.Mode&0o111 == 0 {
				return fmt.Errorf("archive executable %s lacks executable metadata", h.Name)
			}
			if h.Name == wantExecutable {
				executable, nextErr = io.ReadAll(io.LimitReader(tr, 100<<20))
				if nextErr != nil {
					return nextErr
				}
			}
		}
	}
	for name := range want {
		if !got[name] {
			return fmt.Errorf("archive missing %s", name)
		}
	}
	for name := range got {
		if !want[name] {
			return fmt.Errorf("archive contains unexpected file %s", name)
		}
	}
	if NativePlatform(platform) {
		if err := verifyEmbeddedBuildInfo(executable, wantExecutable, version, commit); err != nil {
			return err
		}
	}
	return nil
}

func verifyEmbeddedBuildInfo(data []byte, name, version, commit string) error {
	dir, err := os.MkdirTemp("", "batchweaver-install-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o755); err != nil {
		return err
	}
	out, err := run(dir, nil, path, "version", "--json")
	if err != nil {
		return fmt.Errorf("packaged binary smoke: %w", err)
	}
	var info struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return fmt.Errorf("packaged version JSON: %w", err)
	}
	if info.Version != version || info.Commit != commit {
		return fmt.Errorf("packaged binary metadata mismatch: version=%q commit=%q", info.Version, info.Commit)
	}
	if _, err := run(dir, nil, path, "help"); err != nil {
		return fmt.Errorf("packaged help smoke: %w", err)
	}
	return nil
}

func validateArchiveName(name string) error {
	clean := filepath.ToSlash(filepath.Clean(name))
	if name == "" || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") || clean != filepath.ToSlash(name) {
		return fmt.Errorf("unsafe archive path %q", name)
	}
	return nil
}

// Reproduce rebuilds a manifest in an isolated output directory and compares
// every declared deterministic artifact byte-for-byte.
func Reproduce(manifestPath, root string) error {
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}
	temp, err := os.MkdirTemp("", "batchweaver-reproduce-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	rebuilt, err := Build(BuildOptions{Root: root, Output: temp, Version: manifest.Version, Snapshot: true})
	if err != nil {
		return err
	}
	want := make(map[string]Artifact)
	for _, artifact := range manifest.Artifacts {
		want[artifact.Path] = artifact
	}
	for _, artifact := range rebuilt.Artifacts {
		original, ok := want[artifact.Path]
		if !ok {
			return fmt.Errorf("BW9015: rebuilt unexpected artifact %s", artifact.Path)
		}
		if original.Digest != artifact.Digest || original.Size != artifact.Size {
			return fmt.Errorf("BW9015: artifact %s is not byte-reproducible", artifact.Path)
		}
		delete(want, artifact.Path)
	}
	if len(want) != 0 {
		missing := make([]string, 0, len(want))
		for path := range want {
			missing = append(missing, path)
		}
		sort.Strings(missing)
		return fmt.Errorf("BW9015: rebuilt artifacts missing: %s", strings.Join(missing, ", "))
	}
	for _, name := range []string{"release-manifest.json", "SHA256SUMS"} {
		original, _, digestErr := fileDigest(filepath.Join(filepath.Dir(manifestPath), name))
		if digestErr != nil {
			return digestErr
		}
		candidate, _, digestErr := fileDigest(filepath.Join(temp, name))
		if digestErr != nil {
			return digestErr
		}
		if original != candidate {
			return fmt.Errorf("BW9015: %s is not byte-reproducible", name)
		}
	}
	return nil
}

func NativePlatform(platform string) bool { return platform == runtime.GOOS+"/"+runtime.GOARCH }
