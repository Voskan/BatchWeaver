package adaptive

import (
	"fmt"
	"sort"
)

// HistogramData is the deterministic, JSON-friendly wire form of a Histogram.
// Buckets are stored as parallel ascending-index/count arrays so encoding is
// canonical and byte-stable for identical inputs.
type HistogramData struct {
	Accuracy     float64  `json:"accuracy"`
	Count        uint64   `json:"count"`
	Sum          float64  `json:"sum"`
	Min          float64  `json:"min"`
	Max          float64  `json:"max"`
	ZeroCount    uint64   `json:"zero_count"`
	Overflow     bool     `json:"overflow"`
	BucketIndex  []int    `json:"bucket_index"`
	BucketCounts []uint64 `json:"bucket_counts"`
}

// Encode returns the deterministic wire form of the histogram.
func (h *Histogram) Encode() HistogramData {
	idx := h.sortedIndices()
	counts := make([]uint64, len(idx))
	for i, k := range idx {
		counts[i] = h.buckets[k]
	}
	return HistogramData{
		Accuracy:     h.Accuracy(),
		Count:        h.count,
		Sum:          h.sum,
		Min:          h.min,
		Max:          h.max,
		ZeroCount:    h.zeroCount,
		Overflow:     h.overflow,
		BucketIndex:  idx,
		BucketCounts: counts,
	}
}

// Decode rebuilds a Histogram from its wire form, validating structural
// invariants so a corrupted or hostile profile cannot produce an inconsistent
// sketch.
func (d HistogramData) Decode() (*Histogram, error) {
	if len(d.BucketIndex) != len(d.BucketCounts) {
		return nil, fmt.Errorf("adaptive: histogram bucket arrays have mismatched lengths (%d vs %d)", len(d.BucketIndex), len(d.BucketCounts))
	}
	if len(d.BucketIndex) > histogramMaxBuckets {
		return nil, fmt.Errorf("adaptive: histogram exceeds bucket bound (%d > %d)", len(d.BucketIndex), histogramMaxBuckets)
	}
	if !sort.IntsAreSorted(d.BucketIndex) {
		return nil, fmt.Errorf("adaptive: histogram bucket indices are not sorted")
	}
	h := NewHistogram(d.Accuracy)
	var bucketTotal uint64
	for i, k := range d.BucketIndex {
		if _, dup := h.buckets[k]; dup {
			return nil, fmt.Errorf("adaptive: histogram has duplicate bucket index %d", k)
		}
		h.buckets[k] = d.BucketCounts[i]
		bucketTotal += d.BucketCounts[i]
	}
	if bucketTotal+d.ZeroCount != d.Count {
		return nil, fmt.Errorf("adaptive: histogram count %d does not match bucket totals %d", d.Count, bucketTotal+d.ZeroCount)
	}
	h.zeroCount = d.ZeroCount
	h.count = d.Count
	h.sum = d.Sum
	h.min = d.Min
	h.max = d.Max
	h.overflow = d.Overflow
	return h, nil
}

// digestParts contributes the histogram's canonical fields to a digest.
func (d HistogramData) digestParts() []string {
	parts := []string{
		"hist",
		fmt.Sprintf("acc=%g", d.Accuracy),
		fmt.Sprintf("n=%d", d.Count),
		fmt.Sprintf("z=%d", d.ZeroCount),
	}
	for i, k := range d.BucketIndex {
		parts = append(parts, fmt.Sprintf("%d:%d", k, d.BucketCounts[i]))
	}
	return parts
}
