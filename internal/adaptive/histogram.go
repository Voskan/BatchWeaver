package adaptive

import (
	"fmt"
	"math"
	"sort"
)

// Histogram is a bounded, mergeable, deterministic distribution sketch with a
// configurable relative accuracy. It uses logarithmic bucketing (a simplified
// DDSketch): a positive value v maps to bucket index ceil(log(v)/log(gamma)),
// which guarantees a relative error of at most (gamma-1) on any quantile. Bucket
// count is bounded; values beyond the representable range clamp to the extreme
// buckets and set an overflow flag so callers never mistake a clamp for exact
// data.
//
// A Histogram is not safe for concurrent use; the collector owns one per metric
// and serializes access. Serialization is deterministic (buckets are emitted in
// ascending index order), so identical inputs always yield identical digests.
type Histogram struct {
	// gamma is the bucket growth factor; relative error is bounded by gamma-1.
	gamma float64
	// logGamma caches ln(gamma).
	logGamma float64
	// maxBuckets bounds the number of distinct populated buckets.
	maxBuckets int
	// buckets maps a bucket index to its count. Index 0 is reserved conceptually;
	// real indices are non-zero. The zero/near-zero bucket is tracked separately.
	buckets map[int]uint64
	// zeroCount counts values at or below the zero threshold.
	zeroCount uint64
	// count, sum, min, max summarize the raw stream.
	count    uint64
	sum      float64
	min      float64
	max      float64
	overflow bool
}

// DefaultHistogramAccuracy is the default relative accuracy (2%).
const DefaultHistogramAccuracy = 0.02

// histogramMaxBuckets bounds populated buckets to keep memory and serialization
// size bounded even under adversarial inputs.
const histogramMaxBuckets = 2048

// zeroThreshold is the smallest magnitude tracked outside the zero bucket.
const zeroThreshold = 1e-9

// NewHistogram returns a Histogram with the given relative accuracy in (0,1). An
// accuracy outside that range is clamped to DefaultHistogramAccuracy.
func NewHistogram(accuracy float64) *Histogram {
	if !(accuracy > 0 && accuracy < 1) {
		accuracy = DefaultHistogramAccuracy
	}
	gamma := 1 + accuracy
	return &Histogram{
		gamma:      gamma,
		logGamma:   math.Log(gamma),
		maxBuckets: histogramMaxBuckets,
		buckets:    make(map[int]uint64),
	}
}

// index returns the bucket index for a strictly positive value.
func (h *Histogram) index(v float64) int {
	return int(math.Ceil(math.Log(v) / h.logGamma))
}

// value returns a representative value for a bucket index (the geometric upper
// edge), used for quantile and mean-from-bucket estimation.
func (h *Histogram) value(idx int) float64 {
	return math.Pow(h.gamma, float64(idx))
}

// Observe records a single non-negative sample. Negative values are treated as
// zero. If adding a new bucket would exceed the bound, the value is merged into
// the nearest existing extreme bucket and the overflow flag is set.
func (h *Histogram) Observe(v float64) {
	h.ObserveN(v, 1)
}

// ObserveN records n copies of a non-negative sample, used when scaling sampled
// observations back to an estimated population.
func (h *Histogram) ObserveN(v float64, n uint64) {
	if n == 0 {
		return
	}
	if v < 0 {
		v = 0
	}
	if h.count == 0 || v < h.min {
		h.min = v
	}
	if h.count == 0 || v > h.max {
		h.max = v
	}
	h.count += n
	h.sum += v * float64(n)
	if v <= zeroThreshold {
		h.zeroCount += n
		return
	}
	idx := h.index(v)
	if _, ok := h.buckets[idx]; !ok && len(h.buckets) >= h.maxBuckets {
		h.overflow = true
		idx = h.nearestExisting(idx)
	}
	h.buckets[idx] += n
}

// nearestExisting returns the populated bucket index closest to idx. It is only
// called when the bucket cap is reached, which is rare in practice.
func (h *Histogram) nearestExisting(idx int) int {
	best := idx
	bestDist := int(^uint(0) >> 1)
	for k := range h.buckets {
		d := k - idx
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			bestDist, best = d, k
		}
	}
	return best
}

// Count returns the number of observed samples.
func (h *Histogram) Count() uint64 { return h.count }

