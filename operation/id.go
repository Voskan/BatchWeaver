package operation

import (
	"errors"
	"fmt"
	"strings"
)

// ID is a stable operation identifier composed of dot-separated lowercase
// segments, for example "users.get" or "pricing.region.lookup".
//
// An ID is used in configuration, diagnostics, catalogs, and future generated
// registries, so its syntax is strict and never silently repaired. The zero
// value is the empty string, which is invalid.
type ID string

// ErrInvalidID is returned when an ID fails validation.
var ErrInvalidID = errors.New("invalid operation id")

const (
	maxIDBytes      = 255
	maxSegmentBytes = 63
)

// reservedPrefixes are first-segment values reserved for BatchWeaver-owned
// internal operations. User configuration must not use them.
var reservedPrefixes = map[string]struct{}{
	"batchweaver": {},
	"internal":    {},
	"system":      {},
}

// ParseID validates value and returns it as an ID. It does not modify the input
// in any way; invalid input is rejected rather than trimmed or lowercased.
func ParseID(value string) (ID, error) {
	if value == "" {
		return "", fmt.Errorf("%w: id is empty", ErrInvalidID)
	}
	if len(value) > maxIDBytes {
		return "", fmt.Errorf("%w: id exceeds %d bytes", ErrInvalidID, maxIDBytes)
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return "", fmt.Errorf("%w: id contains whitespace", ErrInvalidID)
	}
	if strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return "", fmt.Errorf("%w: id has a leading or trailing dot", ErrInvalidID)
	}
	segments := strings.Split(value, ".")
	if len(segments) < 2 {
		return "", fmt.Errorf("%w: id must have at least two segments", ErrInvalidID)
	}
	for i, seg := range segments {
		if err := validateSegment(seg); err != nil {
			return "", fmt.Errorf("%w: segment %d %q: %w", ErrInvalidID, i+1, seg, err)
		}
	}
	if _, reserved := reservedPrefixes[segments[0]]; reserved {
		return "", fmt.Errorf("%w: first segment %q is reserved", ErrInvalidID, segments[0])
	}
	return ID(value), nil
}

// MustParseID is like ParseID but panics on error. It is intended for
// package-level constants where an invalid ID is a programmer error. The panic
// message is concise and deterministic and contains no pointer addresses.
func MustParseID(value string) ID {
	id, err := ParseID(value)
	if err != nil {
		panic(fmt.Sprintf("operation.MustParseID(%q): %v", value, err))
	}
	return id
}

// validateSegment checks a single ID segment.
func validateSegment(seg string) error {
	if seg == "" {
		return errors.New("segment is empty")
	}
	if len(seg) > maxSegmentBytes {
		return fmt.Errorf("segment exceeds %d bytes", maxSegmentBytes)
	}
	for i := 0; i < len(seg); i++ {
		c := seg[i]
		switch {
		case c >= 'a' && c <= 'z':
			// always allowed
		case i == 0:
			return errors.New("segment must start with a lowercase letter")
		case (c >= '0' && c <= '9') || c == '_' || c == '-':
			// allowed after the first character
		default:
			return fmt.Errorf("segment contains invalid character %q", c)
		}
	}
	last := seg[len(seg)-1]
	if last == '_' || last == '-' {
		return errors.New("segment must not end with '_' or '-'")
	}
	if strings.Contains(seg, "--") || strings.Contains(seg, "__") ||
		strings.Contains(seg, "-_") || strings.Contains(seg, "_-") {
		return errors.New("segment must not contain consecutive separators")
	}
	return nil
}

// Validate reports whether the ID is well-formed.
func (id ID) Validate() error {
	_, err := ParseID(string(id))
	return err
}

// String returns the ID as a string.
func (id ID) String() string { return string(id) }

// Segments returns the dot-separated segments of the ID. It returns nil for an
// invalid ID.
func (id ID) Segments() []string {
	if id.Validate() != nil {
		return nil
	}
	return strings.Split(string(id), ".")
}

// MarshalText implements encoding.TextMarshaler, rejecting invalid IDs.
func (id ID) MarshalText() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return []byte(id), nil
}

// UnmarshalText implements encoding.TextUnmarshaler with strict validation.
func (id *ID) UnmarshalText(data []byte) error {
	parsed, err := ParseID(string(data))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}
