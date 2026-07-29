package proof

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sort"
	"strings"
)

// digestLen is the number of hex characters shown in short IDs. Full digests are
// retained where identity must be strong (invalidation).
const digestLen = 12

// hashParts returns a stable full SHA-256 hex digest over null-separated parts.
func hashParts(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		_, _ = io.WriteString(h, p)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// shortID returns a prefixed short digest, e.g. "bwproof_1a2b3c4d5e6f".
func shortID(prefix string, parts ...string) string {
	return prefix + "_" + hashParts(parts...)[:digestLen]
}

// candidateDigest derives the digest over the analyzed facts of a candidate. It
// changes when the candidate's structure, resolved targets, effect surface, or
// operation contract change, which is exactly when a prior proof must be
// reconsidered. It intentionally excludes host paths beyond the portable
// locations already normalized by the analysis package.
func candidateDigest(f candidateFacts) string {
	parts := []string{
		"candidate", f.id, f.operation, f.structure,
		"compat", f.compatibility,
		"disabled", boolStr(f.disabled),
		"kind", f.kind,
		"contract", f.contractDigest,
	}
	// Targets and effects are order-normalized to keep the digest stable.
	parts = append(parts, "dispatch", strings.Join(sortedCopy(f.dispatch), ","))
	parts = append(parts, "effects", strings.Join(sortedCopy(f.effects), ","))
	parts = append(parts, "complete", boolStr(f.effectsComplete))
	parts = append(parts, "receiver", strings.Join(sortedCopy(f.receivers), ","))
	parts = append(parts, "context", strings.Join(sortedCopy(f.contexts), ","))
	return hashParts(parts...)
}

// proofID derives the stable proof certificate identity. Per the model, it
// derives from canonical candidate identity, the ordered obligation outcomes,
// strategy outcomes, contract digest, assumption digest, and both schema
// versions. Volatile fields such as timestamps never contribute.
func proofID(candDigest string, obligations []ObligationResult, strategies []StrategyEligibility, contractDigest, assumptionDigest string) string {
	parts := []string{
		"proof", SchemaVersion, StrategyRegistryVersion,
		"candidate", candDigest,
		"contract", contractDigest,
		"assumptions", assumptionDigest,
	}
	for _, o := range obligations {
		parts = append(parts, o.ID, string(o.Status))
	}
	for _, s := range strategies {
		parts = append(parts, s.Strategy, string(s.Status))
	}
	return shortID("bwproof", parts...)
}

// boolStr renders a stable string for a bool.
func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// sortedCopy returns a sorted copy of s.
func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// candidateFacts is the normalized fact bundle used to derive digests. It is a
// projection of analysis facts with no pointer identity.
type candidateFacts struct {
	id              string
	operation       string
	structure       string
	compatibility   string
	disabled        bool
	kind            string
	contractDigest  string
	dispatch        []string
	effects         []string
	effectsComplete bool
	receivers       []string
	contexts        []string
}
