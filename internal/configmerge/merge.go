// Package configmerge merges decoded configuration node trees according to
// BatchWeaver's documented version-1 merge semantics.
//
// Merge operates on the position-aware node trees produced by configdecode, so
// presence is inherent (a key present in a document was explicitly set) and
// source positions are preserved for conflict diagnostics. Included documents
// are merged first in listed order; the including document is applied last.
package configmerge

import (
	"fmt"

	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/Voskan/BatchWeaver/internal/configdecode"
)

// operationsKey is the top-level field whose entries merge by operation ID.
const operationsKey = "operations"

// replaceKey, when true within an operation, permits replacing an inherited
// definition instead of raising a duplicate-operation conflict.
const replaceKey = "replace"

// Merge returns the result of applying override on top of base. Both must be
// mapping nodes (or nil). Top-level scalar and mapping sections from override
// replace or field-merge those in base; the operations map merges by ID with
// duplicate IDs reported as conflicts unless the incoming operation sets
// "replace: true". Diagnostics describe conflicts with both source locations.
func Merge(base, override *configdecode.Node, diags *diagnostics.Collection) *configdecode.Node {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}
	if !base.IsMapping() || !override.IsMapping() {
		// Non-mapping override replaces base entirely.
		return override
	}

	out := &configdecode.Node{Kind: configdecode.KindMapping, Pos: base.Pos}
	index := make(map[string]int)
	appendEntry := func(e configdecode.MapEntry) {
		if i, ok := index[e.Key]; ok {
			out.Entries[i] = e
			return
		}
		index[e.Key] = len(out.Entries)
		out.Entries = append(out.Entries, e)
	}

	for _, e := range base.Entries {
		appendEntry(e)
	}
	for _, e := range override.Entries {
		if e.Key == operationsKey {
			merged := mergeOperations(entryValue(base, operationsKey), e.Value, diags)
			appendEntry(configdecode.MapEntry{Key: operationsKey, KeyPos: e.KeyPos, Value: merged})
			continue
		}
		if prev, ok := entryLookup(base, e.Key); ok && e.Value.IsMapping() && prev.IsMapping() {
			appendEntry(configdecode.MapEntry{Key: e.Key, KeyPos: e.KeyPos, Value: Merge(prev, e.Value, diags)})
			continue
		}
		// Scalars and sequences replace wholesale.
		appendEntry(e)
	}
	return out
}

// mergeOperations merges two operation mappings by ID.
func mergeOperations(base, override *configdecode.Node, diags *diagnostics.Collection) *configdecode.Node {
	out := &configdecode.Node{Kind: configdecode.KindMapping}
	index := make(map[string]int)
	posOf := make(map[string]diagnostics.Position)
	add := func(e configdecode.MapEntry) {
		index[e.Key] = len(out.Entries)
		posOf[e.Key] = e.KeyPos
		out.Entries = append(out.Entries, e)
	}
	if base.IsMapping() {
		for _, e := range base.Entries {
			add(e)
		}
	}
	if override.IsMapping() {
		for _, e := range override.Entries {
			if i, exists := index[e.Key]; exists {
				if isReplace(e.Value) {
					out.Entries[i] = e
					posOf[e.Key] = e.KeyPos
					continue
				}
				d := diag(diagnostics.Position(e.KeyPos), fmt.Sprintf("duplicate operation %q; set \"replace: true\" to override an included definition", e.Key))
				if prev := posOf[e.Key]; !prev.IsZero() {
					d.Related = []diagnostics.RelatedInformation{{Message: "previous definition", Range: diagnostics.AtPosition(prev)}}
				}
				diags.Add(d)
				continue
			}
			add(e)
		}
	}
	return out
}

// isReplace reports whether an operation node explicitly sets replace: true.
func isReplace(n *configdecode.Node) bool {
	v, _, ok := n.Get(replaceKey)
	if !ok {
		return false
	}
	b, ok := configdecode.AsBool(v)
	return ok && b
}

// entryValue returns the value node for key in a mapping, or nil.
func entryValue(n *configdecode.Node, key string) *configdecode.Node {
	v, _, ok := n.Get(key)
	if !ok {
		return nil
	}
	return v
}

// entryLookup returns the value node for key and whether it exists.
func entryLookup(n *configdecode.Node, key string) (*configdecode.Node, bool) {
	v, _, ok := n.Get(key)
	return v, ok
}

// diag builds a duplicate-operation config diagnostic.
func diag(pos diagnostics.Position, msg string) diagnostics.Diagnostic {
	return diagnostics.Diagnostic{
		Code:     configdecode.CodeDuplicateOperation,
		Severity: diagnostics.SeverityError,
		Message:  msg,
		Source:   "config",
		Range:    diagnostics.AtPosition(pos),
	}
}
