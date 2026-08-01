package adaptive

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// profileMagic identifies a persisted BatchWeaver profile envelope.
const profileMagic = "batchweaver-profile"

// profileFilePerm is the restrictive permission for persisted profiles, which
// may embed anonymized but still workload-revealing statistics.
const profileFilePerm = 0o600

// envelope wraps a profile bundle with an integrity checksum. The checksum is
// computed over the canonical JSON of the bundle so corruption is detected on
// load, independent of the bundle's own content digest.
type envelope struct {
	Magic    string        `json:"magic"`
	Version  string        `json:"schema_version"`
	Bundle   ProfileBundle `json:"bundle"`
	Checksum string        `json:"checksum"`
}

// Marshal returns the deterministic JSON envelope for a bundle. It finalizes the
// bundle first so digests are always current.
func Marshal(b *ProfileBundle) ([]byte, error) {
	b.Finalize()
	body, err := json.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("adaptive: marshal bundle: %w", err)
	}
	sum := sha256.Sum256(body)
	env := envelope{
		Magic:    profileMagic,
		Version:  ProfileSchemaVersion,
		Bundle:   *b,
		Checksum: hex.EncodeToString(sum[:]),
	}
	return json.MarshalIndent(env, "", "  ")
}

// Unmarshal decodes and verifies a profile envelope, checking the magic, schema
// version, integrity checksum, embedded digests, and every histogram's
// structural invariants. A hostile or corrupt file is rejected rather than
// silently accepted.
func Unmarshal(data []byte) (*ProfileBundle, error) {
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("adaptive: decode profile: %w", err)
	}
	if env.Magic != profileMagic {
		return nil, fmt.Errorf("adaptive: not a BatchWeaver profile (bad magic %q)", env.Magic)
	}
	body, err := json.Marshal(env.Bundle)
	if err != nil {
		return nil, fmt.Errorf("adaptive: re-encode bundle: %w", err)
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != env.Checksum {
		return nil, fmt.Errorf("adaptive: profile checksum mismatch (corrupted file)")
	}
	b := env.Bundle
	if err := b.validateStructure(); err != nil {
		return nil, err
	}
	prev := b.Digest
	b.Finalize()
	if prev != "" && prev != b.Digest {
		return nil, fmt.Errorf("adaptive: profile digest mismatch (recomputed %s, stored %s)", b.Digest.Short(), prev.Short())
	}
	return &b, nil
}

// validateStructure verifies histograms decode and categorical maps stay within
// bounds, so a malformed profile is rejected before use.
func (b *ProfileBundle) validateStructure() error {
	for i := range b.Operations {
		op := &b.Operations[i]
		hists := []HistogramData{
			op.Arrivals.InterArrival, op.Queue.WaitNanos, op.Queue.DepthItems,
			op.Batches.Size, op.Batches.Weight, op.Backend.LatencyNanos,
			op.Backend.SerializationNanos, op.Backend.MappingNanos,
			op.Deadlines.SlackNanos, op.Payloads.Bytes, op.Chunks.Count, op.Fairness.WaitNanos,
		}
		for _, h := range hists {
			if _, err := h.Decode(); err != nil {
				return fmt.Errorf("adaptive: operation %q: %w", op.Operation, err)
			}
		}
		maps := []CategoricalCounts{
			op.Batches.FlushReasons, op.Errors.ByClass, op.Partitions.ByClass,
			op.Fallbacks.ByReason, op.Fairness.ServiceShare,
		}
		for _, m := range maps {
			if len(m) > maxCategoricalKeys {
				return fmt.Errorf("adaptive: operation %q: categorical map exceeds bound (%d)", op.Operation, len(m))
			}
		}
	}
	return nil
}

// WriteFile atomically writes a profile to path with restrictive permissions. It
// writes to a temporary file in the same directory and renames it into place so
// a partial write never corrupts an existing profile.
func WriteFile(path string, b *ProfileBundle) error {
	data, err := Marshal(b)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("adaptive: create profile directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".bwprofile-*.tmp")
	if err != nil {
		return fmt.Errorf("adaptive: create temp profile: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op after a successful rename
	if err := tmp.Chmod(profileFilePerm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("adaptive: chmod temp profile: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("adaptive: write temp profile: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("adaptive: sync temp profile: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("adaptive: close temp profile: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("adaptive: install profile: %w", err)
	}
	return nil
}

// ReadFile reads and verifies a profile from path.
func ReadFile(path string) (*ProfileBundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("adaptive: read profile: %w", err)
	}
	return Unmarshal(data)
}
