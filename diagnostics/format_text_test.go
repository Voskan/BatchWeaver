package diagnostics

import (
	"bytes"
	"strings"
	"testing"
)

func TestTextFormatterGolden(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := NewTextFormatter().Format(&buf, sampleCollection()); err != nil {
		t.Fatalf("format: %v", err)
	}
	checkGolden(t, "diagnostics.txt", buf.Bytes())
}

func TestTextFormatterDeterministic(t *testing.T) {
	t.Parallel()
	var a, b bytes.Buffer
	f := NewTextFormatter()
	if err := f.Format(&a, sampleCollection()); err != nil {
		t.Fatal(err)
	}
	if err := f.Format(&b, sampleCollection()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Errorf("text output not deterministic")
	}
}

func TestTextFormatterNoColor(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := (TextFormatter{Color: true}).Format(&buf, sampleCollection()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("output contains ANSI escape codes despite no-color policy")
	}
}
