package textdistance

import "testing"

func TestLevenshtein(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"max_size", "max_batch_items", 9},
		{"max_size", "max_size", 0},
		{"mas_size", "max_size", 1},
	}
	for _, tt := range tests {
		if got := Levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("Levenshtein(%q,%q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestClosest(t *testing.T) {
	t.Parallel()
	fields := []string{"max_size", "min_size", "max_wait", "max_weight"}
	if got, ok := Closest("max_sze", fields, 2); !ok || got != "max_size" {
		t.Errorf("Closest = %q, %v; want max_size, true", got, ok)
	}
	if _, ok := Closest("completely_different", fields, 2); ok {
		t.Errorf("Closest matched an unrelated field")
	}
}

func TestClosestDeterministicTie(t *testing.T) {
	t.Parallel()
	// "ax" is distance 1 from both "bx" and "ax?"; ensure deterministic choice.
	got, ok := Closest("ax", []string{"bx", "cx"}, 1)
	if !ok || got != "bx" {
		t.Errorf("Closest tie = %q, %v; want bx, true", got, ok)
	}
}
