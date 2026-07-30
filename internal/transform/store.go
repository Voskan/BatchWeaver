package transform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StateDir is the ignored directory under the workspace root where plans,
// overlays, and backups live.
const StateDir = ".batchweaver"

// planDir returns the cache directory for a plan.
func planDir(root, planID string) string {
	return filepath.Join(root, StateDir, "cache", "transform", planID)
}

// SavePlan persists a plan and its transformed file bytes under the workspace
// state directory using atomic writes. Transformed bytes are stored
// content-addressed; the deterministic plan.json stores only digests.
func SavePlan(root string, plan *Plan) error {
	dir := planDir(root, plan.ID)
	filesDir := filepath.Join(dir, "files")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return err
	}
	for _, fp := range plan.Files {
		name := digestFileName(fp.TransformedDigest)
		if err := atomicWrite(filepath.Join(filesDir, name), fp.transformed, 0o644); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(dir, "plan.json"), append(data, '\n'), 0o644)
}

// LoadPlan reads a plan and reattaches its transformed and original file bytes.
// It verifies that transformed bytes match their recorded digest.
func LoadPlan(root, planID string) (*Plan, error) {
	dir := planDir(root, planID)
	data, err := os.ReadFile(filepath.Join(dir, "plan.json"))
	if err != nil {
		return nil, fmt.Errorf("load plan %s: %w", planID, err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("decode plan %s: %w", planID, err)
	}
	for i := range plan.Files {
		fp := &plan.Files[i]
		b, err := os.ReadFile(filepath.Join(dir, "files", digestFileName(fp.TransformedDigest)))
		if err != nil {
			return nil, fmt.Errorf("plan %s: missing transformed file %s: %w", planID, fp.Path, err)
		}
		if hashBytes(b) != fp.TransformedDigest {
			return nil, fmt.Errorf("plan %s: transformed file %s is corrupt", planID, fp.Path)
		}
		fp.transformed = b
		if orig, oerr := os.ReadFile(filepath.Join(root, filepath.FromSlash(fp.Path))); oerr == nil {
			fp.original = orig
		}
	}
	return &plan, nil
}

// ListPlans returns the plan IDs currently cached under the workspace root.
func ListPlans(root string) ([]string, error) {
	base := filepath.Join(root, StateDir, "cache", "transform")
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// CleanPlans removes all cached transformation plans under the workspace root.
func CleanPlans(root string) error {
	base := filepath.Join(root, StateDir, "cache", "transform")
	if err := os.RemoveAll(base); err != nil {
		return err
	}
	return nil
}

// digestFileName maps a "sha256:hex" digest to a safe file name.
func digestFileName(digest string) string {
	return strings.TrimPrefix(digest, "sha256:") + ".go"
}

// atomicWrite writes data to path via a temporary sibling file and rename.
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".bw-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// ModuleRoot returns the workspace module root for a directory. It is exported
// for command wiring.
func ModuleRoot(dir string) (string, error) { return moduleRoot(dir) }
