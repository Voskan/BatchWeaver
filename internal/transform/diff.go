package transform

import (
	"fmt"
	"strings"
)

// DefaultDiffContext is the number of unchanged context lines around each hunk.
const DefaultDiffContext = 3

// UnifiedDiff returns a deterministic unified diff between original and
// transformed content. Paths are rendered with the conventional a/ and b/
// prefixes. No timestamps are emitted, so the output is stable across runs and
// hosts.
func UnifiedDiff(path string, original, transformed []byte, context int) string {
	if context <= 0 {
		context = DefaultDiffContext
	}
	a := splitLinesKeep(string(original))
	b := splitLinesKeep(string(transformed))
	ops := diffOps(a, b)
	hunks := groupHunks(ops, a, b, context)
	if len(hunks) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- a/%s\n", path)
	fmt.Fprintf(&sb, "+++ b/%s\n", path)
	for _, h := range hunks {
		sb.WriteString(h)
	}
	return sb.String()
}

// op is a single diff operation over lines.
type op struct {
	kind byte // ' ' equal, '-' delete (in a), '+' insert (in b)
	ai   int
	bi   int
}

// diffOps computes a line edit script using the classic LCS dynamic program.
// Inputs are bounded to developer-scale files.
func diffOps(a, b []string) []op {
	n, m := len(a), len(b)
	// lcs[i][j] = LCS length of a[i:], b[j:].
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case a[i] == b[j]:
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}
	var ops []op
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, op{' ', i, j})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			ops = append(ops, op{'-', i, j})
			i++
		default:
			ops = append(ops, op{'+', i, j})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, op{'-', i, j})
	}
	for ; j < m; j++ {
		ops = append(ops, op{'+', i, j})
	}
	return ops
}

// groupHunks assembles unified-diff hunks with the given context.
func groupHunks(ops []op, a, b []string, context int) []string {
	// Find indices of changed ops.
	changed := func(o op) bool { return o.kind != ' ' }
	var hunks []string
	i := 0
	for i < len(ops) {
		if !changed(ops[i]) {
			i++
			continue
		}
		// Expand a window [start,end) covering this change plus context, merging
		// nearby changes separated by <= 2*context equal lines.
		start := i
		for start > 0 && withinContext(ops, start-1, context, changed) {
			start--
		}
		end := i
		for end < len(ops) {
			if changed(ops[end]) {
				end++
				continue
			}
			// include trailing context, and merge if another change is close.
			if hasChangeWithin(ops, end, context, changed) {
				end++
				continue
			}
			break
		}
		// trim to context lines at both ends
		hStart := start
		hEnd := end
		lead := 0
		for hStart > 0 && !changed(ops[hStart-1]) && lead < context {
			hStart--
			lead++
		}
		trail := 0
		for hEnd < len(ops) && !changed(ops[hEnd]) && trail < context {
			hEnd++
			trail++
		}
		hunks = append(hunks, renderHunk(ops[hStart:hEnd], a, b))
		i = hEnd
	}
	return hunks
}

func withinContext(ops []op, idx, context int, changed func(op) bool) bool {
	// keep expanding left only across context-count equal lines that precede a change
	count := 0
	for k := idx; k >= 0 && count <= context; k-- {
		if changed(ops[k]) {
			return true
		}
		count++
	}
	return false
}

func hasChangeWithin(ops []op, idx, context int, changed func(op) bool) bool {
	for k := idx; k < len(ops) && k < idx+2*context; k++ {
		if changed(ops[k]) {
			return true
		}
	}
	return false
}

// renderHunk renders one @@ hunk.
func renderHunk(ops []op, a, b []string) string {
	var aStart, bStart, aCount, bCount int
	aStart, bStart = -1, -1
	for _, o := range ops {
		switch o.kind {
		case ' ':
			if aStart < 0 {
				aStart, bStart = o.ai, o.bi
			}
			aCount++
			bCount++
		case '-':
			if aStart < 0 {
				aStart, bStart = o.ai, o.bi
			}
			aCount++
		case '+':
			if aStart < 0 {
				aStart, bStart = o.ai, o.bi
			}
			bCount++
		}
	}
	if aStart < 0 {
		aStart, bStart = 0, 0
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n", aStart+1, aCount, bStart+1, bCount)
	for _, o := range ops {
		switch o.kind {
		case ' ':
			sb.WriteString(" " + a[o.ai])
		case '-':
			sb.WriteString("-" + a[o.ai])
		case '+':
			sb.WriteString("+" + b[o.bi])
		}
	}
	return sb.String()
}

// splitLinesKeep splits s into lines, each retaining its trailing newline except
// possibly the last.
func splitLinesKeep(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:]+"\n\\ No newline at end of file\n")
	}
	return out
}

// PlanDiff renders the full unified diff for a plan across all files in stable
// path order.
func PlanDiff(plan *Plan, context int) string {
	var sb strings.Builder
	for _, fp := range plan.Files {
		sb.WriteString(UnifiedDiff(fp.Path, fp.original, fp.transformed, context))
	}
	return sb.String()
}
