package adaptive

// OverloadState is the scheduler's coarse load state.
type OverloadState string

const (
	// LoadNormal is normal operation.
	LoadNormal OverloadState = "normal"
	// LoadElevated is elevated load: shed low priority and flush earlier.
	LoadElevated OverloadState = "elevated"
	// LoadCritical is critical load: reject or fall back to protect the system.
	LoadCritical OverloadState = "critical"
)

// OverloadSignals are the measured load signals. No single signal forces a
// state; the detector combines them. Ratios are current/capacity in [0,1+].
type OverloadSignals struct {
	QueueDepthRatio     float64
	QueueAgeRatio       float64 // head-of-line age / max acceptable age
	MemoryRatio         float64
	BackendLatencyRatio float64 // current p95 / target p95
	TimeoutRate         float64
	ThrottleRate        float64
	CPURatio            float64
	GoroutineRatio      float64
	PoolSaturation      float64
	RejectedRate        float64
}

// OverloadConfig configures the detector's watermarks. Values are the fraction
// of capacity that marks elevated and critical load.
type OverloadConfig struct {
	QueueHighWatermark     float64 `json:"queue_high_watermark"`
	QueueCriticalWatermark float64 `json:"queue_critical_watermark"`
	// Policy selects the admission action taken under load.
	Policy AdmissionPolicy `json:"policy"`
}

func (c OverloadConfig) withDefaults() OverloadConfig {
	if c.QueueHighWatermark <= 0 {
		c.QueueHighWatermark = 0.8
	}
	if c.QueueCriticalWatermark <= 0 {
		c.QueueCriticalWatermark = 0.95
	}
	if c.Policy == "" {
		c.Policy = AdmitShedLowPriority
	}
	return c
}

// AdmissionPolicy selects what admission control does under load.
type AdmissionPolicy string

const (
	// AdmitAccept accepts the request.
	AdmitAccept AdmissionPolicy = "accept"
	// AdmitBlock blocks until capacity is available.
	AdmitBlock AdmissionPolicy = "block"
	// AdmitReject rejects with a typed error.
	AdmitReject AdmissionPolicy = "reject"
	// AdmitFallbackDirect runs the scalar fallback directly.
	AdmitFallbackDirect AdmissionPolicy = "fallback-direct"
	// AdmitShedLowPriority sheds low-priority requests only.
	AdmitShedLowPriority AdmissionPolicy = "shed-low-priority"
	// AdmitFlushEarly flushes the current batch early to relieve pressure.
	AdmitFlushEarly AdmissionPolicy = "flush-early"
)

// OverloadDetector computes the load state from signals using configured
// watermarks. It is stateless aside from its configuration and safe for
// concurrent use.
type OverloadDetector struct {
	cfg OverloadConfig
}

// NewOverloadDetector returns a detector for the configuration.
func NewOverloadDetector(cfg OverloadConfig) *OverloadDetector {
	return &OverloadDetector{cfg: cfg.withDefaults()}
}

// State returns the overload state implied by the signals. A signal at or above
// the critical watermark yields critical; at or above the high watermark yields
// elevated. Timeout, throttle, and rejection pressure also escalate.
func (d *OverloadDetector) State(s OverloadSignals) OverloadState {
	hi, crit := d.cfg.QueueHighWatermark, d.cfg.QueueCriticalWatermark
	maxRatio := max6(s.QueueDepthRatio, s.QueueAgeRatio, s.MemoryRatio, s.CPURatio, s.GoroutineRatio, s.PoolSaturation)
	maxRatio = maxf(maxRatio, s.BackendLatencyRatio)
	switch {
	case maxRatio >= crit || s.TimeoutRate > 0.05 || s.RejectedRate > 0.1:
		return LoadCritical
	case maxRatio >= hi || s.ThrottleRate > 0 || s.TimeoutRate > 0.01:
		return LoadElevated
	default:
		return LoadNormal
	}
}

// AdmissionRequest describes a request seeking admission under load.
type AdmissionRequest struct {
	// LowPriority marks a request eligible for shedding first.
	LowPriority bool
	// Critical marks a request that must not be shed (health checks, system).
	Critical bool
	// HasFallback reports whether a scalar fallback is available.
	HasFallback bool
}

