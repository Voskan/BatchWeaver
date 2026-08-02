package documents

import (
	"unicode/utf16"
	"unicode/utf8"

	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

// Mapper converts between LSP positions (zero-based line, UTF-16 code-unit
// character offset) and byte offsets for one document content. It is the single
// canonical coordinate mapper used across the server, so every feature agrees.
// A Mapper is immutable and safe for concurrent reads.
type Mapper struct {
	content []byte
	// lineStarts[i] is the byte offset at which line i begins. There is always at
	// least one entry (0).
	lineStarts []int
}

// NewMapper builds a Mapper over content.
func NewMapper(content []byte) *Mapper {
	starts := []int{0}
	for i, b := range content {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return &Mapper{content: content, lineStarts: starts}
}

// Content returns the mapped bytes.
func (m *Mapper) Content() []byte { return m.content }

// LineCount returns the number of lines.
func (m *Mapper) LineCount() int { return len(m.lineStarts) }

// lineBytes returns the byte slice of a line excluding the trailing newline.
func (m *Mapper) lineBytes(line int) []byte {
	if line < 0 || line >= len(m.lineStarts) {
		return nil
	}
	start := m.lineStarts[line]
	end := len(m.content)
	if line+1 < len(m.lineStarts) {
		end = m.lineStarts[line+1]
	}
	seg := m.content[start:end]
	// Trim the line terminator so a character offset never lands inside it.
	if n := len(seg); n > 0 && seg[n-1] == '\n' {
		seg = seg[:n-1]
		if n := len(seg); n > 0 && seg[n-1] == '\r' {
			seg = seg[:n-1]
		}
	}
	return seg
}

// OffsetToPosition converts a byte offset to an LSP position, clamping to the
// document bounds so an out-of-range offset never panics.
func (m *Mapper) OffsetToPosition(offset int) protocol.Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(m.content) {
		offset = len(m.content)
	}
	// Binary-free linear-ish search is fine; find the last line start <= offset.
	line := 0
	for line+1 < len(m.lineStarts) && m.lineStarts[line+1] <= offset {
		line++
	}
	col := utf16Len(m.content[m.lineStarts[line]:offset])
	return protocol.Position{Line: uint32(line), Character: uint32(col)}
}

// PositionToOffset converts an LSP position to a byte offset, clamping to line
// and content bounds.
func (m *Mapper) PositionToOffset(pos protocol.Position) int {
	line := int(pos.Line)
	if line < 0 {
		return 0
	}
	if line >= len(m.lineStarts) {
		return len(m.content)
	}
	start := m.lineStarts[line]
	seg := m.lineBytes(line)
	byteInLine := utf16OffsetToByte(seg, int(pos.Character))
	return start + byteInLine
}

// RangeToByteRange converts an LSP range to a start/end byte offset pair.
func (m *Mapper) RangeToByteRange(r protocol.Range) (int, int) {
	return m.PositionToOffset(r.Start), m.PositionToOffset(r.End)
}

// ByteRangeToRange converts a start/end byte offset pair to an LSP range.
func (m *Mapper) ByteRangeToRange(start, end int) protocol.Range {
	return protocol.Range{Start: m.OffsetToPosition(start), End: m.OffsetToPosition(end)}
}

// LineByteColToOffset converts a 1-based line and 1-based byte column (the form
// go/token reports) to a byte offset, clamping to line and content bounds.
func (m *Mapper) LineByteColToOffset(line1, col1 int) int {
	idx := line1 - 1
	if idx < 0 {
		return 0
	}
	if idx >= len(m.lineStarts) {
		return len(m.content)
	}
	off := m.lineStarts[idx] + (col1 - 1)
	if col1 < 1 {
		off = m.lineStarts[idx]
	}
	if off > len(m.content) {
		off = len(m.content)
	}
	return off
}

// utf16Len returns the number of UTF-16 code units in b.
func utf16Len(b []byte) int {
	n := 0
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size == 1 {
			n++ // invalid byte counts as one unit
			b = b[1:]
			continue
		}
		if r > 0xFFFF {
			n += 2 // surrogate pair
		} else {
			n++
		}
		b = b[size:]
	}
	return n
}

// utf16OffsetToByte returns the byte index in seg corresponding to the given
// UTF-16 code-unit offset, clamping to the end of seg.
func utf16OffsetToByte(seg []byte, u16 int) int {
	if u16 <= 0 {
		return 0
	}
	units := 0
	byteIdx := 0
	for byteIdx < len(seg) {
		r, size := utf8.DecodeRune(seg[byteIdx:])
		w := 1
		if r > 0xFFFF {
			w = 2
		}
		if units+w > u16 {
			break
		}
		units += w
		byteIdx += size
		if units == u16 {
			break
		}
	}
	return byteIdx
}

// EncodeUTF16 is exported for tests: it returns the UTF-16 encoding length of s.
func EncodeUTF16(s string) int { return len(utf16.Encode([]rune(s))) }
