package diagnostics

import (
	"errors"
	"fmt"
	"regexp"
)

// Code is a stable diagnostic identifier of the form BW<CATEGORY><NNN>, for
// example "BWCFG021". The prefix is always "BW"; the category is one or more
// uppercase ASCII letters; and the suffix is exactly three decimal digits.
//
// Codes are part of BatchWeaver's compatibility surface: once a code is
// committed and documented, it must not be reused for a different meaning.
type Code string

// ErrInvalidCode is returned when a Code fails validation.
var ErrInvalidCode = errors.New("invalid diagnostic code")

// codePattern matches a valid diagnostic code: the BW prefix, an uppercase
// category, and a three-digit numeric suffix.
var codePattern = regexp.MustCompile(`^BW[A-Z]+[0-9]{3}$`)

// Validate reports whether the code is well-formed. It does not check that the
// code is registered; registration is documented separately.
func (c Code) Validate() error {
	if c == "" {
		return fmt.Errorf("%w: code is empty", ErrInvalidCode)
	}
	if !codePattern.MatchString(string(c)) {
		return fmt.Errorf("%w: %q must match BW<CATEGORY><NNN>", ErrInvalidCode, string(c))
	}
	return nil
}

// Category returns the uppercase category segment between the "BW" prefix and
// the numeric suffix, for example "CFG" for "BWCFG021". It returns an empty
// string when the code is not well-formed.
func (c Code) Category() string {
	if c.Validate() != nil {
		return ""
	}
	// Strip the leading "BW" and the trailing three digits.
	return string(c[2 : len(c)-3])
}

// String returns the code as a string.
func (c Code) String() string { return string(c) }

// MarshalText implements encoding.TextMarshaler. It rejects invalid codes so
// that malformed codes never enter serialized output.
func (c Code) MarshalText() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return []byte(c), nil
}

// UnmarshalText implements encoding.TextUnmarshaler with strict validation.
func (c *Code) UnmarshalText(data []byte) error {
	candidate := Code(data)
	if err := candidate.Validate(); err != nil {
		return err
	}
	*c = candidate
	return nil
}