// Sum returns the sum of observed values.
func (h *Histogram) Sum() float64 { return h.sum }

// Min returns the smallest observed value, or 0 when empty.
func (h *Histogram) Min() float64 { return h.min }

// Max returns the largest observed value, or 0 when empty.
func (h *Histogram) Max() float64 { return h.max }

// Overflowed reports whether any value was clamped into an extreme bucket.
func (h *Histogram) Overflowed() bool { return h.overflow }

// Mean returns the arithmetic mean, or 0 when empty.
func (h *Histogram) Mean() float64 {
	if h.count == 0 {
		return 0
	}
	return h.sum / float64(h.count)
}

// Quantile returns an estimate of the q-quantile for q in [0,1]. The estimate
// has a relative error of at most gamma-1. It returns 0 for an empty histogram.
func (h *Histogram) Quantile(q float64) float64 {
	if h.count == 0 {
		return 0
	}
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}
	// Rank of the target sample (1-based).
	rank := uint64(math.Ceil(q * float64(h.count)))
	if rank == 0 {
		rank = 1
	}
	var cum uint64
	cum += h.zeroCount
	if rank <= cum {
		return 0
	}
	for _, idx := range h.sortedIndices() {
		cum += h.buckets[idx]
		if rank <= cum {
			est := h.value(idx)
			// Clamp to observed range so estimates never exceed reality.
			if est > h.max {
				est = h.max
			}
			if est < h.min {
				est = h.min
			}
			return est
		}
	}
	return h.max
}

// CDF returns the fraction of observed samples less than or equal to v, in
// [0,1]. It returns 0 for an empty histogram. The estimate inherits the
// histogram's relative accuracy.
func (h *Histogram) CDF(v float64) float64 {
	if h.count == 0 {
		return 0
	}
	var below uint64
	if v >= 0 {
		below += h.zeroCount
	}
	if v > zeroThreshold {
		target := h.index(v)
		for _, idx := range h.sortedIndices() {
			if idx <= target {
				below += h.buckets[idx]
			}
		}
	}
	return float64(below) / float64(h.count)
}

// sortedIndices returns populated bucket indices in ascending order for
// deterministic iteration and serialization.
func (h *Histogram) sortedIndices() []int {
	idx := make([]int, 0, len(h.buckets))
	for k := range h.buckets {
		idx = append(idx, k)
	}
	sort.Ints(idx)
	return idx
}

// Merge folds other into h. Both must share the same accuracy; merging
// histograms with differing gamma is rejected because it would silently corrupt
// quantile estimates.
func (h *Histogram) Merge(other *Histogram) error {
	if other == nil {
		return nil
	}
	if math.Abs(h.gamma-other.gamma) > 1e-12 {
		return fmt.Errorf("adaptive: cannot merge histograms with different accuracy (%.6f vs %.6f)", h.gamma-1, other.gamma-1)
	}
	if other.count == 0 {
		return nil
	}
	if h.count == 0 || other.min < h.min {
		h.min = other.min
	}
	if h.count == 0 || other.max > h.max {
		h.max = other.max
	}
	h.count += other.count
	h.sum += other.sum
	h.zeroCount += other.zeroCount
	h.overflow = h.overflow || other.overflow
	for _, idx := range other.sortedIndices() {
		if _, ok := h.buckets[idx]; !ok && len(h.buckets) >= h.maxBuckets {
			h.overflow = true
			h.buckets[h.nearestExisting(idx)] += other.buckets[idx]
			continue
		}
		h.buckets[idx] += other.buckets[idx]
	}
	return nil
}

// Clone returns a deep copy of the histogram.
func (h *Histogram) Clone() *Histogram {
	c := &Histogram{
		gamma:      h.gamma,
		logGamma:   h.logGamma,
		maxBuckets: h.maxBuckets,
		buckets:    make(map[int]uint64, len(h.buckets)),
		zeroCount:  h.zeroCount,
		count:      h.count,
		sum:        h.sum,
		min:        h.min,
		max:        h.max,
		overflow:   h.overflow,
	}
	for k, v := range h.buckets {
		c.buckets[k] = v
	}
	return c
}

// Accuracy returns the configured relative accuracy (gamma-1).
func (h *Histogram) Accuracy() float64 { return h.gamma - 1 }
