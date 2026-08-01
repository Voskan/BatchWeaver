package adaptive

import (
	"sort"
	"sync"
	"time"

	"github.com/Voskan/BatchWeaver/internal/buildinfo"
)

// CollectionMode selects how much workload data the collector records, trading
// fidelity for overhead. Production deployments should avoid full event storage.
type CollectionMode string

const (
	// CollectOff disables collection entirely.
	CollectOff CollectionMode = "off"
	// CollectCounters records only cumulative counters.
	CollectCounters CollectionMode = "counters"
	// CollectHistograms records counters plus bounded histograms (recommended for
	// production).
	CollectHistograms CollectionMode = "histograms"
	// CollectSampledEvents records histograms plus a sampled event stream.
	CollectSampledEvents CollectionMode = "sampled-events"
	// CollectFullLocalDebug records everything for local debugging only.
	CollectFullLocalDebug CollectionMode = "full-local-debug"
)

// recordsHistograms reports whether the mode populates histograms.
func (m CollectionMode) recordsHistograms() bool {
	switch m {
	case CollectHistograms, CollectSampledEvents, CollectFullLocalDebug:
		return true
	default:
		return false
	}
}

// recordsEvents reports whether the mode retains a sampled event stream.
func (m CollectionMode) recordsEvents() bool {
	return m == CollectSampledEvents || m == CollectFullLocalDebug
}

// CallObservation is one logical, caller-visible call. It carries no raw keys or
// identifiers: partition and tenant are pre-anonymized class labels (or raw
// values the collector will anonymize), and error/fallback reasons are class
// labels.
type CallObservation struct {
	Operation      string
	Adapter        string
	PartitionRaw   string // anonymized by the collector; never stored raw
	TenantRaw      string // anonymized by the collector; never stored raw
	QueueWait      time.Duration
	QueueDepth     int
	DeadlineSlack  time.Duration
	HasDeadline    bool
	DeadlineMissed bool
	PayloadBytes   int
	Duplicate      bool
	Fallback       bool
	FallbackReason string
	ErrorClass     string
	Mode           ExecutionMode
	Latency        time.Duration
	SampleKey      uint64
}

// BatchObservation is one dispatched batch.
type BatchObservation struct {
	Operation      string
	Adapter        string
	Size           int
	Weight         int64
	Chunks         int
	FlushReason    string
	BackendCall    bool
	BackendLatency time.Duration
	Serialization  time.Duration
	Mapping        time.Duration
	Throttled      bool
}

// Collector accumulates workload observations into a ProfileBundle with bounded
// memory and cardinality. It is safe for concurrent use. Overhead is documented
// and configurable through the collection mode; CollectOff makes every Record a
// cheap no-op.
type Collector struct {
	mode     CollectionMode
	accuracy float64
	sampler  *Sampler
	clock    Clock
	abi      string
	config   Digest

	partClasser   *Classer
	tenantClasser *Classer

	mu      sync.Mutex
	ops     map[string]*opAccumulator
	dropped uint64
	started time.Time
}

// CollectorOptions configures a Collector.
type CollectorOptions struct {
	Mode           CollectionMode
	Accuracy       float64
	Sampler        *Sampler
	Clock          Clock
	RuntimeABI     string
	ConfigDigest   Digest
	MaxPartClasses int
	MaxTenantClass int
	// ClasserSeed, when non-zero, makes anonymized class labels reproducible for
	// deterministic offline collection and tests. Production collection leaves it
	// zero so labels are randomly salted and cannot be correlated across sessions.
	ClasserSeed uint64
}

