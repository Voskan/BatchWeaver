package adaptive

import (
	"math"
	"sort"
)

// WorkloadPattern names a synthetic arrival pattern.
type WorkloadPattern string

const (
	// PatternPoisson is Poisson-like exponential inter-arrival.
	PatternPoisson WorkloadPattern = "poisson"
	// PatternBursty alternates bursts and quiet gaps.
	PatternBursty WorkloadPattern = "bursty"
	// PatternPeriodic has fixed inter-arrival spacing.
	PatternPeriodic WorkloadPattern = "periodic"
	// PatternHeavyTail has heavy-tailed inter-arrival gaps.
	PatternHeavyTail WorkloadPattern = "heavy-tail"
)

// WorkloadSpec parameterizes a deterministic synthetic workload. Given the same
// spec (including Seed) it always produces the same events, so replay and
// simulation are reproducible across platforms.
type WorkloadSpec struct {
	Pattern          WorkloadPattern
	Operation        string
	Count            int
	RatePerSec       float64
	Seed             uint64
	DuplicationRate  float64 // fraction of arrivals reusing a recent key
	DistinctKeys     int     // key space size
	TenantClasses    int     // number of tenant classes
	TenantSkew       float64 // 0 uniform, 1 highly skewed
	DeadlineFraction float64 // fraction of calls with a deadline
	DeadlineNanos    int64   // relative deadline for deadlined calls
	PayloadBytes     int
	// PhaseChangeAt, when >0, doubles the arrival rate after this many events to
	// simulate a workload phase change.
	PhaseChangeAt int
}

// Event is one synthetic (or recorded) logical call.
type Event struct {
	ArrivalNanos   int64  `json:"arrival_nanos"`
	Operation      string `json:"operation"`
	PartitionClass string `json:"partition_class"`
	TenantClass    string `json:"tenant_class"`
	Key            uint64 `json:"key"`
	DeadlineNanos  int64  `json:"deadline_nanos"` // relative to arrival; 0 = none
	PayloadBytes   int    `json:"payload_bytes"`
}

// rng is a small deterministic splitmix64 generator, used so synthetic
// workloads are reproducible without depending on the math/rand global state or
// its platform-specific streams.
type rng struct{ state uint64 }

func newRNG(seed uint64) *rng { return &rng{state: seed + 0x9e3779b97f4a7c15} }

func (r *rng) next() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// float01 returns a deterministic float in [0,1).
func (r *rng) float01() float64 {
	return float64(r.next()>>11) / float64(1<<53)
}

// exp returns an exponential variate with the given mean.
func (r *rng) exp(mean float64) float64 {
	u := r.float01()
	if u <= 0 {
		u = 1e-12
	}
	return -mean * math.Log(u)
}

// GenerateWorkload produces a deterministic event stream for the spec. Events are
// returned sorted by arrival time.
func GenerateWorkload(spec WorkloadSpec) []Event {
	if spec.Count <= 0 {
		return nil
	}
	if spec.RatePerSec <= 0 {
		spec.RatePerSec = 1000
	}
	if spec.DistinctKeys <= 0 {
		spec.DistinctKeys = spec.Count
	}
	if spec.TenantClasses <= 0 {
		spec.TenantClasses = 1
	}
	r := newRNG(spec.Seed)

	events := make([]Event, 0, spec.Count)
	var t float64
	var recentKey uint64
	for i := 0; i < spec.Count; i++ {
		rate := spec.RatePerSec
		if spec.PhaseChangeAt > 0 && i >= spec.PhaseChangeAt {
			rate = spec.RatePerSec * 2
		}
		mean := 1e9 / rate
		gap := interArrival(r, spec.Pattern, mean, i)
		t += gap

		var key uint64
		if i > 0 && r.float01() < spec.DuplicationRate {
			key = recentKey
		} else {
			key = uint64(r.next() % uint64(spec.DistinctKeys))
			recentKey = key
		}

		var deadline int64
		if spec.DeadlineFraction > 0 && r.float01() < spec.DeadlineFraction {
			deadline = spec.DeadlineNanos
			if deadline <= 0 {
				deadline = int64(5 * mean)
			}
		}

		events = append(events, Event{
			ArrivalNanos:   int64(t),
			Operation:      spec.Operation,
			PartitionClass: "class_p0",
			TenantClass:    tenantClass(r, spec),
			Key:            key,
			DeadlineNanos:  deadline,
			PayloadBytes:   spec.PayloadBytes,
		})
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].ArrivalNanos < events[j].ArrivalNanos })
	return events
}

// interArrival returns the next gap for a pattern.
func interArrival(r *rng, pattern WorkloadPattern, mean float64, i int) float64 {
	switch pattern {
	case PatternPeriodic:
		return mean
	case PatternBursty:
		// Ten tight arrivals, then a long gap.
		if i%10 == 9 {
			return mean * 10
		}
		return mean * 0.1
	case PatternHeavyTail:
		// Occasional very large gaps (Pareto-like via exponentiated uniform).
		u := r.float01()
		return mean * math.Pow(1-u, -0.5)
	default: // Poisson
		return r.exp(mean)
	}
}

// tenantClass picks a tenant class, optionally skewed toward class 0.
func tenantClass(r *rng, spec WorkloadSpec) string {
	if spec.TenantClasses <= 1 {
		return "class_t0"
	}
	u := r.float01()
	idx := int(u * float64(spec.TenantClasses))
	if spec.TenantSkew > 0 && r.float01() < spec.TenantSkew {
		idx = 0 // bias toward the first tenant class
	}
	if idx >= spec.TenantClasses {
		idx = spec.TenantClasses - 1
	}
	return "class_t" + itoa(idx)
}

// itoa is a tiny non-allocating-ish integer to string for class labels.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
