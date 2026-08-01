package adapter

import (
	"fmt"
	"strings"
)

// Rejection reason codes (BW6xxx). This range is distinct from analysis (BW3xxx),
// proof (BW5xxx), and transformation/runtime (BW34xx/BW4xxx) codes.
const (
	CodeManifestIncompatible   = "BW6001"
	CodeCapabilityMissing      = "BW6002"
	CodeSQLDynamic             = "BW6101"
	CodeSQLUnsupported         = "BW6102"
	CodeSQLVolatile            = "BW6103"
	CodeParamAmbiguous         = "BW6104"
	CodeProjectionAmbiguous    = "BW6105"
	CodeCardinalityUnsupported = "BW6106"
	CodeTxnUnavailable         = "BW6201"
	CodeParamLimitExceeded     = "BW6202"
	CodeRedisUnbatchable       = "BW6401"
	CodeRedisSlotSpan          = "BW6402"
	CodeVerificationFailed     = "BW6501"
)

// Rejection explains why a query is outside the supported synthesis subset.
type Rejection struct {
	Code   string
	Reason string
	Node   string
	Offset int
}

func (r *Rejection) Error() string {
	return fmt.Sprintf("%s: %s (%q at offset %d)", r.Code, r.Reason, r.Node, r.Offset)
}

// Column is a (optionally qualified, optionally aliased) column reference.
type Column struct {
	Table string
	Name  string
	Alias string
}

// Relation is a table reference with an optional alias.
type Relation struct {
	Name  string
	Alias string
}

// ParsedQuery is a validated exact-key SELECT within the supported subset.
type ParsedQuery struct {
	Projection []Column
	Relation   Relation
	KeyColumn  Column
	KeyParam   int
	Extra      []string
}

// tokenKind classifies a SQL token.
type tokenKind int

const (
	tEOF tokenKind = iota
	tIdent
	tKeyword
	tPunct
	tParam
	tNumber
	tString
	tStar
)

type token struct {
	kind tokenKind
	text string
	up   string
	off  int
}

// keywords recognized by the narrow parser. Unsupported keywords are recognized
// so they can be rejected with a precise reason rather than misparsed.
var keywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "AND": true, "AS": true,
	"IS": true, "NOT": true, "NULL": true,
	// Unsupported (recognized to reject precisely):
	"JOIN": true, "INNER": true, "LEFT": true, "RIGHT": true, "FULL": true,
	"CROSS": true, "GROUP": true, "ORDER": true, "BY": true, "LIMIT": true,
	"OFFSET": true, "UNION": true, "INTERSECT": true, "EXCEPT": true,
	"HAVING": true, "WINDOW": true, "DISTINCT": true, "FOR": true, "WITH": true,
	"INSERT": true, "UPDATE": true, "DELETE": true, "MERGE": true, "OR": true,
	"ON": true, "USING": true, "IN": true, "LATERAL": true, "RETURNING": true,
}

// unsupportedKeywords trigger an immediate rejection wherever they appear.
var unsupportedKeywords = map[string]string{
	"JOIN": "joins", "INNER": "joins", "LEFT": "joins", "RIGHT": "joins",
	"FULL": "joins", "CROSS": "joins", "GROUP": "GROUP BY", "ORDER": "ORDER BY",
	"LIMIT": "LIMIT", "OFFSET": "OFFSET", "UNION": "set operations",
	"INTERSECT": "set operations", "EXCEPT": "set operations", "HAVING": "HAVING",
	"WINDOW": "window functions", "DISTINCT": "DISTINCT", "FOR": "locking clauses",
	"WITH": "CTEs", "OR": "OR predicates", "IN": "IN predicates",
	"INSERT": "writes", "UPDATE": "writes", "DELETE": "writes", "MERGE": "writes",
	"USING": "joins", "LATERAL": "lateral joins", "RETURNING": "RETURNING",
}

