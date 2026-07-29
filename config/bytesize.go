package config

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ErrInvalidByteSize is returned when a byte-size string cannot be parsed.
var ErrInvalidByteSize = errors.New("invalid byte size")

// ByteSize is a non-negative size in bytes. It is parsed from strings with an
// explicit unit; bare integers are rejected to avoid ambiguity. Binary units
// (KiB, MiB, GiB, TiB) and decimal units (KB, MB, GB, TB) are accepted; the
// canonical rendering uses binary units.
type ByteSize int64

// Binary unit multipliers.
const (
	kiB = 1024
	miB = kiB * 1024
	giB = miB * 1024
	tiB = giB * 1024
)

// byteUnits maps a unit suffix to its multiplier. Longer suffixes are checked
// first by ParseByteSize.
var byteUnits = []struct {
	suffix string
	mult   int64
}{
	{"KiB", kiB}, {"MiB", miB}, {"GiB", giB}, {"TiB", tiB},
	{"KB", 1000}, {"MB", 1000 * 1000}, {"GB", 1000 * 1000 * 1000}, {"TB", 1000 * 1000 * 1000 * 1000},
	{"B", 1},
}

// ParseByteSize parses a byte size such as "64KiB" or "4MiB". It requires an
// explicit unit and rejects negative values and overflow.
func ParseByteSize(s string) (ByteSize, error) {
	if s == "" {
		return 0, fmt.Errorf("%w: empty", ErrInvalidByteSize)
	}
	for _, u := range byteUnits {
		if strings.HasSuffix(s, u.suffix) {
			numPart := strings.TrimSuffix(s, u.suffix)
			if numPart == "" {
				return 0, fmt.Errorf("%w: %q has no numeric part", ErrInvalidByteSize, s)
			}
			n, err := strconv.ParseInt(numPart, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("%w: %q: %w", ErrInvalidByteSize, s, err)
			}
			if n < 0 {
				return 0, fmt.Errorf("%w: %q must not be negative", ErrInvalidByteSize, s)
			}
			if n != 0 && u.mult > math.MaxInt64/n {
				return 0, fmt.Errorf("%w: %q overflows", ErrInvalidByteSize, s)
			}
			return ByteSize(n * u.mult), nil
		}
	}
	return 0, fmt.Errorf("%w: %q has no recognized unit", ErrInvalidByteSize, s)
}

// Bytes returns the size in bytes.
func (b ByteSize) Bytes() int64 { return int64(b) }

// String renders the size canonically using the largest binary unit that divides
// it evenly. Zero renders as "0B".
func (b ByteSize) String() string {
	if b == 0 {
		return "0B"
	}
	n := int64(b)
	for _, u := range []struct {
		suffix string
		mult   int64
	}{{"TiB", tiB}, {"GiB", giB}, {"MiB", miB}, {"KiB", kiB}} {
		if n%u.mult == 0 {
			return strconv.FormatInt(n/u.mult, 10) + u.suffix
		}
	}
	return strconv.FormatInt(n, 10) + "B"
}

// MarshalText implements encoding.TextMarshaler.
func (b ByteSize) MarshalText() ([]byte, error) { return []byte(b.String()), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *ByteSize) UnmarshalText(data []byte) error {
	v, err := ParseByteSize(string(data))
	if err != nil {
		return err
	}
	*b = v
	return nil
}