// NewCollector returns a Collector for the given options.
func NewCollector(opts CollectorOptions) *Collector {
	if opts.Clock == nil {
		opts.Clock = SystemClock()
	}
	if opts.Sampler == nil {
		opts.Sampler = NewSampler(SampleAll, 1)
	}
	if opts.Accuracy <= 0 {
		opts.Accuracy = DefaultHistogramAccuracy
	}
	if opts.Mode == "" {
		opts.Mode = CollectHistograms
	}
	partClasser := NewClasser(opts.MaxPartClasses)
	tenantClasser := NewClasser(opts.MaxTenantClass)
	if opts.ClasserSeed != 0 {
		partClasser = NewDeterministicClasser(opts.MaxPartClasses, opts.ClasserSeed)
		tenantClasser = NewDeterministicClasser(opts.MaxTenantClass, opts.ClasserSeed^0x5555555555555555)
	}
	return &Collector{
		mode:          opts.Mode,
		accuracy:      opts.Accuracy,
		sampler:       opts.Sampler,
		clock:         opts.Clock,
		abi:           opts.RuntimeABI,
		config:        opts.ConfigDigest,
		partClasser:   partClasser,
		tenantClasser: tenantClasser,
		ops:           make(map[string]*opAccumulator),
		started:       opts.Clock.Now(),
	}
}

// opAccumulator holds mutable accumulation state for one operation.
type opAccumulator struct {
	operation string
	adapter   string

	logicalCalls uint64
	interArrival *Histogram
	lastArrival  time.Time
	hasArrival   bool

	queueWait  *Histogram
	queueDepth *Histogram
	rejections uint64

	batchSize    *Histogram
	batchWeight  *Histogram
	batches      uint64
	backendCalls uint64
	flushReasons CategoricalCounts

	backendLatency *Histogram
	serialization  *Histogram
	mapping        *Histogram
	throttleEvents uint64

	deadlineSlack *Histogram
	withDeadline  uint64
	deadlineMiss  uint64

	errByClass CategoricalCounts
	errTotal   uint64

	duplicates uint64
	unique     uint64

	payloadBytes *Histogram

	partByClass    CategoricalCounts
	distinctPartCl map[string]struct{}

	fbByReason CategoricalCounts
	fbTotal    uint64

	chunks *Histogram

	fairShare CategoricalCounts
	fairWait  *Histogram

	modes map[ExecutionMode]*modeAccumulator
}

type modeAccumulator struct {
	calls        uint64
	backendCalls uint64
	latency      *Histogram
}

func (c *Collector) newOpAccumulator(operation, adapter string) *opAccumulator {
	return &opAccumulator{
		operation:      operation,
		adapter:        adapter,
		interArrival:   NewHistogram(c.accuracy),
		queueWait:      NewHistogram(c.accuracy),
		queueDepth:     NewHistogram(c.accuracy),
		batchSize:      NewHistogram(c.accuracy),
		batchWeight:    NewHistogram(c.accuracy),
		flushReasons:   CategoricalCounts{},
		backendLatency: NewHistogram(c.accuracy),
		serialization:  NewHistogram(c.accuracy),
		mapping:        NewHistogram(c.accuracy),
		deadlineSlack:  NewHistogram(c.accuracy),
		errByClass:     CategoricalCounts{},
		payloadBytes:   NewHistogram(c.accuracy),
		partByClass:    CategoricalCounts{},
		distinctPartCl: map[string]struct{}{},
		fbByReason:     CategoricalCounts{},
		chunks:         NewHistogram(c.accuracy),
		fairShare:      CategoricalCounts{},
		fairWait:       NewHistogram(c.accuracy),
		modes:          map[ExecutionMode]*modeAccumulator{},
	}
}

// opFor returns the accumulator for an operation, creating it on demand. The
// caller must hold c.mu.
func (c *Collector) opFor(operation, adapter string) *opAccumulator {
	acc := c.ops[operation]
	if acc == nil {
		acc = c.newOpAccumulator(operation, adapter)
		c.ops[operation] = acc
	}
	return acc
}

