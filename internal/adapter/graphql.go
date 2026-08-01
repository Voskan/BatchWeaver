package adapter

import (
	"sort"
	"strings"
)

// GraphQL diagnostic codes (BW71xx). The BW7xxx range is reserved for network
// protocol adapters, distinct from backend adapters (BW6xxx).
const (
	CodeGraphQLBindingMissing     = "BW7101"
	CodeGraphQLSelectionPartition = "BW7102"
	CodeGraphQLDirectiveBarrier   = "BW7103"
	CodeGraphQLAuthPartition      = "BW7104"
	CodeGraphQLNullability        = "BW7105"
	CodeGraphQLParse              = "BW7106"
	CodeGraphQLSubscriptionScope  = "BW7107"
)

// OperationType is a GraphQL operation type.
type OperationType string

// GraphQL operation types.
const (
	OpQuery        OperationType = "query"
	OpMutation     OperationType = "mutation"
	OpSubscription OperationType = "subscription"
)

// GraphQLField is a field selection in the framework-neutral model.
type GraphQLField struct {
	Alias      string
	Name       string
	Directives []string
	Spreads    []string           // fragment spread names
	Inlines    []GraphQLSelection // inline fragment selection sets
	Sel        *GraphQLSelection
}

// ResponseName returns the field's response key (alias if present, else name).
func (f GraphQLField) ResponseName() string {
	if f.Alias != "" {
		return f.Alias
	}
	return f.Name
}

// GraphQLSelection is a selection set.
type GraphQLSelection struct {
	Fields []GraphQLField
}

// GraphQLOperation is a parsed operation.
type GraphQLOperation struct {
	Type      OperationType
	Name      string
	Selection GraphQLSelection
}

// GraphQLDocument is a parsed GraphQL document.
type GraphQLDocument struct {
	Operations []GraphQLOperation
	Fragments  map[string]GraphQLSelection
}

// ResolverWave is one execution wave: field response paths ready at the same
// dependency frontier.
type ResolverWave struct {
	Depth  int
	Fields []string
}

// ParseGraphQL parses a GraphQL document using a recursive-descent parser (no
// regex). It supports named and anonymous operations, query/mutation/subscription,
// aliases, arguments, directives, fragment definitions, fragment spreads, and
// inline fragments — the subset needed for resolver-wave analysis.
func ParseGraphQL(src string) (*GraphQLDocument, *Rejection) {
	toks, rej := gqlTokenize(src)
	if rej != nil {
		return nil, rej
	}
	p := &gqlParser{toks: toks}
	return p.parseDocument()
}

// ResolverWaves computes execution waves for an operation by breadth. Wave 0 is
// the operation's top-level fields; each subsequent wave holds the fields of the
// previous wave's selection sets. Fragment spreads and inline fragments are
// expanded. Response paths use aliases.
func ResolverWaves(doc *GraphQLDocument, op GraphQLOperation) []ResolverWave {
	var waves []ResolverWave
	type node struct {
		path string
		sel  GraphQLSelection
	}
	frontier := []node{{path: "", sel: op.Selection}}
	depth := 0
	for len(frontier) > 0 {
		var names []string
		var next []node
		for _, n := range frontier {
			for _, f := range expandFields(doc, n.sel) {
				rp := f.ResponseName()
				full := rp
				if n.path != "" {
					full = n.path + "." + rp
				}
				names = append(names, full)
				if f.Sel != nil && len(f.Sel.Fields) > 0 {
					next = append(next, node{path: full, sel: *f.Sel})
				}
			}
		}
		sort.Strings(names)
		if len(names) > 0 {
			waves = append(waves, ResolverWave{Depth: depth, Fields: names})
		}
		frontier = next
		depth++
	}
	return waves
}

// expandFields returns the direct fields of a selection with fragment spreads and
// inline fragments expanded (one level; nested handled by recursion during wave
// walking).
func expandFields(doc *GraphQLDocument, sel GraphQLSelection) []GraphQLField {
	var out []GraphQLField
	out = append(out, sel.Fields...)
	for _, f := range sel.Fields {
		for _, name := range f.Spreads {
			if frag, ok := doc.Fragments[name]; ok {
				out = append(out, frag.Fields...)
			}
		}
		for _, inl := range f.Inlines {
			out = append(out, inl.Fields...)
		}
	}
	// Spreads/inlines attached directly at this selection level are modeled as
	// fields with empty Name and populated Spreads/Inlines.
	var filtered []GraphQLField
	for _, f := range out {
		if f.Name == "" {
			for _, name := range f.Spreads {
				if frag, ok := doc.Fragments[name]; ok {
					filtered = append(filtered, frag.Fields...)
				}
			}
			for _, inl := range f.Inlines {
				filtered = append(filtered, inl.Fields...)
			}
			continue
		}
		filtered = append(filtered, f)
	}
	return filtered
}

