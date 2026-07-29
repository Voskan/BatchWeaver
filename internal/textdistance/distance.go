// Package textdistance provides a small, standard-library-only edit-distance
// implementation used to suggest corrections for misspelled configuration field
// names. It is deliberately minimal and has no dependencies.
package textdistance

// Levenshtein returns the Levenshtein edit distance between a and b: the minimum
// number of single-character insertions, deletions, or substitutions needed to
// transform a into b. Comparison is over bytes, which is sufficient for the
// ASCII configuration field names this package is used with.
func Levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	// Two-row dynamic programming to keep allocation O(len(b)).
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

// Closest returns the candidate closest to target within maxDistance, and true,
// or an empty string and false when no candidate qualifies. Ties are broken
// deterministically by preferring the smaller distance and then the
// lexicographically smaller candidate.
func Closest(target string, candidates []string, maxDistance int) (string, bool) {
	best := ""
	bestDist := maxDistance + 1
	for _, c := range candidates {
		d := Levenshtein(target, c)
		if d < bestDist || (d == bestDist && c < best) {
			best, bestDist = c, d
		}
	}
	if bestDist <= maxDistance {
		return best, true
	}
	return "", false
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
