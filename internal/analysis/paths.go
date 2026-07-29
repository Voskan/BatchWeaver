package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path/filepath"
	"strings"
)

// digestLen is the number of hex characters shown for a short digest. The full
// SHA-256 is available internally; short IDs are collision-checked by the
// indexes that mint them.
const digestLen = 16

// shortDigest returns a stable short hex digest over the null-separated parts.
func shortDigest(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		_, _ = io.WriteString(h, p)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:digestLen]
}

// pathContext holds the roots used to portablize absolute file paths.
type pathContext struct {
	root       string // workspace/module root; repository-relative paths hang off this
	goroot     string
	gomodcache string
}

// portable converts an absolute file path into a stable, machine-independent
// form: repository-relative for files under the workspace root, "std://" for the
// standard library, "mod://" for module-cache dependencies, and the base name as
// a last resort. All separators are forward slashes.
func (pc pathContext) portable(abs string) string {
	if abs == "" {
		return ""
	}
	clean := filepath.Clean(abs)
	if pc.root != "" {
		if rel, err := filepath.Rel(pc.root, clean); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	if pc.goroot != "" {
		if rel, err := filepath.Rel(filepath.Join(pc.goroot, "src"), clean); err == nil && !strings.HasPrefix(rel, "..") {
			return "std://" + filepath.ToSlash(rel)
		}
	}
	if pc.gomodcache != "" {
		if rel, err := filepath.Rel(pc.gomodcache, clean); err == nil && !strings.HasPrefix(rel, "..") {
			return "mod://" + filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(filepath.Base(clean))
}
