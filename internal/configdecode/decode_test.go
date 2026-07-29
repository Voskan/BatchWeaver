package configdecode

import (
	"testing"

	"github.com/Voskan/BatchWeaver/diagnostics"
)

func TestParseYAMLPositions(t *testing.T) {
	t.Parallel()
	node, diags := ParseYAML("x.yaml", []byte("version: 1\nname: hello\n"))
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %v", diags.Diagnostics())
	}
	v, _, ok := node.Get("version")
	if !ok || v.ScalarType != ScalarInt || v.Value != "1" {
		t.Errorf("version node = %+v", v)
	}
	if v.Pos.Line != 1 {
		t.Errorf("version position line = %d, want 1", v.Pos.Line)
	}
	n, _, _ := node.Get("name")
	if n.Pos.Line != 2 {
		t.Errorf("name position line = %d, want 2", n.Pos.Line)
	}
}

func TestParseJSONPositionsAndTypes(t *testing.T) {
	t.Parallel()
	src := []byte("{\n  \"version\": 1,\n  \"flag\": true,\n  \"list\": [\"a\", \"b\"]\n}")
	node, diags := ParseJSON("x.json", src)
	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %v", diags.Diagnostics())
	}
	v, _, _ := node.Get("version")
	if got, ok := AsInt(v); !ok || got != 1 {
		t.Errorf("version = %v %v", got, ok)
	}
	f, _, _ := node.Get("flag")
	if got, ok := AsBool(f); !ok || !got {
		t.Errorf("flag = %v %v", got, ok)
	}
	l, _, _ := node.Get("list")
	if !l.IsSequence() || len(l.Elems) != 2 {
		t.Errorf("list = %+v", l)
	}
}

func TestParseJSONDuplicateKey(t *testing.T) {
	t.Parallel()
	_, diags := ParseJSON("x.json", []byte(`{"a":1,"a":2}`))
	if !hasCode(diags, CodeDuplicateKey) {
		t.Errorf("expected duplicate-key diagnostic")
	}
}

func TestParseJSONTrailingContent(t *testing.T) {
	t.Parallel()
	_, diags := ParseJSON("x.json", []byte(`{"a":1} garbage`))
	if !hasCode(diags, CodeTrailingContent) {
		t.Errorf("expected trailing-content diagnostic")
	}
}

func TestParseYAMLRejectsAnchors(t *testing.T) {
	t.Parallel()
	src := []byte("a: &anchor value\nb: *anchor\n")
	_, diags := ParseYAML("x.yaml", src)
	if !hasCode(diags, CodeUnsupportedConstruct) && !hasCode(diags, CodeSyntax) {
		t.Errorf("expected anchors to be rejected; got %v", diags.Diagnostics())
	}
}

func TestParseYAMLMultipleDocuments(t *testing.T) {
	t.Parallel()
	src := []byte("a: 1\n---\nb: 2\n")
	_, diags := ParseYAML("x.yaml", src)
	if !hasCode(diags, CodeMultipleDocuments) {
		t.Errorf("expected multiple-documents diagnostic; got %v", diags.Diagnostics())
	}
}

func hasCode(c diagnostics.Collection, code diagnostics.Code) bool {
	for _, d := range c.Diagnostics() {
		if d.Code == code {
			return true
		}
	}
	return false
}

func FuzzParseYAML(f *testing.F) {
	for _, s := range []string{"version: 1", "a: [1, 2]", "", "{bad", "a: &x 1\nb: *x"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic regardless of input.
		_, _ = ParseYAML("fuzz.yaml", []byte(s))
	})
}

func FuzzParseJSON(f *testing.F) {
	for _, s := range []string{`{"a":1}`, `[1,2,3]`, `"x"`, ``, `{bad`, `{"a":1,"a":2}`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ParseJSON("fuzz.json", []byte(s))
	})
}
