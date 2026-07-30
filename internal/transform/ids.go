package transform

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sort"
)

const shortLen = 12

// hashParts returns a stable full SHA-256 hex digest over null-separated parts.
func hashParts(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		_, _ = io.WriteString(h, p)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// hashBytes returns a "sha256:"-prefixed digest of b.
func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// shortID returns "<prefix>_<12 hex>".
func shortID(prefix string, parts ...string) string {
	return prefix + "_" + hashParts(parts...)[:shortLen]
}

// sortedCopy returns a sorted copy of s.
func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}
