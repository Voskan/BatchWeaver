package operation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Voskan/BatchWeaver/diagnostics"
)

// KeyContract describes properties of an operation's key that affect batching.
// It is deliberately small in version 1; richer key modeling arrives with the
// compiler and runtime prompts.
type KeyContract struct {
	comparable         bool
	canonicalKeySymbol Symbol
}

// ErrInvalidKeyContract is returned when a KeyContract fails validation.
var ErrInvalidKeyContract = errors.New("invalid key contract")

// DefaultKeyContract returns a key contract that does not require comparable
// keys and defines no canonical-key derivation.
func DefaultKeyContract() KeyContract { return KeyContract{} }

// NewKeyContract returns a key contract. If canonical is non-zero it is
// validated. Set comparable when keys are guaranteed to be Go-comparable.
func NewKeyContract(comparable bool, canonical Symbol) (KeyContract, error) {
	c := KeyContract{comparable: comparable, canonicalKeySymbol: canonical}
	if err := c.Validate(); err != nil {
		return KeyContract{}, err
	}
	return c, nil
}

// Comparable reports whether keys are guaranteed to be Go-comparable.
func (c KeyContract) Comparable() bool { return c.comparable }

// CanonicalKeySymbol returns the symbol used to derive a canonical key, if any.
func (c KeyContract) CanonicalKeySymbol() Symbol { return c.canonicalKeySymbol }

// Validate reports whether the key contract is well-formed.
func (c KeyContract) Validate() error {
	if !c.canonicalKeySymbol.IsZero() {
		if err := c.canonicalKeySymbol.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidKeyContract, err)
		}
	}
	return nil
}

// Extension carries opaque, namespaced adapter data that BatchWeaver preserves
// but does not interpret. Keeping it as isolated bytes avoids spreading
// map[string]any through the core semantic types.
type Extension struct {
	// Namespace is a vendor-owned key such as "example.com/vendor/adapter".
	Namespace string
	// Data is the raw, uninterpreted extension payload.
	Data []byte
}

// ErrInvalidExtension is returned when an Extension fails validation.
var ErrInvalidExtension = errors.New("invalid extension")

// Validate reports whether the extension has a well-formed namespace.
func (e Extension) Validate() error {
	if e.Namespace == "" {
		return fmt.Errorf("%w: namespace is empty", ErrInvalidExtension)
	}
	if strings.ContainsAny(e.Namespace, " \t\r\n") {
		return fmt.Errorf("%w: namespace contains whitespace", ErrInvalidExtension)
	}
	if !strings.Contains(e.Namespace, "/") && !strings.Contains(e.Namespace, ".") {
		return fmt.Errorf("%w: namespace %q must be a namespaced key", ErrInvalidExtension, e.Namespace)
	}
	return nil
}

// clone returns a deep copy of the extension so stored data is not aliased.
func (e Extension) clone() Extension {
	return Extension{Namespace: e.Namespace, Data: append([]byte(nil), e.Data...)}
}

// Source locates where an operation spec was declared, for diagnostics. It is
// metadata only and never contributes to the semantic digest.
type Source struct {
	// Range is the declaration location in a configuration file, if known.
	Range diagnostics.Range
}