// NormalizeSelectionDigest returns a deterministic digest of a field's normalized
// sub-selection (alias-independent, sorted), used to partition resolver calls
// that share a key but require different sub-selections.
func NormalizeSelectionDigest(doc *GraphQLDocument, f GraphQLField) string {
	if f.Sel == nil {
		return "sha:leaf"
	}
	var names []string
	for _, sf := range expandFields(doc, *f.Sel) {
		names = append(names, sf.Name+NormalizeSelectionDigest(doc, sf))
	}
	sort.Strings(names)
	return "{" + strings.Join(names, ",") + "}"
}

// --- GraphQL tokenizer and parser ---

type gqlTok struct {
	kind string // name, punct, string, var, eof
	text string
	off  int
}

func gqlTokenize(s string) ([]gqlTok, *Rejection) {
	var toks []gqlTok
	i, n := 0, len(s)
	for i < n {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ',':
			i++
		case c == '#':
			for i < n && s[i] != '\n' {
				i++
			}
		case c == '{' || c == '}' || c == '(' || c == ')' || c == ':' || c == '[' || c == ']' || c == '=' || c == '@' || c == '!':
			toks = append(toks, gqlTok{"punct", string(c), i})
			i++
		case c == '.' && i+2 < n && s[i+1] == '.' && s[i+2] == '.':
			toks = append(toks, gqlTok{"punct", "...", i})
			i += 3
		case c == '$':
			j := i + 1
			for j < n && isGQLNamePart(s[j]) {
				j++
			}
			toks = append(toks, gqlTok{"var", s[i:j], i})
			i = j
		case c == '"':
			j := i + 1
			for j < n && s[j] != '"' {
				j++
			}
			toks = append(toks, gqlTok{"string", s[i:min(j+1, n)], i})
			i = j + 1
		case isGQLNameStart(c):
			j := i
			for j < n && isGQLNamePart(s[j]) {
				j++
			}
			toks = append(toks, gqlTok{"name", s[i:j], i})
			i = j
		default:
			// Numbers and other literals are consumed as opaque names for
			// argument skipping; anything unrecognized is a single-char token.
			if (c >= '0' && c <= '9') || c == '-' {
				j := i
				for j < n && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.' || s[j] == '-' || s[j] == 'e' || s[j] == 'E') {
					j++
				}
				toks = append(toks, gqlTok{"name", s[i:j], i})
				i = j
				continue
			}
			toks = append(toks, gqlTok{"punct", string(c), i})
			i++
		}
	}
	toks = append(toks, gqlTok{"eof", "", n})
	return toks, nil
}

func isGQLNameStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isGQLNamePart(c byte) bool { return isGQLNameStart(c) || (c >= '0' && c <= '9') }

type gqlParser struct {
	toks []gqlTok
	pos  int
}

func (p *gqlParser) cur() gqlTok  { return p.toks[p.pos] }
func (p *gqlParser) next() gqlTok { t := p.toks[p.pos]; p.pos++; return t }

func (p *gqlParser) parseDocument() (*GraphQLDocument, *Rejection) {
	doc := &GraphQLDocument{Fragments: map[string]GraphQLSelection{}}
	for p.cur().kind != "eof" {
		t := p.cur()
		switch {
		case t.kind == "punct" && t.text == "{":
			// Anonymous query.
			sel, rej := p.parseSelectionSet()
			if rej != nil {
				return nil, rej
			}
			doc.Operations = append(doc.Operations, GraphQLOperation{Type: OpQuery, Selection: sel})
		case t.kind == "name" && (t.text == "query" || t.text == "mutation" || t.text == "subscription"):
			op, rej := p.parseOperation()
			if rej != nil {
				return nil, rej
			}
			doc.Operations = append(doc.Operations, op)
		case t.kind == "name" && t.text == "fragment":
			name, sel, rej := p.parseFragmentDef()
			if rej != nil {
				return nil, rej
			}
			doc.Fragments[name] = sel
		default:
			return nil, &Rejection{Code: CodeGraphQLParse, Reason: "unexpected token at top level", Node: t.text, Offset: t.off}
		}
	}
	return doc, nil
}

