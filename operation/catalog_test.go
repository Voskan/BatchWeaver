package operation

import (
	"testing"

	"github.com/Voskan/BatchWeaver/diagnostics"
)

func sampleSpec(t *testing.T, id string) Spec {
	t.Helper()
	return MustNewSpec(MustParseID(id), ReadOnly(), WithOrderedResults(), WithRequestScope())
}

func TestCatalogBuilderAndList(t *testing.T) {
	t.Parallel()
	b := NewCatalogBuilder()
	b.Add(sampleSpec(t, "b.two"))
	b.Add(sampleSpec(t, "a.one"))
	cat := b.Build()
	if cat.Len() != 2 {
		t.Fatalf("Len = %d", cat.Len())
	}
	ids := cat.IDs()
	if ids[0] != "a.one" || ids[1] != "b.two" {
		t.Errorf("IDs not sorted: %v", ids)
	}
	if _, ok := cat.Get("a.one"); !ok {
		t.Errorf("Get(a.one) missing")
	}
}

func TestCatalogBuilderDuplicate(t *testing.T) {
	t.Parallel()
	b := NewCatalogBuilder()
	b.Add(MustNewSpec(MustParseID("users.get"), ReadOnly(),
		WithSource(Source{Range: diagnostics.Range{Start: diagnostics.Position{File: "a.yaml", Line: 1, Column: 1}}})))
	b.Add(MustNewSpec(MustParseID("users.get"), ReadOnly(),
		WithSource(Source{Range: diagnostics.Range{Start: diagnostics.Position{File: "b.yaml", Line: 2, Column: 1}}})))
	diags := b.Diagnostics()
	if !containsCode(diags, "BWOP017") {
		t.Fatalf("expected duplicate diagnostic BWOP017; got %v", diags.Diagnostics())
	}
	// Related information should point at the previous declaration.
	found := false
	for _, d := range diags.Diagnostics() {
		if d.Code == "BWOP017" && len(d.Related) == 1 && d.Related[0].Range.Start.File == "a.yaml" {
			found = true
		}
	}
	if !found {
		t.Errorf("duplicate diagnostic missing related previous location")
	}
	if b.Build().Len() != 1 {
		t.Errorf("duplicate should keep only the first spec")
	}
}

func TestCatalogDigestDeterministicAndSensitive(t *testing.T) {
	t.Parallel()
	build := func(ids ...string) Catalog {
		b := NewCatalogBuilder()
		for _, id := range ids {
			b.Add(sampleSpec(t, id))
		}
		return b.Build()
	}
	// Insertion order must not change the digest.
	d1 := build("a.one", "b.two").Digest()
	d2 := build("b.two", "a.one").Digest()
	if d1 != d2 {
		t.Errorf("digest depends on insertion order: %s vs %s", d1, d2)
	}
	// A different set changes the digest.
	if build("a.one").Digest() == d1 {
		t.Errorf("digest did not change with different content")
	}
	if len(d1) < len("sha256:") || d1[:7] != "sha256:" {
		t.Errorf("digest not prefixed: %s", d1)
	}
}

func TestCatalogMerge(t *testing.T) {
	t.Parallel()
	a := func() Catalog { b := NewCatalogBuilder(); b.Add(sampleSpec(t, "a.one")); return b.Build() }()
	b := func() Catalog { b := NewCatalogBuilder(); b.Add(sampleSpec(t, "b.two")); return b.Build() }()
	merged, err := a.Merge(b, ConflictError)
	if err != nil || merged.Len() != 2 {
		t.Fatalf("merge disjoint: len=%d err=%v", merged.Len(), err)
	}

	dup := func() Catalog { b := NewCatalogBuilder(); b.Add(sampleSpec(t, "a.one")); return b.Build() }()
	if _, err := a.Merge(dup, ConflictError); err == nil {
		t.Errorf("ConflictError merge should fail on duplicate")
	}
	if m, err := a.Merge(dup, ConflictKeepExisting); err != nil || m.Len() != 1 {
		t.Errorf("ConflictKeepExisting: len=%d err=%v", m.Len(), err)
	}
}
