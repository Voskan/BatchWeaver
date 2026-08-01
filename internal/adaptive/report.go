package adaptive

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// OperationTuning is one operation's analysis in a tuning report.
type OperationTuning struct {
	Operation              string       `json:"operation"`
	Current                Settings     `json:"current"`
	Recommended            Settings     `json:"recommended"`
	Applied                bool         `json:"applied"`
	Evidence               Evidence     `json:"evidence"`
	Confidence             Confidence   `json:"confidence"`
	ConfidenceLabel        string       `json:"confidence_label"`
	Reasons                []string     `json:"reasons"`
	Diagnostics            []Diagnostic `json:"diagnostics,omitempty"`
	ExpectedImprovementPct float64      `json:"expected_improvement_pct"`
	DecisionID             string       `json:"decision_id"`
}

// TuningReport is a deterministic tuning report across operations. It is
// truthful about uncertainty: every expected effect is labeled a model estimate.
type TuningReport struct {
	GeneratedUnixNanos int64             `json:"generated_unix_nanos"`
	Objective          ObjectivePolicy   `json:"objective"`
	ControllerVersion  string            `json:"controller_version"`
	CostModelVersion   string            `json:"cost_model_version"`
	ProfileDigest      Digest            `json:"profile_digest"`
	Operations         []OperationTuning `json:"operations"`
}

// AnalyzeBundle runs the controller over every operation in a bundle and returns
// a tuning report, ranked by expected improvement (largest first). currentByOp
// supplies each operation's current settings; a missing entry uses zero
// settings. It never applies changes; use the controller directly for that.
func AnalyzeBundle(ctrl *Controller, bundle *ProfileBundle, currentByOp map[string]Settings, ageSeconds float64) TuningReport {
	rep := TuningReport{
		GeneratedUnixNanos: ctrl.cfg.Clock.Now().UnixNano(),
		Objective:          ctrl.cfg.Objective,
		ControllerVersion:  ControllerVersion,
		CostModelVersion:   CostModelVersion,
		ProfileDigest:      bundle.Digest,
	}
	for i := range bundle.Operations {
		op := &bundle.Operations[i]
		cur := currentByOp[op.Operation]
		d := ctrl.Recommend(RecommendInput{
			Profile:           op,
			Current:           cur,
			ProfileAgeSeconds: ageSeconds,
			ProfileDigest:     op.Digest,
		})
		improvement := 0.0
		if d.ExpectedCurrent.Total > 0 {
			improvement = 100 * (d.ExpectedCurrent.Total - d.Expected.Total) / d.ExpectedCurrent.Total
		}
		rep.Operations = append(rep.Operations, OperationTuning{
			Operation:              op.Operation,
			Current:                d.Previous,
			Recommended:            d.New,
			Applied:                d.Applied,
			Evidence:               d.Evidence,
			Confidence:             d.Confidence,
			ConfidenceLabel:        d.Confidence.Label(),
			Reasons:                d.Reasons,
			Diagnostics:            d.Diagnostics,
			ExpectedImprovementPct: improvement,
			DecisionID:             d.ID,
		})
	}
	sort.SliceStable(rep.Operations, func(i, j int) bool {
		if rep.Operations[i].ExpectedImprovementPct != rep.Operations[j].ExpectedImprovementPct {
			return rep.Operations[i].ExpectedImprovementPct > rep.Operations[j].ExpectedImprovementPct
		}
		return rep.Operations[i].Operation < rep.Operations[j].Operation
	})
	return rep
}

// JSON renders the report as indented JSON.
func (r TuningReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// Text renders a plain-text tuning report.
func (r TuningReport) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "BatchWeaver tuning report\n\nObjective:\n  %s\n\n", r.Objective)
	fmt.Fprintf(&b, "Controller:\n  %s\nCost model:\n  %s\n\n", r.ControllerVersion, r.CostModelVersion)
	if len(r.Operations) == 0 {
		b.WriteString("No operations in profile.\n")
		return b.String()
	}
	for _, o := range r.Operations {
		fmt.Fprintf(&b, "Operation:\n  %s\n\n", o.Operation)
		fmt.Fprintf(&b, "Current policy:\n  max_wait:        %s\n  max_batch_size:  %d\n  concurrency:     %d\n\n",
			o.Current.MaxWait(), o.Current.MaxBatchSize, o.Current.MaxConcurrency)
		fmt.Fprintf(&b, "Measured:\n  arrival rate:       %.1f/s\n  duplicate rate:     %.3f\n  p95 queue delay:    %s\n  p95 backend:        %s\n  deadline miss rate: %.4f\n\n",
			o.Evidence.ArrivalRatePerSec, o.Evidence.DuplicateRate,
			time.Duration(o.Evidence.P95QueueDelayNanos), time.Duration(o.Evidence.P95BackendNanos), o.Evidence.DeadlineMissRate)
		fmt.Fprintf(&b, "Recommendation (model estimate):\n  max_wait:        %s\n  max_batch_size:  %d\n  concurrency:     %d\n\n",
			o.Recommended.MaxWait(), o.Recommended.MaxBatchSize, o.Recommended.MaxConcurrency)
		fmt.Fprintf(&b, "Expected effect (model estimate):\n  modeled cost change: %+.1f%%\n\n", o.ExpectedImprovementPct)
		fmt.Fprintf(&b, "Confidence:\n  %s\n\n", o.ConfidenceLabel)
		if len(o.Reasons) > 0 {
			b.WriteString("Reasons:\n")
			for _, rs := range o.Reasons {
				fmt.Fprintf(&b, "  - %s\n", rs)
			}
			b.WriteString("\n")
		}
		if o.Applied {
			b.WriteString("Status:\n  applied within hard bounds\n")
		} else {
			b.WriteString("Status:\n  advisory (not applied)\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Markdown renders a Markdown tuning report suitable for a repository artifact.
func (r TuningReport) Markdown() string {
	var b strings.Builder
	b.WriteString("# BatchWeaver tuning report\n\n")
	fmt.Fprintf(&b, "- Objective: `%s`\n- Controller: `%s`\n- Cost model: `%s`\n- Profile: `%s`\n\n",
		r.Objective, r.ControllerVersion, r.CostModelVersion, r.ProfileDigest.Short())
	b.WriteString("> Expected effects are model estimates, not guarantees.\n\n")
	b.WriteString("## Operation rankings\n\n")
	b.WriteString("| Operation | Current wait | Rec. wait | Current size | Rec. size | Modeled change | Confidence | Applied |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, o := range r.Operations {
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %d | %+.1f%% | %s | %t |\n",
			o.Operation, o.Current.MaxWait(), o.Recommended.MaxWait(),
			o.Current.MaxBatchSize, o.Recommended.MaxBatchSize,
			o.ExpectedImprovementPct, o.ConfidenceLabel, o.Applied)
	}
	b.WriteString("\n## Details\n\n")
	for _, o := range r.Operations {
		fmt.Fprintf(&b, "### %s\n\n", o.Operation)
		fmt.Fprintf(&b, "- Arrival rate: %.1f/s\n- Duplicate rate: %.3f\n- p95 queue delay: %s\n- p95 backend: %s\n\n",
			o.Evidence.ArrivalRatePerSec, o.Evidence.DuplicateRate,
			time.Duration(o.Evidence.P95QueueDelayNanos), time.Duration(o.Evidence.P95BackendNanos))
		for _, rs := range o.Reasons {
			fmt.Fprintf(&b, "- %s\n", rs)
		}
		b.WriteString("\n")
	}
	return b.String()
}
