package editor

import (
	"sort"
	"strconv"
	"strings"

	"github.com/Voskan/BatchWeaver/internal/analysis"
)

// ScanSummary returns a deterministic plain-text summary of a workspace scan.
func ScanSummary(res *Result) string {
	snap := res.Snapshot
	var b strings.Builder
	b.WriteString("BatchWeaver workspace scan\n\n")
	b.WriteString("Operations:   " + strconv.Itoa(len(snap.Operations)) + "\n")
	b.WriteString("Call sites:   " + strconv.Itoa(len(snap.CallSites)) + "\n")
	b.WriteString("Candidates:   " + strconv.Itoa(len(snap.Candidates)) + "\n")
	b.WriteString("Diagnostics:  " + strconv.Itoa(len(snap.Diagnostics)) + "\n")
	b.WriteString("\nNo source changes have been applied.\n")
	return b.String()
}

// OperationGraphText renders operations and their candidate call sites as a
// deterministic DOT graph. When operationID is set, only that operation and its
// candidates are included.
func OperationGraphText(res *Result, operationID string) string {
	snap := res.Snapshot
	ops := append([]analysis.Operation(nil), snap.Operations...)
	sort.Slice(ops, func(i, j int) bool { return ops[i].ID < ops[j].ID })
	var b strings.Builder
	b.WriteString("digraph batchweaver {\n  rankdir=LR;\n")
	for _, op := range ops {
		if operationID != "" && op.ID != operationID {
			continue
		}
		b.WriteString("  \"op:" + op.ID + "\" [label=\"" + op.ID + "\", shape=box];\n")
		cands := candidatesFor(snap, op.ID)
		for _, c := range cands {
			node := "cand:" + c.ID
			b.WriteString("  \"" + node + "\" [label=\"" + c.StructuralContext + "\\n" +
				strconv.Itoa(len(c.CallSites)) + " sites\", shape=ellipse];\n")
			b.WriteString("  \"op:" + op.ID + "\" -> \"" + node + "\";\n")
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// candidatesFor returns the candidates of an operation, ID-sorted.
func candidatesFor(snap *analysis.Snapshot, opID string) []analysis.Candidate {
	var out []analysis.Candidate
	for _, c := range snap.Candidates {
		if c.Operation == opID {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// PreviewText returns a deterministic preview summary for a candidate. It shows
// the operation binding, structural context, evidence, and the CLI command that
// produces the exact deterministic diff. No source is modified.
func PreviewText(res *Result, candidateID string) string {
	var cand analysis.Candidate
	found := false
	for _, c := range res.Snapshot.Candidates {
		if c.ID == candidateID {
			cand, found = c, true
			break
		}
	}
	if !found {
		return "BatchWeaver: candidate " + candidateID + " not found in the current snapshot.\n" +
			"The snapshot may have changed; re-run the scan."
	}
	op := res.opByID[cand.Operation]
	var b strings.Builder
	b.WriteString("BatchWeaver transformation preview\n\n")
	b.WriteString("Operation:\n  " + cand.Operation + "\n\n")
	b.WriteString("Binding:\n  scalar: " + dash(op.ScalarSymbol) + "\n  batch:  " + dash(op.BatchSymbol) + "\n\n")
	b.WriteString("Structural context:\n  " + cand.StructuralContext + "\n\n")
	b.WriteString("Call sites:\n  " + strconv.Itoa(len(cand.CallSites)) + "\n\n")
	if len(cand.Evidence) > 0 {
		b.WriteString("Evidence:\n")
		for _, e := range cand.Evidence {
			b.WriteString("  - " + e + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Proof and exact diff:\n")
	b.WriteString("  Preview shows the candidate summary. For the exact deterministic\n")
	b.WriteString("  diff and proof certificate, run:\n\n")
	b.WriteString("    batchweaver prove\n")
	b.WriteString("    batchweaver transform diff --candidate=" + cand.ID + "\n\n")
	b.WriteString("No source changes have been applied.\n")
	return b.String()
}

// DoctorText returns a snapshot-derived environment summary for the editor.
func DoctorText(res *Result) string {
	snap := res.Snapshot
	var b strings.Builder
	b.WriteString("BatchWeaver editor doctor\n\n")
	b.WriteString("Go version:     " + snap.GoVersion + "\n")
	b.WriteString("Schema:         " + snap.SchemaVersion + "\n")
	b.WriteString("Packages:       " + strconv.Itoa(snap.PackagesLoaded) + "\n")
	b.WriteString("Operations:     " + strconv.Itoa(len(snap.Operations)) + "\n")
	if snap.Incomplete {
		b.WriteString("\nNote: analysis is incomplete (some packages failed to load).\n")
	}
	return b.String()
}