// AdmissionDecision is the explicit, observable admission outcome. Requests are
// never shed silently; a shed or rejected request carries a diagnostic.
type AdmissionDecision struct {
	Action AdmissionPolicy `json:"action"`
	State  OverloadState   `json:"state"`
	Reason string          `json:"reason"`
	Diag   *Diagnostic     `json:"diagnostic,omitempty"`
}

// Admit decides how to handle a request given the current signals and the
// configured policy. It applies the policy only under elevated/critical load;
// under normal load it always accepts.
func (d *OverloadDetector) Admit(req AdmissionRequest, s OverloadSignals) AdmissionDecision {
	state := d.State(s)
	if state == LoadNormal {
		return AdmissionDecision{Action: AdmitAccept, State: state, Reason: "load normal"}
	}
	if req.Critical {
		return AdmissionDecision{Action: AdmitAccept, State: state, Reason: "critical request protected from shedding"}
	}
	policy := d.cfg.Policy
	switch policy {
	case AdmitShedLowPriority:
		if req.LowPriority {
			return shed(state, "low-priority request shed under load")
		}
		if state == LoadCritical && req.HasFallback {
			return fallback(state, "critical load: routing to scalar fallback")
		}
		return AdmissionDecision{Action: AdmitAccept, State: state, Reason: "non-low-priority admitted"}
	case AdmitFallbackDirect:
		if req.HasFallback {
			return fallback(state, "load control: scalar fallback")
		}
		return reject(state, "load control: no fallback available")
	case AdmitReject:
		return reject(state, "admission control rejected under load")
	case AdmitFlushEarly:
		return AdmissionDecision{Action: AdmitFlushEarly, State: state, Reason: "flushing early to relieve queue pressure"}
	case AdmitBlock:
		return AdmissionDecision{Action: AdmitBlock, State: state, Reason: "blocking until capacity is available"}
	default:
		return AdmissionDecision{Action: AdmitAccept, State: state, Reason: "default accept"}
	}
}

func shed(state OverloadState, reason string) AdmissionDecision {
	d := newDiag(CodeRequestShed, "warning", "", reason)
	return AdmissionDecision{Action: AdmitShedLowPriority, State: state, Reason: reason, Diag: &d}
}

func reject(state OverloadState, reason string) AdmissionDecision {
	d := newDiag(CodeAdmissionRejected, "warning", "", reason)
	return AdmissionDecision{Action: AdmitReject, State: state, Reason: reason, Diag: &d}
}

func fallback(state OverloadState, reason string) AdmissionDecision {
	return AdmissionDecision{Action: AdmitFallbackDirect, State: state, Reason: reason}
}

// Backpressure is the signal exposed to callers and adapters so they can slow
// down or route around an overloaded scheduler. The API is intentionally small.
type Backpressure struct {
	State              OverloadState `json:"state"`
	QueueFull          bool          `json:"queue_full"`
	Overloaded         bool          `json:"overloaded"`
	EstimatedWaitNanos int64         `json:"estimated_wait_nanos"`
	RetryAfterNanos    int64         `json:"retry_after_nanos"`
	RecommendDirect    bool          `json:"recommend_direct"`
}

// Backpressure derives a backpressure signal from the current signals and an
// estimated wait. It recommends direct mode under critical load.
func (d *OverloadDetector) Backpressure(s OverloadSignals, estimatedWaitNanos, retryAfterNanos int64) Backpressure {
	state := d.State(s)
	return Backpressure{
		State:              state,
		QueueFull:          s.QueueDepthRatio >= 1,
		Overloaded:         state != LoadNormal,
		EstimatedWaitNanos: estimatedWaitNanos,
		RetryAfterNanos:    retryAfterNanos,
		RecommendDirect:    state == LoadCritical,
	}
}

// OverloadDiagnostic returns a BW8301 diagnostic when the state is not normal.
func (d *OverloadDetector) OverloadDiagnostic(s OverloadSignals) *Diagnostic {
	if state := d.State(s); state != LoadNormal {
		diag := newDiag(CodeOverload, "warning", "", "scheduler load state: "+string(state))
		return &diag
	}
	return nil
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func max6(a, b, c, d, e, f float64) float64 {
	return maxf(maxf(maxf(a, b), maxf(c, d)), maxf(e, f))
}
