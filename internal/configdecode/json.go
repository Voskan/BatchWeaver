package configdecode

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/Voskan/BatchWeaver/diagnostics"
)

// ParseJSON parses JSON bytes into a node tree with accurate positions. It is a
// small, dependency-free recursive-descent parser so that JSON gains the same
// position-aware, duplicate-key-rejecting, single-document semantics as YAML. It
// rejects duplicate object keys and trailing content.
func ParseJSON(file string, src []byte) (*Node, diagnostics.Collection) {
	p := &jsonParser{file: file, src: src, line: 1, col: 1}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(jsonAbort); !ok {
				panic(r)
			}
		}
	}()
	p.skipWS()
	node := p.parseValue()
	if !p.aborted {
		p.skipWS()
		if p.i < len(p.src) {
			p.diags.Add(diag(CodeTrailingContent, p.pos(), "unexpected trailing content after JSON document"))
			return nil, p.diags
		}
	}
	if p.diags.HasErrors() {
		return nil, p.diags
	}
	return node, p.diags
}

// jsonAbort is panicked to unwind on a fatal syntax error.
type jsonAbort struct{}

type jsonParser struct {
	file    string
	src     []byte
	i       int
	line    int
	col     int
	diags   diagnostics.Collection
	aborted bool
}

func (p *jsonParser) pos() diagnostics.Position {
	return diagnostics.Position{File: p.file, Offset: p.i, Line: p.line, Column: p.col}
}

func (p *jsonParser) fail(msg string) {
	p.diags.Add(diag(CodeSyntax, p.pos(), "JSON syntax error: "+msg))
	p.aborted = true
	panic(jsonAbort{})
}

func (p *jsonParser) advance() byte {
	c := p.src[p.i]
	p.i++
	if c == '\n' {
		p.line++
		p.col = 1
	} else {
		p.col++
	}
	return c
}

func (p *jsonParser) skipWS() {
	for p.i < len(p.src) {
		switch p.src[p.i] {
		case ' ', '\t', '\r', '\n':
			p.advance()
		default:
			return
		}
	}
}

func (p *jsonParser) peek() (byte, bool) {
	if p.i >= len(p.src) {
		return 0, false
	}
	return p.src[p.i], true
}

func (p *jsonParser) parseValue() *Node {
	c, ok := p.peek()
	if !ok {
		p.fail("unexpected end of input")
	}
	switch {
	case c == '{':
		return p.parseObject()
	case c == '[':
		return p.parseArray()
	case c == '"':
		start := p.pos()
		s := p.parseString()
		return &Node{Kind: KindScalar, ScalarType: ScalarString, Value: s, Pos: start}
	case c == '-' || (c >= '0' && c <= '9'):
		return p.parseNumber()
	case c == 't' || c == 'f':
		return p.parseLiteral()
	case c == 'n':
		return p.parseNull()
	default:
		p.fail(fmt.Sprintf("unexpected character %q", string(c)))
		return nil
	}
}

func (p *jsonParser) parseObject() *Node {
	start := p.pos()
	p.advance() // consume '{'
	out := &Node{Kind: KindMapping, Pos: start}
	seen := make(map[string]diagnostics.Position)
	p.skipWS()
	if c, ok := p.peek(); ok && c == '}' {
		p.advance()
		return out
	}
	for {
		p.skipWS()
		if c, ok := p.peek(); !ok || c != '"' {
			p.fail("expected string key")
		}
		keyPos := p.pos()
		key := p.parseString()
		p.skipWS()
		if c, ok := p.peek(); !ok || c != ':' {
			p.fail("expected ':' after key")
		}
		p.advance() // consume ':'
		p.skipWS()
		val := p.parseValue()
		if prev, dup := seen[key]; dup {
			d := diag(CodeDuplicateKey, keyPos, fmt.Sprintf("duplicate object key %q", key))
			d.Related = []diagnostics.RelatedInformation{{Message: "first defined here", Range: diagnostics.AtPosition(prev)}}
			p.diags.Add(d)
		} else {
			seen[key] = keyPos
			out.Entries = append(out.Entries, MapEntry{Key: key, KeyPos: keyPos, Value: val})
		}
		p.skipWS()
		c, ok := p.peek()
		if !ok {
			p.fail("unterminated object")
		}
		if c == ',' {
			p.advance()
			continue
		}
		if c == '}' {
			p.advance()
			return out
		}
		p.fail("expected ',' or '}'")
	}
}

