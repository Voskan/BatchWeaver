package adaptive

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// Schema and component versions. They are independent of the analysis, proof,
// transform, bridge, and adapter schema versions and are bumped when the on-disk
// or decision-affecting format of the corresponding component changes. Consumers
// use them for compatibility checks and cache invalidation.
const (
	// ProfileSchemaVersion identifies the workload profile bundle schema.
	ProfileSchemaVersion = "batchweaver.profile/v1alpha1"
	// CostModelVersion identifies the cost model formulation.
	CostModelVersion = "batchweaver.cost/v1alpha1"
	// ControllerVersion identifies the adaptive controller decision logic.
	ControllerVersion = "batchweaver.controller/v1alpha1"
	// WaveSchemaVersion identifies the multi-operation wave DAG schema.
	WaveSchemaVersion = "batchweaver.wave/v1alpha1"
	// ReplaySchemaVersion identifies the deterministic replay input/output schema.
	ReplaySchemaVersion = "batchweaver.replay/v1alpha1"
)

// Digest is a full lowercase hex SHA-256 digest used for content addressing,
// compatibility checks, and invalidation.
type Digest string

// digestLen is the number of hex characters shown in short, human-facing IDs.
const digestLen = 12

// hashParts returns a stable full SHA-256 hex digest over null-separated parts.
// The null separator makes the concatenation unambiguous so that distinct part
// lists never collide.
func hashParts(parts ...string) Digest {
	h := sha256.New()
	for _, p := range parts {
		_, _ = io.WriteString(h, p)
		_, _ = h.Write([]byte{0})
	}
	return Digest(hex.EncodeToString(h.Sum(nil)))
}

// shortID returns a prefixed short digest, for example "bwdec_1a2b3c4d5e6f".
func shortID(prefix string, parts ...string) string {
	return prefix + "_" + string(hashParts(parts...))[:digestLen]
}

// Short returns the leading digestLen hex characters of the digest.
func (d Digest) Short() string {
	if len(d) <= digestLen {
		return string(d)
	}
	return string(d)[:digestLen]
}
