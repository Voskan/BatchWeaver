package adaptive

import (
	"math"
	"testing"
)

func TestHistogramQuantileWithinAccuracy(t *testing.T) {
	h := NewHistogram(0.01)
	for i := 1; i <= 10000; i++ {
		h.Observe(float64(i))
	}
	if got := h.Count(); got != 10000 {
		t.Fatalf("count = %d, want 10000", got)
	}
	// p50 should be near 5000 within the relative accuracy.
	p50 := h.Quantile(0.50)
	if rel := math.Abs(p50-5000) / 5000; rel > 0.03 {
		t.Errorf("p50 = %.1f, relative error %.4f too high", p50, rel)
	}
	p95 := h.Quantile(0.95)
	if rel := math.Abs(p95-9500) / 9500; rel > 0.03 {
		t.Errorf("p95 = %.1f, relative error %.4f too high", p95, rel)
	}
	if h.Max() != 10000 || h.Min() != 1 {
		t.Errorf("min/max = %.0f/%.0f, want 1/10000", h.Min(), h.Max())
	}
}

func TestHistogramZeroAndCDF(t *testing.T) {
	h := NewHistogram(0.02)
	h.ObserveN(0, 100)
	h.ObserveN(1000, 100)
	if got := h.CDF(0); math.Abs(got-0.5) > 0.01 {
		t.Errorf("CDF(0) = %.3f, want ~0.5", got)
	}
	if got := h.CDF(2000); got < 0.99 {
		t.Errorf("CDF(2000) = %.3f, want ~1", got)
	}
}

func TestHistogramMergeAndEncodeRoundTrip(t *testing.T) {
	a := NewHistogram(0.02)
	b := NewHistogram(0.02)
	for i := 0; i < 500; i++ {
		a.Observe(float64(i))
		b.Observe(float64(i + 500))
	}
	if err := a.Merge(b); err != nil {
		t.Fatalf("merge: %v", err)
	}
	if a.Count() != 1000 {
		t.Fatalf("merged count = %d, want 1000", a.Count())
	}
	enc := a.Encode()
	dec, err := enc.Decode()
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if dec.Count() != a.Count() || dec.Sum() != a.Sum() {
		t.Errorf("round trip mismatch: count %d/%d sum %.1f/%.1f", dec.Count(), a.Count(), dec.Sum(), a.Sum())
	}
	// Deterministic encoding.
	enc2 := dec.Encode()
	if len(enc.BucketIndex) != len(enc2.BucketIndex) {
		t.Errorf("bucket count changed on re-encode")
	}
}

func TestHistogramMergeAccuracyMismatch(t *testing.T) {
	a := NewHistogram(0.01)
	b := NewHistogram(0.05)
	if err := a.Merge(b); err == nil {
		t.Fatal("expected error merging histograms with different accuracy")
	}
}

func TestHistogramBucketBound(t *testing.T) {
	h := NewHistogram(0.0001) // tiny accuracy → many buckets
	for i := 0; i < 100000; i++ {
		h.Observe(float64(i)*1.0001 + 1)
	}
	if len(h.buckets) > histogramMaxBuckets {
		t.Fatalf("bucket count %d exceeds bound %d", len(h.buckets), histogramMaxBuckets)
	}
}
