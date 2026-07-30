package transform

import (
	"strings"
	"testing"
)

func TestUnifiedDiffBasic(t *testing.T) {
	t.Parallel()
	a := "line1\nline2\nline3\n"
	b := "line1\nCHANGED\nline3\n"
	d := UnifiedDiff("f.go", []byte(a), []byte(b), 3)
	if !strings.Contains(d, "--- a/f.go") || !strings.Contains(d, "+++ b/f.go") {
		t.Errorf("missing headers:\n%s", d)
	}
	if !strings.Contains(d, "-line2") || !strings.Contains(d, "+CHANGED") {
		t.Errorf("missing change lines:\n%s", d)
	}
}

func TestUnifiedDiffNoChange(t *testing.T) {
	t.Parallel()
	a := "x\ny\n"
	if d := UnifiedDiff("f.go", []byte(a), []byte(a), 3); d != "" {
		t.Errorf("expected empty diff, got:\n%s", d)
	}
}

func TestUnifiedDiffDeterministic(t *testing.T) {
	t.Parallel()
	a := "a\nb\nc\nd\ne\n"
	b := "a\nb\nX\nd\ne\n"
	first := UnifiedDiff("f", []byte(a), []byte(b), 2)
	second := UnifiedDiff("f", []byte(a), []byte(b), 2)
	if first != second {
		t.Error("diff is not deterministic")
	}
}

func FuzzUnifiedDiff(f *testing.F) {
	f.Add("a\nb\n", "a\nc\n")
	f.Add("", "x")
	f.Add("only-a", "")
	f.Fuzz(func(t *testing.T, a, b string) {
		// Must not panic and must be deterministic for any inputs.
		d1 := UnifiedDiff("f", []byte(a), []byte(b), 3)
		d2 := UnifiedDiff("f", []byte(a), []byte(b), 3)
		if d1 != d2 {
			t.Fatal("nondeterministic diff")
		}
		if a == b && d1 != "" {
			t.Fatalf("identical inputs produced a diff:\n%s", d1)
		}
	})
}