func (p *jsonParser) parseArray() *Node {
	start := p.pos()
	p.advance() // consume '['
	out := &Node{Kind: KindSequence, Pos: start}
	p.skipWS()
	if c, ok := p.peek(); ok && c == ']' {
		p.advance()
		return out
	}
	for {
		p.skipWS()
		out.Elems = append(out.Elems, p.parseValue())
		p.skipWS()
		c, ok := p.peek()
		if !ok {
			p.fail("unterminated array")
		}
		if c == ',' {
			p.advance()
			continue
		}
		if c == ']' {
			p.advance()
			return out
		}
		p.fail("expected ',' or ']'")
	}
}

func (p *jsonParser) parseString() string {
	p.advance() // consume opening quote
	var b strings.Builder
	for {
		if p.i >= len(p.src) {
			p.fail("unterminated string")
		}
		c := p.advance()
		switch {
		case c == '"':
			return b.String()
		case c == '\\':
			p.parseEscape(&b)
		case c < 0x20:
			p.fail("control character in string")
		default:
			b.WriteByte(c)
		}
	}
}

func (p *jsonParser) parseEscape(b *strings.Builder) {
	if p.i >= len(p.src) {
		p.fail("unterminated escape")
	}
	c := p.advance()
	switch c {
	case '"', '\\', '/':
		b.WriteByte(c)
	case 'b':
		b.WriteByte('\b')
	case 'f':
		b.WriteByte('\f')
	case 'n':
		b.WriteByte('\n')
	case 'r':
		b.WriteByte('\r')
	case 't':
		b.WriteByte('\t')
	case 'u':
		b.WriteRune(p.parseUnicodeEscape())
	default:
		p.fail("invalid escape sequence")
	}
}

func (p *jsonParser) parseUnicodeEscape() rune {
	r := p.readHex4()
	if utf16.IsSurrogate(rune(r)) {
		if p.i+1 < len(p.src) && p.src[p.i] == '\\' && p.src[p.i+1] == 'u' {
			p.advance()
			p.advance()
			r2 := p.readHex4()
			if dec := utf16.DecodeRune(rune(r), rune(r2)); dec != utf8.RuneError {
				return dec
			}
		}
		return utf8.RuneError
	}
	return rune(r)
}

func (p *jsonParser) readHex4() int {
	if p.i+4 > len(p.src) {
		p.fail("invalid \\u escape")
	}
	v, err := strconv.ParseUint(string(p.src[p.i:p.i+4]), 16, 32)
	if err != nil {
		p.fail("invalid \\u escape")
	}
	for k := 0; k < 4; k++ {
		p.advance()
	}
	return int(v)
}

func (p *jsonParser) parseNumber() *Node {
	start := p.pos()
	begin := p.i
	isFloat := false
	for p.i < len(p.src) {
		c := p.src[p.i]
		if c >= '0' && c <= '9' || c == '-' || c == '+' {
			p.advance()
			continue
		}
		if c == '.' || c == 'e' || c == 'E' {
			isFloat = true
			p.advance()
			continue
		}
		break
	}
	text := string(p.src[begin:p.i])
	st := ScalarInt
	if isFloat {
		st = ScalarFloat
		if _, err := strconv.ParseFloat(text, 64); err != nil {
			p.fail("invalid number")
		}
	} else if _, err := strconv.ParseInt(text, 10, 64); err != nil {
		p.fail("invalid number")
	}
	return &Node{Kind: KindScalar, ScalarType: st, Value: text, Pos: start}
}

func (p *jsonParser) parseLiteral() *Node {
	start := p.pos()
	if p.consume("true") {
		return &Node{Kind: KindScalar, ScalarType: ScalarBool, Value: "true", Pos: start}
	}
	if p.consume("false") {
		return &Node{Kind: KindScalar, ScalarType: ScalarBool, Value: "false", Pos: start}
	}
	p.fail("invalid literal")
	return nil
}

func (p *jsonParser) parseNull() *Node {
	start := p.pos()
	if p.consume("null") {
		return &Node{Kind: KindScalar, ScalarType: ScalarNull, Value: "null", Pos: start}
	}
	p.fail("invalid literal")
	return nil
}

func (p *jsonParser) consume(word string) bool {
	if p.i+len(word) > len(p.src) || string(p.src[p.i:p.i+len(word)]) != word {
		return false
	}
	for k := 0; k < len(word); k++ {
		p.advance()
	}
	return true
}
