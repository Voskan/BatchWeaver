package adaptive

import (
	"fmt"
	"sync"
	"time"
)

// ObservedMetrics are the post-change measurements a rollback monitor compares
// against the guardrails. They are measured, not modeled.
type ObservedMetrics struct {
	P95QueueDelayNanos   float64
	TimeoutRate          float64
	ErrorRate            float64
	ThrottleRate         float64
	QueueDepthRatio      float64 // current depth / capacity, in [0,1+]
	FairnessRegressed    bool
	VerificationMismatch bool
	ContractViolation    bool
}

// armedChange records an applied change so it can be rolled back if the guardrails
// are breached within the rollback window.
type armedChange struct {
	decisionID string
	previous   Settings
	guardrails SLOGuardrails
	baseline   ObservedMetrics
	armedAt    time.Time
}

// RollbackMonitor watches applied changes and, when a guardrail is breached
// within the rollback window, produces a rollback decision that restores the
// prior settings. It is safe for concurrent use and holds no secrets.
type RollbackMonitor struct {
	clock Clock

	mu    sync.Mutex
	armed map[string]armedChange // operation -> armed change
}

// NewRollbackMonitor returns a monitor using the given clock.
func NewRollbackMonitor(clock Clock) *RollbackMonitor {
	if clock == nil {
		clock = SystemClock()
	}
	return &RollbackMonitor{clock: clock, armed: map[string]armedChange{}}
}

// Arm records an applied decision for rollback surveillance. Only decisions that
// were applied and have a rollback window are armed. baseline captures the
// pre-change measurements used to detect regressions.
func (m *RollbackMonitor) Arm(d *Decision, baseline ObservedMetrics) {
	if !d.Applied || d.Guardrails.RollbackWindow <= 0 {
		return
	}
	m.mu.Lock()
	m.armed[d.Operation] = armedChange{
		decisionID: d.ID,
		previous:   d.Previous,
		guardrails: d.Guardrails,
		baseline:   baseline,
		armedAt:    m.clock.Now(),
	}
	m.mu.Unlock()
}

// Evaluate checks current measurements for the operation against the armed
// guardrails. When a breach is found within the window it returns a rollback
// decision restoring the previous settings and disarms; otherwise it returns
// (nil, false). After the window elapses without breach, the change is disarmed
// (committed) and (nil, false) is returned.
func (m *RollbackMonitor) Evaluate(operation string, cur ObservedMetrics) (*Decision, bool) {
	m.mu.Lock()
	ac, ok := m.armed[operation]
	if !ok {
		m.mu.Unlock()
		return nil, false
	}
	now := m.clock.Now()
	if now.Sub(ac.armedAt) > ac.guardrails.RollbackWindow {
		delete(m.armed, operation) // commit: window elapsed without breach
		m.mu.Unlock()
		return nil, false
	}
	reason, breached := guardrailBreach(ac, cur)
	if !breached {
		m.mu.Unlock()
		return nil, false
	}
	delete(m.armed, operation)
	m.mu.Unlock()

	d := &Decision{
		Operation:          operation,
		Previous:           Settings{}, // unknown post-change value; New restores prior
		New:                ac.previous,
		Mode:               TuningActive,
		Applied:            true,
		Guardrails:         ac.guardrails,
		Reasons:            []string{"automatic rollback: " + reason, "restoring settings prior to decision " + ac.decisionID},
		Diagnostics:        []Diagnostic{newDiag(CodePolicyRolledBack, "warning", operation, reason)},
		TimestampUnixNanos: now.UnixNano(),
	}
	d.finalize()
	return d, true
}

// guardrailBreach reports the first guardrail the current metrics violate.
func guardrailBreach(ac armedChange, cur ObservedMetrics) (string, bool) {
	g := ac.guardrails
	if cur.VerificationMismatch {
		return "runtime verification mismatch", true
	}
	if cur.ContractViolation {
		return "contract violation", true
	}
	if g.P95QueueDelayNanos > 0 && cur.P95QueueDelayNanos > float64(g.P95QueueDelayNanos) {
		return fmt.Sprintf("p95 queue delay %.0fns exceeds guardrail %dns", cur.P95QueueDelayNanos, g.P95QueueDelayNanos), true
	}
	if g.TimeoutRate > 0 && cur.TimeoutRate > g.TimeoutRate {
		return fmt.Sprintf("timeout rate %.4f exceeds guardrail %.4f", cur.TimeoutRate, g.TimeoutRate), true
	}
	// Error-rate regression is relative to the pre-change baseline.
	if cur.ErrorRate > ac.baseline.ErrorRate+g.ErrorRateRegression {
		return fmt.Sprintf("error rate regressed from %.4f to %.4f", ac.baseline.ErrorRate, cur.ErrorRate), true
	}
	if cur.ThrottleRate > ac.baseline.ThrottleRate && ac.baseline.ThrottleRate == 0 && cur.ThrottleRate > 0 {
		return "backend throttling appeared after the change", true
	}
	if cur.FairnessRegressed {
		return "fairness regression", true
	}
	return "", false
}

// Armed reports whether an operation currently has an armed change.
func (m *RollbackMonitor) Armed(operation string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.armed[operation]
	return ok
}
