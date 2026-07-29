package operation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"

	"github.com/Voskan/BatchWeaver/diagnostics"
)

// codeDuplicateID reports a duplicate operation ID within a catalog.
const codeDuplicateID diagnostics.Code = "BWOP017"

// ConflictBehavior selects how catalog merges resolve duplicate IDs.
type ConflictBehavior uint8

const (
	// ConflictError treats a duplicate ID as an error.
	ConflictError ConflictBehavior = iota
	// ConflictReplace lets the incoming spec replace the existing one.
	ConflictReplace
	// ConflictKeepExisting keeps the existing spec and ignores the incoming one.
	ConflictKeepExisting
)

// Catalog is an immutable-by-convention, deterministic set of operation specs
// keyed by ID. Construct it with a CatalogBuilder or Merge; read methods return
// copies and never expose internal storage.
type Catalog struct {
	specs []Spec // sorted by ID
	index map[ID]int
}

// CatalogBuilder accumulates specs and records a diagnostic for each duplicate
// ID, including the previous declaration's source location when available.
type CatalogBuilder struct {
	specs []Spec
	index map[ID]int
	diags diagnostics.Collection
}

// NewCatalogBuilder returns an empty CatalogBuilder.
func NewCatalogBuilder() *CatalogBuilder {
	return &CatalogBuilder{index: make(map[ID]int)}
}

// Add records a spec. If its ID already exists, Add emits a duplicate-ID
// diagnostic (with related information pointing at the previous declaration) and
// keeps the first declaration.
func (b *CatalogBuilder) Add(spec Spec) {
	if prev, ok := b.index[spec.id]; ok {
		d := diagnostics.Diagnostic{
			Code:     codeDuplicateID,
			Severity: diagnostics.SeverityError,
			Message:  fmt.Sprintf("duplicate operation id %q", spec.id),
			Source:   operationSource,
			Range:    spec.source.Range,
		}
		if prevRange := b.specs[prev].source.Range; !prevRange.IsZero() {
			d.Related = []diagnostics.RelatedInformation{{
				Message: "previous declaration",
				Range:   prevRange,
			}}
		}
		b.diags.Add(d)
		return
	}
	b.index[spec.id] = len(b.specs)
	b.specs = append(b.specs, spec)
}

// Diagnostics returns the diagnostics accumulated during building (duplicates)
// combined with the validation diagnostics of every added spec.
func (b *CatalogBuilder) Diagnostics() diagnostics.Collection {
	var c diagnostics.Collection
	c.AddCollection(b.diags)
	for i := range b.specs {
		c.AddCollection(b.specs[i].Validate())
	}
	return c
}

// Build returns the catalog of successfully added specs, sorted by ID.
func (b *CatalogBuilder) Build() Catalog {
	return newCatalogFromSpecs(b.specs)
}

// newCatalogFromSpecs builds a sorted, indexed catalog from specs.
func newCatalogFromSpecs(specs []Spec) Catalog {
	out := make([]Spec, len(specs))
	copy(out, specs)
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	index := make(map[ID]int, len(out))
	for i := range out {
		index[out[i].id] = i
	}
	return Catalog{specs: out, index: index}
}

// Get returns the spec with the given ID.
func (c Catalog) Get(id ID) (Spec, bool) {
	i, ok := c.index[id]
	if !ok {
		return Spec{}, false
	}
	return c.specs[i], true
}

// Len returns the number of specs.
func (c Catalog) Len() int { return len(c.specs) }

// List returns the specs in lexicographic ID order.
func (c Catalog) List() []Spec {
	out := make([]Spec, len(c.specs))
	copy(out, c.specs)
	return out
}

// IDs returns the operation IDs in lexicographic order.
func (c Catalog) IDs() []ID {
	out := make([]ID, len(c.specs))
	for i := range c.specs {
		out[i] = c.specs[i].id
	}
	return out
}

// Validate returns the combined validation diagnostics of every spec.
func (c Catalog) Validate() diagnostics.Collection {
	var out diagnostics.Collection
	for i := range c.specs {
		out.AddCollection(c.specs[i].Validate())
	}
	return out
}

// Digest returns a deterministic SHA-256 digest of the catalog's semantic
// content, formatted as "sha256:<lowercase-hex>". It uses the canonical
// encoding of each spec in ID order and excludes source positions.
func (c Catalog) Digest() string {
	h := sha256.New()
	for _, sp := range c.specs { // already sorted by ID
		_, _ = io.WriteString(h, sp.Canonical())
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// Merge returns a new catalog combining c and other. Duplicate IDs are resolved
// according to behavior; ConflictError returns an error naming the first
// conflicting ID.
func (c Catalog) Merge(other Catalog, behavior ConflictBehavior) (Catalog, error) {
	merged := make([]Spec, 0, len(c.specs)+len(other.specs))
	seen := make(map[ID]int)
	appendSpec := func(sp Spec, incoming bool) error {
		if idx, ok := seen[sp.id]; ok {
			switch behavior {
			case ConflictError:
				return fmt.Errorf("catalog merge conflict on operation id %q", sp.id)
			case ConflictReplace:
				if incoming {
					merged[idx] = sp
				}
			case ConflictKeepExisting:
				// keep the existing spec
			}
			return nil
		}
		seen[sp.id] = len(merged)
		merged = append(merged, sp)
		return nil
	}
	for _, sp := range c.specs {
		if err := appendSpec(sp, false); err != nil {
			return Catalog{}, err
		}
	}
	for _, sp := range other.specs {
		if err := appendSpec(sp, true); err != nil {
			return Catalog{}, err
		}
	}
	return newCatalogFromSpecs(merged), nil
}
