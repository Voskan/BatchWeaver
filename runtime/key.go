package runtime

import (
	"bytes"
	"hash/maphash"
	"unsafe"
)

// keySeed is a process-wide seed for the built-in key strategies. Hash values
// only accelerate bucket lookup; Equal always confirms identity, so the seed
// never affects correctness.
var keySeed = maphash.MakeSeed()

// KeyStrategy defines how the runtime clones, hashes, compares, and sizes keys
// of type K. It exists because K need not be comparable and because keys may
// contain mutable data that must be owned by the runtime.
//
// Implementations must be safe for concurrent use and must not panic for valid
// inputs. Hash may collide; Equal is authoritative for identity.
type KeyStrategy[K any] interface {
	// Clone returns owned key material safe to retain after the caller returns.
	// For immutable keys it may return the input unchanged.
	Clone(K) K
	// Hash returns a hash used only to accelerate lookup; collisions are allowed.
	Hash(K) uint64
	// Equal reports whether two keys are identical.
	Equal(K, K) bool
	// EstimateBytes returns a non-negative estimate of the key's size in bytes,
	// used for queue-byte accounting.
	EstimateBytes(K) int
}

// comparableStrategy implements KeyStrategy for comparable keys.
type comparableStrategy[K comparable] struct{}

// ComparableKeys returns a KeyStrategy for any comparable key type. It hashes
// with hash/maphash (no reflection in the hot path) and compares with ==.
func ComparableKeys[K comparable]() KeyStrategy[K] { return comparableStrategy[K]{} }

func (comparableStrategy[K]) Clone(k K) K       { return k }
func (comparableStrategy[K]) Hash(k K) uint64   { return maphash.Comparable(keySeed, k) }
func (comparableStrategy[K]) Equal(a, b K) bool { return a == b }
func (comparableStrategy[K]) EstimateBytes(k K) int {
	return int(unsafe.Sizeof(k))
}

// stringStrategy implements KeyStrategy for string keys.
type stringStrategy struct{}

// StringKeys returns a KeyStrategy specialized for string keys.
func StringKeys() KeyStrategy[string] { return stringStrategy{} }

func (stringStrategy) Clone(s string) string      { return s }
func (stringStrategy) Hash(s string) uint64       { return maphash.String(keySeed, s) }
func (stringStrategy) Equal(a, b string) bool     { return a == b }
func (stringStrategy) EstimateBytes(s string) int { return len(s) }

// bytesStrategy implements KeyStrategy for []byte keys with defensive cloning.
type bytesStrategy struct{}

// BytesKeys returns a KeyStrategy for []byte keys. Clone copies the slice so the
// runtime never retains caller-owned backing arrays.
func BytesKeys() KeyStrategy[[]byte] { return bytesStrategy{} }

func (bytesStrategy) Clone(b []byte) []byte {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
func (bytesStrategy) Hash(b []byte) uint64       { return maphash.Bytes(keySeed, b) }
func (bytesStrategy) Equal(a, b []byte) bool     { return bytes.Equal(a, b) }
func (bytesStrategy) EstimateBytes(b []byte) int { return len(b) }
