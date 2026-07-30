package transform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BackupManifestSchema versions the backup manifest.
const BackupManifestSchema = "batchweaver.backup/v1alpha1"

// BackupManifest records everything needed to revert a materialization.
type BackupManifest struct {
	SchemaVersion     string               `json:"schema_version"`
	MaterializationID string               `json:"materialization_id"`
	PlanID            string               `json:"plan_id"`
	Workspace         string               `json:"workspace"`
	Tool              string               `json:"tool"`
	State             MaterializationState `json:"state"`
	Files             []BackupFile         `json:"files"`
}

// BackupFile is one file's backup entry.
type BackupFile struct {
	Path              string `json:"path"`
	OriginalDigest    string `json:"original_digest"`
	TransformedDigest string `json:"transformed_digest"`
	BackupObject      string `json:"backup_object"`
	Committed         bool   `json:"committed"`
}

// MaterializeResult summarizes a materialization.
type MaterializeResult struct {
	MaterializationID string
	FilesChanged      int
	ManifestPath      string
}

// Materialize writes a plan's transformed files into the working tree after
// verifying every source precondition, taking a full backup first. It is atomic
// per file and refuses to proceed if any source file changed since planning.
func Materialize(root, tool string, plan *Plan) (*MaterializeResult, error) {
	// Precondition: current source digests must match the plan.
	for _, fp := range plan.Files {
		cur, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fp.Path)))
		if err != nil {
			return nil, fmt.Errorf("materialize: read %s: %w", fp.Path, err)
		}
		if hashBytes(cur) != fp.OriginalDigest {
			return nil, fmt.Errorf("BW3701: materialization precondition changed for %s", fp.Path)
		}
	}

	matID := shortID("bwmat", plan.ID, plan.Digest)
	backupDir := filepath.Join(root, StateDir, "backups", matID)
	if err := os.MkdirAll(filepath.Join(backupDir, "files"), 0o755); err != nil {
		return nil, err
	}

	manifest := &BackupManifest{
		SchemaVersion: BackupManifestSchema, MaterializationID: matID,
		PlanID: plan.ID, Workspace: ".", Tool: tool, State: MatWriting,
	}
	for _, fp := range plan.Files {
		manifest.Files = append(manifest.Files, BackupFile{
			Path: fp.Path, OriginalDigest: fp.OriginalDigest,
			TransformedDigest: fp.TransformedDigest,
			BackupObject:      digestFileName(fp.OriginalDigest),
		})
	}
	manifestPath := filepath.Join(backupDir, "manifest.json")
	if err := writeManifest(manifestPath, manifest); err != nil {
		return nil, err
	}

	// Back up originals, then commit transformed files atomically.
	for i, fp := range plan.Files {
		if err := atomicWrite(filepath.Join(backupDir, "files", manifest.Files[i].BackupObject), fp.original, 0o644); err != nil {
			return nil, err
		}
		target := filepath.Join(root, filepath.FromSlash(fp.Path))
		if err := atomicWrite(target, fp.transformed, 0o644); err != nil {
			return nil, fmt.Errorf("materialize: write %s: %w", fp.Path, err)
		}
		manifest.Files[i].Committed = true
		if err := writeManifest(manifestPath, manifest); err != nil {
			return nil, err
		}
	}
	manifest.State = MatCommitted
	if err := writeManifest(manifestPath, manifest); err != nil {
		return nil, err
	}
	return &MaterializeResult{MaterializationID: matID, FilesChanged: len(plan.Files), ManifestPath: manifestPath}, nil
}

// RevertResult summarizes a revert.
type RevertResult struct {
	FilesRestored int
	Conflicts     []string
}

// Revert restores the original files recorded in a materialization backup. It
// refuses to overwrite files that were edited after materialization (a
// transformed-digest mismatch) and reports them as conflicts.
func Revert(root, matID string) (*RevertResult, error) {
	backupDir := filepath.Join(root, StateDir, "backups", matID)
	manifest, err := readManifest(filepath.Join(backupDir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	res := &RevertResult{}
	for _, bf := range manifest.Files {
		if !bf.Committed {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(bf.Path))
		cur, err := os.ReadFile(target)
		if err != nil {
			res.Conflicts = append(res.Conflicts, bf.Path)
			continue
		}
		if hashBytes(cur) != bf.TransformedDigest {
			// The file was edited after materialization; do not overwrite.
			res.Conflicts = append(res.Conflicts, bf.Path)
			continue
		}
		orig, err := os.ReadFile(filepath.Join(backupDir, "files", bf.BackupObject))
		if err != nil {
			res.Conflicts = append(res.Conflicts, bf.Path)
			continue
		}
		if hashBytes(orig) != bf.OriginalDigest {
			res.Conflicts = append(res.Conflicts, bf.Path)
			continue
		}
		if err := atomicWrite(target, orig, 0o644); err != nil {
			return nil, err
		}
		res.FilesRestored++
	}
	if len(res.Conflicts) > 0 {
		manifest.State = MatRevertConflict
	} else {
		manifest.State = MatReverted
	}
	_ = writeManifest(filepath.Join(backupDir, "manifest.json"), manifest)
	return res, nil
}

// RecoverStatus reports the recoverable state of an interrupted materialization.
type RecoverStatus struct {
	MaterializationID string
	State             MaterializationState
	CommittedFiles    int
	TotalFiles        int
}

// Recover inspects incomplete materializations under the workspace and reports
// their state. It is idempotent and never mutates source files on its own.
func Recover(root string) ([]RecoverStatus, error) {
	base := filepath.Join(root, StateDir, "backups")
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []RecoverStatus
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := readManifest(filepath.Join(base, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		committed := 0
		for _, f := range m.Files {
			if f.Committed {
				committed++
			}
		}
		state := m.State
		if state == MatWriting && committed < len(m.Files) {
			state = MatRecoveryReq
		}
		out = append(out, RecoverStatus{
			MaterializationID: m.MaterializationID, State: state,
			CommittedFiles: committed, TotalFiles: len(m.Files),
		})
	}
	return out, nil
}

func writeManifest(path string, m *BackupManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(data, '\n'), 0o644)
}

func readManifest(path string) (*BackupManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m BackupManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decode backup manifest: %w", err)
	}
	return &m, nil
}
