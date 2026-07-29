package configdecode

import (
	"fmt"

	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/Voskan/BatchWeaver/internal/textdistance"
)

// maxSuggestionDistance bounds how far an unknown field may be from a known one
// before no suggestion is offered.
const maxSuggestionDistance = 3

// CheckUnknownFields reports a diagnostic for each entry of node whose key is not
// in known, offering a "did you mean" suggestion when a close match exists.
func CheckUnknownFields(node *Node, known []string, diags *diagnostics.Collection) {
	if !node.IsMapping() {
		return
	}
	set := make(map[string]struct{}, len(known))
	for _, k := range known {
		set[k] = struct{}{}
	}
	for _, e := range node.Entries {
		if _, ok := set[e.Key]; ok {
			continue
		}
		d := diag(CodeUnknownField, e.KeyPos, fmt.Sprintf("unknown field %q", e.Key))
		if suggestion, found := textdistance.Closest(e.Key, known, maxSuggestionDistance); found {
			d.Details = fmt.Sprintf("did you mean %q?", suggestion)
		}
		diags.Add(*ptr(d))
	}
}

// ptr returns a pointer to d; a tiny helper that keeps the Add call readable.
func ptr(d diagnostics.Diagnostic) *diagnostics.Diagnostic { return &d }

// TypeName returns a human-readable name for a node's kind.
func TypeName(node *Node) string {
	switch {
	case node == nil:
		return "nothing"
	case node.IsMapping():
		return "a mapping"
	case node.IsSequence():
		return "a sequence"
	default:
		switch node.ScalarType {
		case ScalarString:
			return "a string"
		case ScalarInt:
			return "an integer"
		case ScalarFloat:
			return "a number"
		case ScalarBool:
			return "a boolean"
		case ScalarNull:
			return "null"
		default:
			return "a scalar"
		}
	}
}

// AsString returns the value of a string scalar and true, or false for any other
// node. It is strict: numeric and boolean scalars are not treated as strings.
func AsString(node *Node) (string, bool) {
	if node.IsScalar() && node.ScalarType == ScalarString {
		return node.Value, true
	}
	return "", false
}

// AsBool returns the value of a boolean scalar and true, or false otherwise.
func AsBool(node *Node) (bool, bool) {
	if node.IsScalar() && node.ScalarType == ScalarBool {
		return node.Value == "true", true
	}
	return false, false
}

// AsInt returns the value of an integer scalar and true, or false otherwise.
func AsInt(node *Node) (int64, bool) {
	if !node.IsScalar() || node.ScalarType != ScalarInt {
		return 0, false
	}
	var n int64
	_, err := fmt.Sscan(node.Value, &n)
	if err != nil {
		return 0, false
	}
	return n, true
}