// tokenize splits SQL into tokens, skipping comments. It never panics.
func tokenize(sql string) ([]token, *Rejection) {
	var toks []token
	i := 0
	n := len(sql)
	for i < n {
		c := sql[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '-' && i+1 < n && sql[i+1] == '-':
			for i < n && sql[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && sql[i+1] == '*':
			i += 2
			for i+1 < n && (sql[i] != '*' || sql[i+1] != '/') {
				i++
			}
			i += 2
		case c == '$':
			j := i + 1
			for j < n && sql[j] >= '0' && sql[j] <= '9' {
				j++
			}
			if j == i+1 {
				return nil, &Rejection{Code: CodeSQLUnsupported, Reason: "unsupported placeholder syntax", Node: "$", Offset: i}
			}
			toks = append(toks, token{kind: tParam, text: sql[i:j], off: i})
			i = j
		case c == '*':
			toks = append(toks, token{kind: tStar, text: "*", off: i})
			i++
		case c == ',' || c == '.' || c == '(' || c == ')' || c == '=' || c == ';':
			toks = append(toks, token{kind: tPunct, text: string(c), off: i})
			i++
		case c == '\'':
			j := i + 1
			for j < n && sql[j] != '\'' {
				j++
			}
			toks = append(toks, token{kind: tString, text: sql[i:min(j+1, n)], off: i})
			i = j + 1
		case c == '"':
			j := i + 1
			for j < n && sql[j] != '"' {
				j++
			}
			toks = append(toks, token{kind: tIdent, text: strings.Trim(sql[i:min(j+1, n)], `"`), off: i})
			i = j + 1
		case isIdentStart(c):
			j := i
			for j < n && isIdentPart(sql[j]) {
				j++
			}
			word := sql[i:j]
			up := strings.ToUpper(word)
			if keywords[up] {
				toks = append(toks, token{kind: tKeyword, text: word, up: up, off: i})
			} else {
				toks = append(toks, token{kind: tIdent, text: word, off: i})
			}
			i = j
		case c >= '0' && c <= '9':
			j := i
			for j < n && ((sql[j] >= '0' && sql[j] <= '9') || sql[j] == '.') {
				j++
			}
			toks = append(toks, token{kind: tNumber, text: sql[i:j], off: i})
			i = j
		default:
			return nil, &Rejection{Code: CodeSQLUnsupported, Reason: "unrecognized character", Node: string(c), Offset: i}
		}
	}
	toks = append(toks, token{kind: tEOF, off: n})
	return toks, nil
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

// ParseExactKeySelect parses a scalar query and accepts it only if it is an
// exact-key SELECT within the supported subset. Anything else returns a precise
// Rejection.
func ParseExactKeySelect(sql string) (ParsedQuery, *Rejection) {
	toks, rej := tokenize(sql)
	if rej != nil {
		return ParsedQuery{}, rej
	}
	// Reject unsupported keywords anywhere.
	for _, t := range toks {
		if t.kind == tKeyword {
			if what, bad := unsupportedKeywords[t.up]; bad {
				return ParsedQuery{}, &Rejection{Code: CodeSQLUnsupported, Reason: "unsupported clause: " + what, Node: t.text, Offset: t.off}
			}
		}
	}
	p := &parser{toks: toks}
	return p.parseSelect()
}

type parser struct {
	toks []token
	pos  int
}

func (p *parser) cur() token  { return p.toks[p.pos] }
func (p *parser) next() token { t := p.toks[p.pos]; p.pos++; return t }

func (p *parser) parseSelect() (ParsedQuery, *Rejection) {
	var q ParsedQuery
	if p.cur().up != "SELECT" {
		return q, &Rejection{Code: CodeSQLUnsupported, Reason: "only SELECT queries are supported", Node: p.cur().text, Offset: p.cur().off}
	}
	p.next()

	// Projection.
	for {
		if p.cur().kind == tStar {
			return q, &Rejection{Code: CodeProjectionAmbiguous, Reason: "SELECT * is not supported for synthesis; list explicit columns", Node: "*", Offset: p.cur().off}
		}
		col, rej := p.parseColumn(true)
		if rej != nil {
			return q, rej
		}
		q.Projection = append(q.Projection, col)
		if p.cur().kind == tPunct && p.cur().text == "," {
			p.next()
			continue
		}
		break
	}

	if p.cur().up != "FROM" {
		return q, &Rejection{Code: CodeSQLUnsupported, Reason: "expected FROM", Node: p.cur().text, Offset: p.cur().off}
	}
	p.next()

	// Relation (single table, optional alias).
	if p.cur().kind != tIdent {
		return q, &Rejection{Code: CodeSQLUnsupported, Reason: "expected a table name", Node: p.cur().text, Offset: p.cur().off}
	}
	q.Relation.Name = p.next().text
	if p.cur().up == "AS" {
		p.next()
	}
	if p.cur().kind == tIdent {
		q.Relation.Alias = p.next().text
	}
	if p.cur().kind == tPunct && p.cur().text == "," {
		return q, &Rejection{Code: CodeSQLUnsupported, Reason: "multiple relations are not supported", Node: ",", Offset: p.cur().off}
	}

	if p.cur().up != "WHERE" {
		return q, &Rejection{Code: CodeSQLUnsupported, Reason: "an exact-key WHERE clause is required", Node: p.cur().text, Offset: p.cur().off}
	}
	p.next()

	// Key predicate: colref '=' $N.
	keyCol, rej := p.parseColumn(false)
	if rej != nil {
		return q, rej
	}
	if p.cur().kind != tPunct || p.cur().text != "=" {
		return q, &Rejection{Code: CodeSQLUnsupported, Reason: "the key predicate must be an equality", Node: p.cur().text, Offset: p.cur().off}
	}
	p.next()
	if p.cur().kind != tParam {
		return q, &Rejection{Code: CodeParamAmbiguous, Reason: "the key must compare to a single bound parameter", Node: p.cur().text, Offset: p.cur().off}
	}
	param := p.next()
	q.KeyColumn = keyCol
	q.KeyParam = paramNumber(param.text)

	// Optional key-independent AND predicates (col IS [NOT] NULL only).
	for p.cur().up == "AND" {
		andOff := p.cur().off
		p.next()
		ec, rej := p.parseColumn(false)
		if rej != nil {
			return q, rej
		}
		if p.cur().up != "IS" {
			return q, &Rejection{Code: CodeSQLUnsupported, Reason: "only key-independent 'IS [NOT] NULL' predicates are supported after the key", Node: p.cur().text, Offset: andOff}
		}
		p.next()
		neg := false
		if p.cur().up == "NOT" {
			neg = true
			p.next()
		}
		if p.cur().up != "NULL" {
			return q, &Rejection{Code: CodeSQLUnsupported, Reason: "expected NULL", Node: p.cur().text, Offset: p.cur().off}
		}
		p.next()
		pred := columnSQL(ec) + " IS "
		if neg {
			pred += "NOT "
		}
		pred += "NULL"
		q.Extra = append(q.Extra, pred)
	}

	// Optional single trailing ';' then EOF.
	if p.cur().kind == tPunct && p.cur().text == ";" {
		p.next()
	}
	if p.cur().kind != tEOF {
		return q, &Rejection{Code: CodeSQLUnsupported, Reason: "unexpected trailing tokens or multiple statements", Node: p.cur().text, Offset: p.cur().off}
	}
	return q, nil
}

// parseColumn parses a colref (ident[.ident]) with an optional alias when
// allowAlias is true. A following '(' means a function call, which is rejected as
// potentially volatile.
func (p *parser) parseColumn(allowAlias bool) (Column, *Rejection) {
	var col Column
	if p.cur().kind != tIdent {
		return col, &Rejection{Code: CodeSQLUnsupported, Reason: "expected a column reference", Node: p.cur().text, Offset: p.cur().off}
	}
	first := p.next().text
	if p.cur().kind == tPunct && p.cur().text == "." {
		p.next()
		if p.cur().kind != tIdent {
			return col, &Rejection{Code: CodeSQLUnsupported, Reason: "expected a column name after '.'", Node: p.cur().text, Offset: p.cur().off}
		}
		col.Table = first
		col.Name = p.next().text
	} else {
		col.Name = first
	}
	if p.cur().kind == tPunct && p.cur().text == "(" {
		return col, &Rejection{Code: CodeSQLVolatile, Reason: "function calls are not supported (potential volatile or statement-level semantics)", Node: first, Offset: p.cur().off}
	}
	if allowAlias {
		if p.cur().up == "AS" {
			p.next()
			if p.cur().kind == tIdent {
				col.Alias = p.next().text
			}
		} else if p.cur().kind == tIdent {
			col.Alias = p.next().text
		}
	}
	return col, nil
}

func paramNumber(s string) int {
	n := 0
	for _, r := range s[1:] {
		n = n*10 + int(r-'0')
	}
	return n
}

// columnSQL renders a column reference back to SQL.
func columnSQL(c Column) string {
	if c.Table != "" {
		return c.Table + "." + c.Name
	}
	return c.Name
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