func (p *gqlParser) parseOperation() (GraphQLOperation, *Rejection) {
	var op GraphQLOperation
	op.Type = OperationType(p.next().text)
	if p.cur().kind == "name" {
		op.Name = p.next().text
	}
	// Optional variable definitions ( ... ) — skip balanced parens.
	if p.cur().kind == "punct" && p.cur().text == "(" {
		p.skipBalanced("(", ")")
	}
	// Optional directives.
	p.skipDirectives()
	sel, rej := p.parseSelectionSet()
	if rej != nil {
		return op, rej
	}
	op.Selection = sel
	return op, nil
}

func (p *gqlParser) parseFragmentDef() (string, GraphQLSelection, *Rejection) {
	p.next() // fragment
	name := ""
	if p.cur().kind == "name" {
		name = p.next().text
	}
	// "on TypeName"
	if p.cur().kind == "name" && p.cur().text == "on" {
		p.next()
		if p.cur().kind == "name" {
			p.next()
		}
	}
	p.skipDirectives()
	sel, rej := p.parseSelectionSet()
	return name, sel, rej
}

func (p *gqlParser) parseSelectionSet() (GraphQLSelection, *Rejection) {
	var sel GraphQLSelection
	if p.cur().kind != "punct" || p.cur().text != "{" {
		return sel, &Rejection{Code: CodeGraphQLParse, Reason: "expected a selection set", Node: p.cur().text, Offset: p.cur().off}
	}
	p.next()
	for p.cur().kind != "punct" || p.cur().text != "}" {
		if p.cur().kind == "eof" {
			return sel, &Rejection{Code: CodeGraphQLParse, Reason: "unterminated selection set", Offset: p.cur().off}
		}
		if p.cur().kind == "punct" && p.cur().text == "..." {
			f, rej := p.parseFragmentUse()
			if rej != nil {
				return sel, rej
			}
			sel.Fields = append(sel.Fields, f)
			continue
		}
		f, rej := p.parseField()
		if rej != nil {
			return sel, rej
		}
		sel.Fields = append(sel.Fields, f)
	}
	p.next() // }
	return sel, nil
}

func (p *gqlParser) parseFragmentUse() (GraphQLField, *Rejection) {
	p.next() // ...
	var f GraphQLField
	if p.cur().kind == "name" && p.cur().text == "on" {
		p.next()
		if p.cur().kind == "name" {
			p.next() // type condition
		}
		p.skipDirectives()
		inl, rej := p.parseSelectionSet()
		if rej != nil {
			return f, rej
		}
		f.Inlines = append(f.Inlines, inl)
		return f, nil
	}
	if p.cur().kind == "name" {
		f.Spreads = append(f.Spreads, p.next().text)
	}
	p.skipDirectives()
	return f, nil
}

func (p *gqlParser) parseField() (GraphQLField, *Rejection) {
	var f GraphQLField
	name := p.next().text
	if p.cur().kind == "punct" && p.cur().text == ":" {
		p.next()
		f.Alias = name
		if p.cur().kind != "name" {
			return f, &Rejection{Code: CodeGraphQLParse, Reason: "expected field name after alias", Node: p.cur().text, Offset: p.cur().off}
		}
		f.Name = p.next().text
	} else {
		f.Name = name
	}
	if p.cur().kind == "punct" && p.cur().text == "(" {
		p.skipBalanced("(", ")")
	}
	f.Directives = p.collectDirectives()
	if p.cur().kind == "punct" && p.cur().text == "{" {
		sub, rej := p.parseSelectionSet()
		if rej != nil {
			return f, rej
		}
		f.Sel = &sub
	}
	return f, nil
}

func (p *gqlParser) collectDirectives() []string {
	var ds []string
	for p.cur().kind == "punct" && p.cur().text == "@" {
		p.next()
		if p.cur().kind == "name" {
			ds = append(ds, p.next().text)
		}
		if p.cur().kind == "punct" && p.cur().text == "(" {
			p.skipBalanced("(", ")")
		}
	}
	return ds
}

func (p *gqlParser) skipDirectives() { _ = p.collectDirectives() }

func (p *gqlParser) skipBalanced(open, close string) {
	depth := 0
	for p.cur().kind != "eof" {
		t := p.next()
		if t.kind == "punct" && t.text == open {
			depth++
		} else if t.kind == "punct" && t.text == close {
			depth--
			if depth == 0 {
				return
			}
		}
	}
}
