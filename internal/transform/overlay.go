package transform

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

// overlayDoc is the standard Go command overlay manifest shape.
type overlayDoc struct {
	Replace map[string]string `json:"Replace"`
}

// WriteOverlay writes a Go command overlay manifest for a saved plan, mapping
// each original source file to its content-addressed transformed backing file.
// It returns the absolute overlay path and the number of mapped files. The plan
// must already be saved (its backing files present under the state directory).
func WriteOverlay(root string, plan *Plan) (string, int, error) {
	dir := planDir(root, plan.ID)
	doc := overlayDoc{Replace: map[string]string{}}
	for _, fp := range plan.Files {
		orig := filepath.Join(root, filepath.FromSlash(fp.Path))
		backing := filepath.Join(dir, "files", digestFileName(fp.TransformedDigest))
		doc.Replace[orig] = backing
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", 0, err
	}
	overlayPath := filepath.Join(dir, "overlay.json")
	if err := atomicWrite(overlayPath, append(data, '\n'), 0o644); err != nil {
		return "", 0, err
	}
	return overlayPath, len(doc.Replace), nil
}

// OverlayManifestDigest returns a stable digest of a plan's overlay mapping in
// workspace-relative terms, for recording in execution results.
func OverlayManifestDigest(plan *Plan) string {
	parts := []string{"overlay", plan.ID}
	for _, fp := range plan.Files {
		parts = append(parts, fp.Path, fp.TransformedDigest)
	}
	return hashParts(parts...)
}

// EnsureSavedOverlay saves a plan (if needed) and writes its overlay, returning
// the overlay path.
func EnsureSavedOverlay(root string, plan *Plan) (string, int, error) {
	if err := SavePlan(root, plan); err != nil {
		return "", 0, fmt.Errorf("save plan: %w", err)
	}
	return WriteOverlay(root, plan)
}
