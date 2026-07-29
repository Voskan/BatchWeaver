package diagnostics

import (
	"sort"
	"strings"
)

// Collection is a deterministic, ordered set of diagnostics.
//
// The zero value is an empty, ready-to-use collection. Mutating methods use a
// pointer receiver; query methods use a value receiver and never expose the
// internal slice, so callers cannot mutate stored diagnostics.
type Collection struct {
	diags []Diagnostic
}

// NewCollection returns an empty collection. The zero value is equally valid;
// this constructor exists for readability at call sites.
func NewCollection() Collection { return Collection{} }

// Add appends a single diagnostic.
func (c *Collection) Add(d Diagnostic) {
	c.diags = append(c.diags, d)
}

// AddCollection appends all diagnostics from other, copying them so the two
// collections do not share storage.
func (c *Collection) AddCollection(other Collection) {
	if len(other.diags) == 0 {
		return
	}
	c.diags = append(c.diags, other.diags...)
}

// Len returns the number of diagnostics.
func (c Collection) Len() int { return len(c.diags) }

// Empty reports whether the collection has no diagnostics.
func (c Collection) Empty() bool { return len(c.diags) == 0 }

// HasErrors reports whether any diagnostic has error severity.
func (c Collection) HasErrors() bool {
	for i := range c.diags {
		if c.diags[i].Severity == SeverityError {
			return true
		}
	}
	return false
}

// Count returns the number of diagnostics with the given severity.
func (c Collection) Count(sev Severity) int {
	n := 0
	for i := range c.diags {
		if c.diags[i].Severity == sev {
			n++
		}
	}
	return n
}

// Diagnostics returns a copy of the diagnostics in insertion order.
func (c Collection) Diagnostics() []Diagnostic {
	if len(c.diags) == 0 {
		return nil
	}
	out := make([]Diagnostic, len(c.diags))
	copy(out, c.diags)
	return out
}

// Sorted returns a new collection whose diagnostics are ordered by file, start
// line, start column, severity (errors first), code, and message. The receiver
// is not modified.
func (c Collection) Sorted() Collection {
	if len(c.diags) == 0 {
		return Collection{}
	}
	out := make([]Diagnostic, len(c.diags))
	copy(out, c.diags)
	sort.SliceStable(out, func(i, j int) bool {
		return lessDiagnostic(out[i], out[j])
	})
	return Collection{diags: out}
}

// Deduplicated returns a new collection with semantically identical diagnostics
// removed, preserving the order of first appearance. Identity is based on code,
// severity, source, start location, and message, not on pointer identity or
// advisory fixes.
func (c Collection) Deduplicated() Collection {
	if len(c.diags) == 0 {
		return Collection{}
	}
	seen := make(map[string]struct{}, len(c.diags))
	out := make([]Diagnostic, 0, len(c.diags))
	for i := range c.diags {
		key := c.diags[i].identity()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c.diags[i])
	}
	return Collection{diags: out}
}

// FilterBySeverity returns a new collection containing only diagnostics with
// the given severity.
func (c Collection) FilterBySeverity(sev Severity) Collection {
	return c.filter(func(d Diagnostic) bool { return d.Severity == sev })
}

// FilterBySource returns a new collection containing only diagnostics whose
// Source equals source.
func (c Collection) FilterBySource(source string) Collection {
	return c.filter(func(d Diagnostic) bool { return d.Source == source })
}

// FilterByCodePrefix returns a new collection containing only diagnostics whose
// code begins with prefix, for example "BWCFG".
func (c Collection) FilterByCodePrefix(prefix string) Collection {
	return c.filter(func(d Diagnostic) bool { return strings.HasPrefix(string(d.Code), prefix) })
}

// filter returns a new collection of diagnostics matching keep.
func (c Collection) filter(keep func(Diagnostic) bool) Collection {
	var out []Diagnostic
	for i := range c.diags {
		if keep(c.diags[i]) {
			out = append(out, c.diags[i])
		}
	}
	return Collection{diags: out}
}

// lessDiagnostic implements the documented sort order.
func lessDiagnostic(a, b Diagnostic) bool {
	as, bs := a.Range.Start, b.Range.Start
	if as.File != bs.File {
		return as.File < bs.File
	}
	if as.Line != bs.Line {
		return as.Line < bs.Line
	}
	if as.Column != bs.Column {
		return as.Column < bs.Column
	}
	if wa, wb := a.Severity.sortWeight(), b.Severity.sortWeight(); wa != wb {
		return wa < wb
	}
	if a.Code != b.Code {
		return a.Code < b.Code
	}
	return a.Message < b.Message
}