// RecordCall records one logical call. It is a no-op when collection is off.
func (c *Collector) RecordCall(o CallObservation) {
	if c.mode == CollectOff || o.Operation == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	acc := c.opFor(o.Operation, o.Adapter)
	acc.logicalCalls++

	now := c.clock.Now()
	if acc.hasArrival {
		gap := now.Sub(acc.lastArrival)
		if gap < 0 {
			gap = 0
		}
		if c.mode.recordsHistograms() {
			acc.interArrival.Observe(float64(gap.Nanoseconds()))
		}
	}
	acc.lastArrival = now
	acc.hasArrival = true

	if o.Duplicate {
		acc.duplicates++
	} else {
		acc.unique++
	}
	if o.HasDeadline {
		acc.withDeadline++
	}
	if o.DeadlineMissed {
		acc.deadlineMiss++
	}
	if o.Fallback {
		acc.fbTotal++
		reason := o.FallbackReason
		if reason == "" {
			reason = "unspecified"
		}
		boundedInc(acc.fbByReason, reason)
	}
	if o.ErrorClass != "" {
		acc.errTotal++
		boundedInc(acc.errByClass, o.ErrorClass)
	}

	partClass := c.partClasser.Class(o.PartitionRaw)
	acc.distinctPartCl[partClass] = struct{}{}
	boundedInc(acc.partByClass, partClass)

	tenantClass := c.tenantClasser.Class(o.TenantRaw)
	boundedInc(acc.fairShare, tenantClass)

	if o.Mode != "" {
		m := acc.modes[o.Mode]
		if m == nil {
			m = &modeAccumulator{latency: NewHistogram(c.accuracy)}
			acc.modes[o.Mode] = m
		}
		m.calls++
	}

	if c.mode.recordsHistograms() {
		acc.queueWait.Observe(float64(o.QueueWait.Nanoseconds()))
		if o.QueueDepth > 0 {
			acc.queueDepth.Observe(float64(o.QueueDepth))
		}
		if o.HasDeadline {
			acc.deadlineSlack.Observe(float64(o.DeadlineSlack.Nanoseconds()))
		}
		if o.PayloadBytes > 0 {
			acc.payloadBytes.Observe(float64(o.PayloadBytes))
		}
		acc.fairWait.Observe(float64(o.QueueWait.Nanoseconds()))
		if o.Mode != "" && o.Latency > 0 {
			acc.modes[o.Mode].latency.Observe(float64(o.Latency.Nanoseconds()))
		}
	}

	// Sampled-event modes honor the sampler for tail retention accounting.
	if c.mode.recordsEvents() {
		isTail := o.ErrorClass != "" || o.DeadlineMissed
		if !c.sampler.Sample(o.SampleKey, isTail) {
			c.dropped++
		}
	}
}

// RecordBatch records one dispatched batch. It is a no-op when collection is off.
func (c *Collector) RecordBatch(o BatchObservation) {
	if c.mode == CollectOff || o.Operation == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	acc := c.opFor(o.Operation, o.Adapter)
	acc.batches++
	if o.BackendCall {
		acc.backendCalls++
	}
	if o.Throttled {
		acc.throttleEvents++
	}
	if o.FlushReason != "" {
		boundedInc(acc.flushReasons, o.FlushReason)
	}
	if c.mode.recordsHistograms() {
		if o.Size > 0 {
			acc.batchSize.Observe(float64(o.Size))
		}
		if o.Weight > 0 {
			acc.batchWeight.Observe(float64(o.Weight))
		}
		if o.Chunks > 0 {
			acc.chunks.Observe(float64(o.Chunks))
		}
		if o.BackendLatency > 0 {
			acc.backendLatency.Observe(float64(o.BackendLatency.Nanoseconds()))
		}
		if o.Serialization > 0 {
			acc.serialization.Observe(float64(o.Serialization.Nanoseconds()))
		}
		if o.Mapping > 0 {
			acc.mapping.Observe(float64(o.Mapping.Nanoseconds()))
		}
	}
}

// RecordRejection records a queue/admission rejection.
func (c *Collector) RecordRejection(operation, adapter string) {
	if c.mode == CollectOff || operation == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.opFor(operation, adapter).rejections++
}

// maxCategoricalKeys bounds any categorical map so a hostile stream of distinct
// reasons or classes cannot grow memory without bound.
const maxCategoricalKeys = 512

// boundedInc increments a categorical count, collapsing overflow keys.
func boundedInc(m CategoricalCounts, key string) {
	if _, ok := m[key]; !ok && len(m) >= maxCategoricalKeys {
		m[overflowClass]++
		return
	}
	m[key]++
}

