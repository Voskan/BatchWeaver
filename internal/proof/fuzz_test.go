package proof

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Voskan/BatchWeaver/internal/analysis"
)

// FuzzProveNoPanic feeds arbitrary bytes into a synthesized snapshot and checks
// that the engine never panics and produces deterministic output.
func FuzzProveNoPanic(f *testing.F) {
	f.Add([]byte("read-only|direct|structural|parameter ctx|field r"))
	f.Add([]byte(""))
	f.Add([]byte("non-idempotent-write||result-dependent||"))
	f.Fuzz(func(t *testing.T, data []byte) {
		parts := strings.SplitN(string(data), "|", 5)
		for len(parts) < 5 {
			parts = append(parts, "")
		}
		states := []string{
			analysis.StatePotentialLoop, analysis.StatePotentialSiblings,
			analysis.StatePotentialFanout, analysis.StateAmbiguousTarget,
			analysis.StateDirectIsolated,
		}
		state := states[int(byteAt(data, 0))%len(states)]
		op := analysis.Operation{ID: "op.x", Compatibility: pick(parts[0], "valid", "unresolved", "invalid"), Kind: parts[0]}
		site := analysis.CallSite{
			ID: "s", Operation: "op.x", EnclosingFunctionID: "F", Targets: int(byteAt(data, 1) % 3),
			Dispatch:      pick(parts[1], analysis.DispatchDirect, analysis.DispatchInterface, analysis.DispatchUnknown),
			KeyDependency: parts[2], ContextArg: parts[3], Receiver: parts[4],
			LoopDepth: 1,
		}
		eff := analysis.EffectSummary{Function: "F", Effects: strings.Fields(parts[4]), Complete: len(data)%2 == 0}
		snap := buildSnapshot(op, site, eff, state)

		in := Input{Snapshot: snap, Reproducible: true, ContractDigest: parts[3]}
		r1, err1 := Prove(context.Background(), in)
		r2, err2 := Prove(context.Background(), in)
		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("nondeterministic error: %v vs %v", err1, err2)
		}
		if err1 != nil {
			return
		}
		var b1, b2 bytes.Buffer
		_ = RenderJSON(&b1, r1)
		_ = RenderJSON(&b2, r2)
		if !bytes.Equal(b1.Bytes(), b2.Bytes()) {
			t.Fatal("nondeterministic proof output for identical input")
		}
		// Every candidate must carry a decision within the closed set.
		for _, c := range r1.CandidateProofs {
			switch c.Decision {
			case DecisionProvenEligible, DecisionProvenIneligible,
				DecisionRequiresAssumption, DecisionUnknown, DecisionDeferred:
			default:
				t.Fatalf("invalid decision %q", c.Decision)
			}
		}
	})
}

// FuzzRedact ensures redaction always removes control characters that would
// corrupt terminal, JSON, or DOT output.
func FuzzRedact(f *testing.F) {
	f.Add("normal text")
	f.Add("line\nbreak\r\x00null")
	f.Fuzz(func(t *testing.T, s string) {
		out := redact(s)
		if strings.ContainsAny(out, "\n\r\x00") {
			t.Fatalf("redact left a control character in %q", out)
		}
	})
}

func byteAt(b []byte, i int) byte {
	if i < len(b) {
		return b[i]
	}
	return 0
}

func pick(s string, options ...string) string {
	for _, o := range options {
		if s == o {
			return o
		}
	}
	return options[0]
}
