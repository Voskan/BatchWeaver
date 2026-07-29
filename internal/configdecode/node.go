// Package configdecode converts YAML and JSON configuration bytes into a
// uniform, position-aware node tree and provides strict decoding primitives on
// top of it.
//
// The node tree is intentionally decoupled from the YAML dependency's token
// types and from Go's encoding/json internals, so that the rest of BatchWeaver
// depends only on this stable representation. Positions are one-based lines and
// columns with a repository- or user-relative file name attached by the caller.
package configdecode

import "github.com/Voskan/BatchWeaver/diagnostics"

// Kind classifies a node in the decoded tree.
type Kind uint8

const (
	// KindScalar is a leaf value.
	KindScalar Kind = iota
	// KindMapping is an ordered set of key/value entries.
	KindMapping
	// KindSequence is an ordered list of values.
	KindSequence
)

// ScalarType records the source type of a scalar so decoding can be strict
// about, for example, rejecting a bare number where a duration string is
// required.
type ScalarType uint8

const (
	// ScalarString is a string scalar.
	ScalarString ScalarType = iota
	// ScalarInt is an integer scalar.
	ScalarInt
	// ScalarFloat is a floating-point scalar.
	ScalarFloat
	// ScalarBool is a boolean scalar.
	ScalarBool
	// ScalarNull is a null scalar.
	ScalarNull
)

// MapEntry is one key/value pair in a mapping node, retaining the key's source
// position for diagnostics.
type MapEntry struct {
	// Key is the mapping key.
	Key string
	// KeyPos is the source position of the key.
	KeyPos diagnostics.Position
	// Value is the associated value node.
	Value *Node
}

// Node is a position-aware element of a decoded configuration document.
type Node struct {
	// Kind is the node kind.
	Kind Kind
	// Pos is the source position of the node's value.
	Pos diagnostics.Position

	// ScalarType and Value are meaningful only for scalar nodes.
	ScalarType ScalarType
	Value      string

	// Entries is meaningful only for mapping nodes.
	Entries []MapEntry

	// Elems is meaningful only for sequence nodes.
	Elems []*Node
}

// IsScalar reports whether the node is a scalar.
func (n *Node) IsScalar() bool { return n != nil && n.Kind == KindScalar }

// IsMapping reports whether the node is a mapping.
func (n *Node) IsMapping() bool { return n != nil && n.Kind == KindMapping }

// IsSequence reports whether the node is a sequence.
func (n *Node) IsSequence() bool { return n != nil && n.Kind == KindSequence }

// IsNull reports whether the node is a null scalar.
func (n *Node) IsNull() bool {
	return n != nil && n.Kind == KindScalar && n.ScalarType == ScalarNull
}

// Get returns the value node and key position for key, and true when present.
// For a non-mapping node it returns false.
func (n *Node) Get(key string) (*Node, diagnostics.Position, bool) {
	if !n.IsMapping() {
		return nil, diagnostics.Position{}, false
	}
	for i := range n.Entries {
		if n.Entries[i].Key == key {
			return n.Entries[i].Value, n.Entries[i].KeyPos, true
		}
	}
	return nil, diagnostics.Position{}, false
}

// Keys returns the mapping keys in source order, or nil for a non-mapping node.
func (n *Node) Keys() []string {
	if !n.IsMapping() {
		return nil
	}
	out := make([]string, len(n.Entries))
	for i := range n.Entries {
		out[i] = n.Entries[i].Key
	}
	return out
}
