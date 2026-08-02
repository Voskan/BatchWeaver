package documents

import (
	"path/filepath"
	"testing"

	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

func TestUTF16Mapping(t *testing.T) {
	// Line 0: ASCII. Line 1: multibyte (é is 2 bytes, 1 UTF-16 unit).
	// Line 2: emoji (😀 is 4 bytes, 2 UTF-16 units).
	content := []byte("abc\ncafé x\n😀y")
	m := NewMapper(content)

	// Character after "café " on line 1 (UTF-16 char 5 -> the 'x').
	off := m.PositionToOffset(protocol.Position{Line: 1, Character: 5})
	if content[off] != 'x' {
		t.Errorf("line1 char5 -> %q, want x", content[off])
	}
	// Round trip.
	pos := m.OffsetToPosition(off)
	if pos.Line != 1 || pos.Character != 5 {
		t.Errorf("round trip = %+v, want {1,5}", pos)
	}

	// Emoji occupies 2 UTF-16 units; the 'y' after it is char 2 on line 2.
	off = m.PositionToOffset(protocol.Position{Line: 2, Character: 2})
	if content[off] != 'y' {
		t.Errorf("line2 char2 -> %q, want y", content[off])
	}
}

func TestMapperCRLF(t *testing.T) {
	m := NewMapper([]byte("a\r\nb"))
	off := m.PositionToOffset(protocol.Position{Line: 1, Character: 0})
	if m.Content()[off] != 'b' {
		t.Errorf("CRLF line start wrong: %q", m.Content()[off])
	}
}

func TestStoreIncrementalChangeAndVersion(t *testing.T) {
	s := NewStore()
	uri := PathToURI(filepath.Join(t.TempDir(), "x.go"))
	if _, err := s.Open(protocol.TextDocumentItem{URI: uri, Version: 1, Text: "hello world"}); err != nil {
		t.Fatal(err)
	}
	// Replace "world" with "gophers".
	rng := &protocol.Range{Start: protocol.Position{Line: 0, Character: 6}, End: protocol.Position{Line: 0, Character: 11}}
	d, err := s.Change(protocol.VersionedTextDocumentIdentifier{URI: uri, Version: 2},
		[]protocol.TextDocumentContentChangeEvent{{Range: rng, Text: "gophers"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(d.Content) != "hello gophers" {
		t.Errorf("content = %q", d.Content)
	}
	// Out-of-order version is rejected.
	if _, err := s.Change(protocol.VersionedTextDocumentIdentifier{URI: uri, Version: 1}, nil); err == nil {
		t.Error("expected out-of-order version rejection")
	}
}

func TestStoreOverlay(t *testing.T) {
	s := NewStore()
	uri := PathToURI(filepath.Join(t.TempDir(), "y.go"))
	if _, err := s.Open(protocol.TextDocumentItem{URI: uri, Version: 1, Text: "package p"}); err != nil {
		t.Fatal(err)
	}
	ov := s.Overlay()
	if len(ov) != 1 {
		t.Fatalf("overlay size = %d", len(ov))
	}
	for path, content := range ov {
		if string(content) != "package p" {
			t.Errorf("overlay content = %q for %s", content, path)
		}
	}
	s.Close(uri)
	if len(s.Overlay()) != 0 {
		t.Error("overlay should be empty after close")
	}
}

func TestURIRoundTrip(t *testing.T) {
	want := filepath.Join(t.TempDir(), "a b", "c.go")
	uri := PathToURI(want)
	path, err := URIToPath(uri)
	if err != nil {
		t.Fatal(err)
	}
	if path != want {
		t.Errorf("round trip = %q, want %q", path, want)
	}
	if _, err := URIToPath("http://example.com"); err == nil {
		t.Error("non-file URI should be rejected")
	}
}
