package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func repositoryRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		data, readErr := os.ReadFile(filepath.Join(dir, "go.mod"))
		if readErr == nil && strings.Contains(string(data), "module github.com/Voskan/BatchWeaver") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("BatchWeaver repository root not found from %s", start)
		}
	}
}

func run(root string, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func git(root string, args ...string) (string, error) {
	out, err := run(root, nil, "git", args...)
	return strings.TrimSpace(string(out)), err
}

func fileDigest(path string) (Digest, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return Digest{}, 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return Digest{}, 0, err
	}
	return Digest{Algorithm: "sha256", Value: hex.EncodeToString(h.Sum(nil))}, n, nil
}

func sourceDigest(root string) (Digest, error) {
	listedBytes, err := run(root, nil, "git", "ls-files", "-z")
	if err != nil {
		return Digest{}, err
	}
	paths := strings.Split(string(listedBytes), "\x00")
	sort.Strings(paths)
	h := sha256.New()
	for _, rel := range paths {
		if rel == "" || strings.HasPrefix(rel, ".agent/") || strings.HasPrefix(rel, "dist/") {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(rel))
		info, statErr := os.Lstat(full)
		if statErr != nil {
			return Digest{}, fmt.Errorf("stat source %s: %w", rel, statErr)
		}
		if !info.Mode().IsRegular() {
			return Digest{}, fmt.Errorf("source digest rejects non-regular tracked path %s", rel)
		}
		data, readErr := os.ReadFile(full)
		if readErr != nil {
			return Digest{}, fmt.Errorf("hash source %s: %w", rel, readErr)
		}
		fmt.Fprintf(h, "%d:%s:%o:%d:", len(rel), rel, info.Mode().Perm()&0o111, len(data))
		h.Write(data)
	}
	return Digest{Algorithm: "sha256", Value: hex.EncodeToString(h.Sum(nil))}, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func copyFile(dst, src string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func cleanOutput(root, output string) (string, error) {
	abs, err := filepath.Abs(output)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if abs == rootAbs || abs == filepath.Dir(rootAbs) || abs == string(filepath.Separator) {
		return "", fmt.Errorf("BW9001: refusing unsafe output directory %s", abs)
	}
	if err := os.RemoveAll(abs); err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	return abs, nil
}

// Root resolves a repository root for contributor-only assurance commands.
func Root(start string) (string, error) { return repositoryRoot(start) }

// RecommendedVersion reads the single canonical candidate version source.
func RecommendedVersion(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "release", "VERSION"))
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("release/VERSION is empty")
	}
	return version, nil
}

// RunGoTest executes one bounded repository test selection.
func RunGoTest(root, pkg, pattern string) error {
	_, err := run(root, []string{"CGO_ENABLED=0"}, "go", "test", pkg, "-run", pattern, "-count=1")
	return err
}

// Clean removes only an explicitly named release output directory after the
// same broad-target safety validation used by Build.
func Clean(root, output string) error {
	abs, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if abs == rootAbs || abs == filepath.Dir(rootAbs) || abs == string(filepath.Separator) {
		return fmt.Errorf("refusing unsafe output directory %s", abs)
	}
	return os.RemoveAll(abs)
}
