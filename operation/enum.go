package operation

import "fmt"

// enumString returns the canonical name for the value v given its ordered name
// table. Out-of-range values return "unknown".
func enumString[T ~uint8](v T, names []string) string {
	i := int(v)
	if i < 0 || i >= len(names) {
		return "unknown"
	}
	return names[i]
}

// parseEnum resolves s to a value using the ordered name table. Matching is
// case-sensitive and strict. On failure it returns the zero value and an error
// wrapping base.
func parseEnum[T ~uint8](s string, names []string, base error) (T, error) {
	for i, n := range names {
		if n == s {
			return T(i), nil
		}
	}
	return 0, fmt.Errorf("%w: %q", base, s)
}

// marshalEnum returns the canonical name bytes for v, or an error wrapping base
// when v is out of range.
func marshalEnum[T ~uint8](v T, names []string, base error) ([]byte, error) {
	i := int(v)
	if i < 0 || i >= len(names) {
		return nil, fmt.Errorf("%w: %d", base, i)
	}
	return []byte(names[i]), nil
}
