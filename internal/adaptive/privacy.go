package adaptive

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// classLabelLen is the number of hex characters in an anonymized class label.
const classLabelLen = 10

// overflowClass is the label used when the distinct-class bound is exceeded, so
// cardinality stays bounded and no raw identifier can leak through an unbounded
// label space.
const overflowClass = "class_overflow"

// Classer maps raw partition or tenant identifiers to stable, non-reversible,
// bounded-cardinality class labels. It uses a keyed hash (HMAC-SHA256) with a
// per-instance random salt so labels are not correlatable across collection
// sessions and cannot be reversed to the raw identifier. The number of distinct
// labels is bounded; once the bound is reached, further identifiers collapse to
// overflowClass.
//
// Classer is safe for concurrent use.
type Classer struct {
	salt     []byte
	maxClass int

	mu    sync.Mutex
	known map[string]string // raw -> label
}

// NewClasser returns a Classer bounded to maxClass distinct labels. A random
// salt is generated so labels cannot be correlated across sessions. If maxClass
// is not positive, a conservative default is used.
func NewClasser(maxClass int) *Classer {
	if maxClass <= 0 {
		maxClass = 256
	}
	salt := make([]byte, 16)
	// crypto/rand.Read never returns a short read; ignoring the error keeps the
	// constructor total. A zero salt would still be private (values are hashed);
	// randomness only defends against cross-session correlation.
	_, _ = rand.Read(salt)
	return &Classer{salt: salt, maxClass: maxClass, known: make(map[string]string)}
}

// NewDeterministicClasser returns a Classer whose salt is derived from seed, so
// class labels are reproducible across runs. This is used for reproducible
// offline collection, replay, and tests. Production collection uses NewClasser
// with a random salt so labels cannot be correlated across sessions; labels
// remain non-reversible either way.
func NewDeterministicClasser(maxClass int, seed uint64) *Classer {
	if maxClass <= 0 {
		maxClass = 256
	}
	salt := make([]byte, 16)
	x := seed + 0x9e3779b97f4a7c15
	for i := 0; i < 16; i++ {
		x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
		salt[i] = byte(x >> 24)
	}
	return &Classer{salt: salt, maxClass: maxClass, known: make(map[string]string)}
}

// Class returns the anonymized class label for a raw identifier. The empty
// identifier maps to a stable "default" label without consuming the bound.
func (c *Classer) Class(raw string) string {
	if raw == "" {
		return "class_default"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if lbl, ok := c.known[raw]; ok {
		return lbl
	}
	if len(c.known) >= c.maxClass {
		return overflowClass
	}
	mac := hmac.New(sha256.New, c.salt)
	_, _ = mac.Write([]byte(raw))
	lbl := "class_" + hex.EncodeToString(mac.Sum(nil))[:classLabelLen]
	c.known[raw] = lbl
	return lbl
}

// DistinctCount returns the number of distinct raw identifiers seen (bounded by
// the configured maximum plus a possible overflow bucket).
func (c *Classer) DistinctCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.known)
}