// Bundle returns the accumulated profile as a finalized, deterministic
// ProfileBundle. The window spans from collector start to now.
func (c *Collector) Bundle() *ProfileBundle {
	c.mu.Lock()
	defer c.mu.Unlock()
	info := buildinfo.Get()
	b := &ProfileBundle{
		Toolchain: ToolchainIdentity{
			GoVersion: info.GoVersion, Version: info.Version, Commit: info.Commit,
			GOOS: info.GOOS, GOARCH: info.GOARCH,
		},
		RuntimeABI:       c.abi,
		ConfigDigest:     c.config,
		CreatedUnixNanos: c.clock.Now().UnixNano(),
		Window: TimeWindow{
			StartUnixNanos: c.started.UnixNano(),
			EndUnixNanos:   c.clock.Now().UnixNano(),
		},
		Redaction: RedactionSummary{
			PartitionClassing: "hmac-sha256-bounded",
			TenantClassing:    "hmac-sha256-bounded",
			RawKeysStored:     false,
			RedactedFields:    c.partClasser.DistinctCount() + c.tenantClasser.DistinctCount(),
		},
	}
	names := make([]string, 0, len(c.ops))
	for name := range c.ops {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b.Operations = append(b.Operations, c.ops[name].build(c.sampler, c.dropped))
	}
	b.Finalize()
	return b
}

// Dropped returns the number of events dropped by sampling.
func (c *Collector) Dropped() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dropped
}

// build converts an accumulator into an OperationProfile.
func (acc *opAccumulator) build(sampler *Sampler, dropped uint64) OperationProfile {
	modes := make([]ExecutionModeProfile, 0, len(acc.modes))
	for m, ma := range acc.modes {
		modes = append(modes, ExecutionModeProfile{
			Mode: m, Calls: ma.calls, BackendCalls: ma.backendCalls,
			LatencyNanos: ma.latency.Encode(),
		})
	}
	fixed, perItem := estimateBackendCosts(acc.backendLatency, acc.batchSize)
	return OperationProfile{
		Operation:      acc.operation,
		Adapter:        acc.adapter,
		ExecutionModes: modes,
		Arrivals: ArrivalProfile{
			InterArrival: acc.interArrival.Encode(),
			LogicalCalls: acc.logicalCalls,
		},
		Queue: QueueProfile{
			WaitNanos:  acc.queueWait.Encode(),
			DepthItems: acc.queueDepth.Encode(),
			Rejections: acc.rejections,
		},
		Batches: BatchProfile{
			Size:         acc.batchSize.Encode(),
			Weight:       acc.batchWeight.Encode(),
			FlushReasons: acc.flushReasons,
			Batches:      acc.batches,
			BackendCalls: acc.backendCalls,
		},
		Backend: BackendProfile{
			LatencyNanos:       acc.backendLatency.Encode(),
			SerializationNanos: acc.serialization.Encode(),
			MappingNanos:       acc.mapping.Encode(),
			FixedCostNanos:     fixed,
			PerItemCostNanos:   perItem,
			ThrottleEvents:     acc.throttleEvents,
		},
		Deadlines: DeadlineProfile{
			SlackNanos:   acc.deadlineSlack.Encode(),
			WithDeadline: acc.withDeadline,
			Misses:       acc.deadlineMiss,
		},
		Errors:     ErrorProfile{ByClass: acc.errByClass, Total: acc.errTotal},
		Duplicates: DuplicateProfile{Duplicates: acc.duplicates, Unique: acc.unique},
		Payloads:   PayloadProfile{Bytes: acc.payloadBytes.Encode()},
		Partitions: PartitionProfile{ByClass: acc.partByClass, DistinctClasses: len(acc.distinctPartCl)},
		Fallbacks:  FallbackProfile{ByReason: acc.fbByReason, Total: acc.fbTotal},
		Chunks:     ChunkProfile{Count: acc.chunks.Encode()},
		Fairness:   FairnessProfile{ServiceShare: acc.fairShare, WaitNanos: acc.fairWait.Encode()},
		Sampling:   sampler.Metadata(dropped),
	}
}
