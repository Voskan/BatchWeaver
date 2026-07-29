package runtime

import "sync"

// memoRecord is one scope-memoized result.
type memoRecord[K any, V any] struct {
	part  string
	key   K
	value V
	found bool
	bytes int
}

// memoTable is a bounded, scope-local memoization store keyed by partition and
// key. It uses insertion-order eviction and is safe for concurrent use by the
// caller goroutines running inside a scope.
type memoTable[K any, V any] struct {
	mu     sync.Mutex
	keys   KeyStrategy[K]
	byHash map[uint64][]*memoRecord[K, V]
	order  []*memoRecord[K, V]
	bytes  int64
}

// hashOf combines the partition and key hashes for bucketing.
func (t *memoTable[K, V]) hashOf(part Partition, key K) uint64 {
	return part.hash ^ t.keys.Hash(key)
}

// get returns the memoized result for (part, key), confirming identity with the
// key strategy so hash collisions never return the wrong value.
func (t *memoTable[K, V]) get(part Partition, key K) (result[V], bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, r := range t.byHash[t.hashOf(part, key)] {
		if r.part == part.encoded && t.keys.Equal(r.key, key) {
			return result[V]{value: r.value, found: r.found}, true
		}
	}
	return result[V]{}, false
}

// put stores a result, evicting the oldest entries when a limit is exceeded. It
// does not overwrite an existing entry for the same identity.
func (t *memoTable[K, V]) put(part Partition, key K, value V, found bool, cfg bindingConfig[K, V]) {
	t.mu.Lock()
	defer t.mu.Unlock()
	h := t.hashOf(part, key)
	for _, r := range t.byHash[h] {
		if r.part == part.encoded && t.keys.Equal(r.key, key) {
			return
		}
	}
	rec := &memoRecord[K, V]{part: part.encoded, key: key, value: value, found: found, bytes: t.keys.EstimateBytes(key)}
	t.byHash[h] = append(t.byHash[h], rec)
	t.order = append(t.order, rec)
	t.bytes += int64(rec.bytes)

	for len(t.order) > cfg.memoMaxItems || (cfg.memoMaxBytes > 0 && t.bytes > cfg.memoMaxBytes) {
		if len(t.order) == 0 {
			break
		}
		t.evictOldest()
	}
}

// evictOldest removes the oldest memoized record.
func (t *memoTable[K, V]) evictOldest() {
	old := t.order[0]
	t.order = t.order[1:]
	t.bytes -= int64(old.bytes)
	bucketKey := t.bucketKeyFor(old)
	bucket := t.byHash[bucketKey]
	for i, r := range bucket {
		if r == old {
			bucket = append(bucket[:i], bucket[i+1:]...)
			break
		}
	}
	if len(bucket) == 0 {
		delete(t.byHash, bucketKey)
	} else {
		t.byHash[bucketKey] = bucket
	}
}

// bucketKeyFor recomputes the bucket key for a stored record.
func (t *memoTable[K, V]) bucketKeyFor(r *memoRecord[K, V]) uint64 {
	// r.part is the encoded partition string; combine with the key hash the same
	// way get/put do. We reconstruct the partition hash from the stored token by
	// hashing the key and folding the partition token hash.
	return partitionEncodedHash(r.part) ^ t.keys.Hash(r.key)
}

// partitionEncodedHash returns the hash used by newPartition for an encoded form.
func partitionEncodedHash(encoded string) uint64 {
	return newPartition(encoded).hash
}
