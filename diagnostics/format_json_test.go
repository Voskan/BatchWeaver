package diagnostics

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestJSONFormatterGolden(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := NewJSONFormatter().Format(&buf, sampleCollection()); err != nil {
		t.Fatalf("format: %v", err)
	}
	checkGolden(t, "diagnostics.json", buf.Bytes())
}

func TestJSONFormatterValidJSONAndTrailingNewline(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := NewJSONFormatter().Format(&buf, sampleCollection()); err != nil {
		t.Fatal(err)
	}
	out := buf.Bytes()
	if len(out) == 0 || out[len(out)-1] != '\n' {
		t.Errorf("JSON output must end with a newline")
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if doc["schema_version"].(float64) != float64(JSONSchemaVersion) {
		t.Errorf("schema_version = %v, want %d", doc["schema_version"], JSONSchemaVersion)
	}
}

func TestJSONFormatterDeterministic(t *testing.T) {
	t.Parallel()
	var a, b bytes.Buffer
	f := NewJSONFormatter()
	if err := f.Format(&a, sampleCollection()); err != nil {
		t.Fatal(err)
	}
	if err := f.Format(&b, sampleCollection()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Errorf("JSON output not deterministic")
	}
}

func TestJSONFormatterNoHTMLEscaping(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := NewJSONFormatter().Format(&buf, sampleCollection()); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf.Bytes(), []byte(`&`)) || bytes.Contains(buf.Bytes(), []byte(`<`)) {
		t.Errorf("output has HTML-escaped characters")
	}
}
