package proof

import "strings"

// Diagnostic codes by obligation family. Codes are stable and documented in
// docs/reference/proof-diagnostics.md.
const (
	CodeDeclaration = "BW5000"
	CodeOrder       = "BW5100"
	CodeDependency  = "BW5200"
	CodeEffect      = "BW5300"
	CodeReceiverKey = "BW5400"
	CodeResultError = "BW5500"
	CodeContext     = "BW5600"
	CodeTransaction = "BW5700"
	CodePanic       = "BW5800"
	CodePrecision   = "BW5900"
)

// diagnostics derives proof diagnostics from a candidate proof. A proven
// eligible candidate produces none. Ineligible candidates produce a warning;
// unknown and assumption-required candidates produce informational diagnostics
// so that unknowns are never silently reported as errors.
func (cc *candidateContext) diagnostics(cp CandidateProof) []Diag {
	if cp.Decision == DecisionProvenEligible {
		return nil
	}
	code, obl, strategy := cc.blockingDiagnostic(cp)
	severity := "info"
	if cp.Decision == DecisionProvenIneligible {
		severity = "warning"
	}
	msg := diagnosticMessage(cp.Decision, obl)
	return []Diag{{
		Code:        code,
		Severity:    severity,
		Message:     msg,
		Location:    cp.Location,
		Candidate:   cp.ID,
		Strategy:    strategy,
		Obligation:  obl,
		Fingerprint: shortID("bwfp", cp.ID, code, obl),
	}}
}

// blockingDiagnostic selects the code, obligation, and strategy that most
// explains a non-eligible decision.
func (cc *candidateContext) blockingDiagnostic(cp CandidateProof) (code, obligation, strategy string) {
	for _, s := range cp.AllowedStrategies {
		if s.Status == cp.Decision && len(s.BlockingObligations) > 0 {
			obligation = s.BlockingObligations[0]
			strategy = s.Strategy
			break
		}
	}
	if obligation == "" {
		for _, o := range cp.Obligations {
			if o.Status == ObligationViolated || o.Status == ObligationUnknown {
				obligation = o.ID
				break
			}
		}
	}
	spec, _ := Obligation(obligation)
	return codeForFamily(spec.Family), obligation, strategy
}

// codeForFamily maps an obligation family to a diagnostic code.
func codeForFamily(family string) string {
	switch family {
	case FamilyDeclaration:
		return CodeDeclaration
	case FamilyOrder:
		return CodeOrder
	case FamilyDependency:
		return CodeDependency
	case FamilyEffect:
		return CodeEffect
	case FamilyReceiver, FamilyKey:
		return CodeReceiverKey
	case FamilyResult, FamilyError:
		return CodeResultError
	case FamilyContext:
		return CodeContext
	case FamilyTransaction:
		return CodeTransaction
	case FamilyPanic:
		return CodePanic
	default:
		return CodePrecision
	}
}

// diagnosticMessage renders a stable diagnostic message.
func diagnosticMessage(decision Decision, obligation string) string {
	spec, _ := Obligation(obligation)
	title := spec.Title
	if title == "" {
		title = "a proof obligation"
	}
	switch decision {
	case DecisionProvenIneligible:
		return "candidate is ineligible: " + title + " is violated"
	case DecisionRequiresAssumption:
		return "candidate requires an explicit assumption: " + title
	case DecisionDeferred:
		return "candidate is deferred to a later stage"
	default:
		return "candidate is unknown: " + title + " could not be proven"
	}
}

// redact removes characters that would break terminal, JSON, or DOT rendering
// and never emits raw key material. It is applied to any free-form text that
// originates from analyzed source.
func redact(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\x00", "")
	return s
}
