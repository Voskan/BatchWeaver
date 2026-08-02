package configdecode

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
)

// ParseYAML parses YAML bytes into a node tree, attaching file as the position
// file name. It rejects multiple documents, duplicate mapping keys, and unsafe
// YAML constructs (anchors, aliases, tags, and merge keys). On a fatal parse
// error it returns a nil node and a collection describing the problem.
func ParseYAML(file string, src []byte) (*Node, diagnostics.Collection) {
	var diags diagnostics.Collection
	f, err := parser.ParseBytes(src, 0)
	if err != nil {
		diags.Add(classifyYAMLError(file, err))
		return nil, diags
	}
	if len(f.Docs) == 0 {
		// An empty document is a null value; normalization reports missing fields.
		return &Node{Kind: KindScalar, ScalarType: ScalarNull, Value: "null", Pos: diagnostics.Position{File: file}}, diags
	}
	if len(f.Docs) > 1 {
		pos := tokenPos(file, documentToken(f.Docs[1]))
		diags.Add(diag(CodeMultipleDocuments, pos, "configuration source must contain exactly one document"))
		return nil, diags
	}
	body := f.Docs[0].Body
	if body == nil {
		return &Node{Kind: KindScalar, ScalarType: ScalarNull, Value: "null", Pos: diagnostics.Position{File: file}}, diags
	}
	node := convertYAML(file, body, &diags)
	if diags.HasErrors() {
		return nil, diags
	}
	return node, diags
}

// convertYAML converts a goccy AST node into a *Node, appending diagnostics for
// unsupported constructs and duplicate keys.
func convertYAML(file string, n ast.Node, diags *diagnostics.Collection) *Node {
	switch v := n.(type) {
	case *ast.MappingNode:
		return convertMapping(file, v.Values, diags)
	case *ast.MappingValueNode:
		return convertMapping(file, []*ast.MappingValueNode{v}, diags)
	case *ast.SequenceNode:
		out := &Node{Kind: KindSequence, Pos: tokenPos(file, v.GetToken())}
		for _, e := range v.Values {
			out.Elems = append(out.Elems, convertYAML(file, e, diags))
		}
		return out
	case *ast.StringNode:
		return &Node{Kind: KindScalar, ScalarType: ScalarString, Value: v.Value, Pos: tokenPos(file, v.GetToken())}
	case *ast.LiteralNode:
		return &Node{Kind: KindScalar, ScalarType: ScalarString, Value: v.Value.Value, Pos: tokenPos(file, v.GetToken())}
	case *ast.IntegerNode:
		return &Node{Kind: KindScalar, ScalarType: ScalarInt, Value: v.GetToken().Value, Pos: tokenPos(file, v.GetToken())}
	case *ast.FloatNode:
		return &Node{Kind: KindScalar, ScalarType: ScalarFloat, Value: v.GetToken().Value, Pos: tokenPos(file, v.GetToken())}
	case *ast.BoolNode:
		return &Node{Kind: KindScalar, ScalarType: ScalarBool, Value: v.GetToken().Value, Pos: tokenPos(file, v.GetToken())}
	case *ast.NullNode:
		return &Node{Kind: KindScalar, ScalarType: ScalarNull, Value: "null", Pos: tokenPos(file, v.GetToken())}
	case *ast.TagNode, *ast.AnchorNode, *ast.AliasNode:
		diags.Add(diag(CodeUnsupportedConstruct, tokenPos(file, n.GetToken()),
			"YAML anchors, aliases, and tags are not supported"))
		return &Node{Kind: KindScalar, ScalarType: ScalarNull, Pos: tokenPos(file, n.GetToken())}
	default:
		diags.Add(diag(CodeUnsupportedConstruct, tokenPos(file, n.GetToken()),
			fmt.Sprintf("unsupported YAML construct %T", n)))
		return &Node{Kind: KindScalar, ScalarType: ScalarNull, Pos: tokenPos(file, n.GetToken())}
	}
}

// convertMapping builds a mapping node, rejecting duplicate keys and merge keys.
func convertMapping(file string, values []*ast.MappingValueNode, diags *diagnostics.Collection) *Node {
	out := &Node{Kind: KindMapping}
	if len(values) > 0 {
		out.Pos = tokenPos(file, values[0].GetToken())
	}
	seen := make(map[string]diagnostics.Position, len(values))
	for _, mv := range values {
		if _, isMerge := mv.Key.(*ast.MergeKeyNode); isMerge {
			diags.Add(diag(CodeUnsupportedConstruct, tokenPos(file, mv.GetToken()),
				"YAML merge keys are not supported"))
			continue
		}
		keyTok := mv.Key.GetToken()
		key := keyTok.Value
		keyPos := tokenPos(file, keyTok)
		if prev, dup := seen[key]; dup {
			d := diag(CodeDuplicateKey, keyPos, fmt.Sprintf("duplicate mapping key %q", key))
			d.Related = []diagnostics.RelatedInformation{{Message: "first defined here", Range: diagnostics.AtPosition(prev)}}
			diags.Add(d)
			continue
		}
		seen[key] = keyPos
		out.Entries = append(out.Entries, MapEntry{
			Key:    key,
			KeyPos: keyPos,
			Value:  convertYAML(file, mv.Value, diags),
		})
	}
	return out
}

// yamlErrPos matches the leading "[line:column]" prefix of a goccy error.
var yamlErrPos = regexp.MustCompile(`\[(\d+):(\d+)\]`)

// classifyYAMLError maps a goccy parse error to a diagnostic, recognizing the
// duplicate-key error (which goccy detects during parsing) so it reports the
// stable BWCFG002 code rather than a generic syntax error.
func classifyYAMLError(file string, err error) diagnostics.Diagnostic {
	msg := err.Error()
	pos := diagnostics.Position{File: file}
	if m := yamlErrPos.FindStringSubmatch(msg); m != nil {
		pos.Line = atoi(m[1])
		pos.Column = atoi(m[2])
	}
	if strings.Contains(msg, "already defined") || strings.Contains(msg, "duplicate") {
		return diag(CodeDuplicateKey, pos, "duplicate mapping key")
	}
	return diag(CodeSyntax, pos, fmt.Sprintf("YAML syntax error: %v", err))
}

// atoi parses a positive integer, returning 0 on error.
func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}

// tokenPos converts a goccy token into a diagnostics.Position with file set.
// A nil token yields a position that carries only the file name.
// documentToken returns the first token of a parsed YAML document, or nil when
// the document has no body. goccy/go-yaml's DocumentNode.GetToken dereferences
// the document body, so a body-less document — for example the second half of
// "---\n---" — would otherwise panic with a nil pointer dereference.
func documentToken(doc *ast.DocumentNode) *token.Token {
	if doc == nil || doc.Body == nil {
		return nil
	}
	return doc.GetToken()
}

func tokenPos(file string, tok *token.Token) diagnostics.Position {
	if tok == nil || tok.Position == nil {
		return diagnostics.Position{File: file}
	}
	return diagnostics.Position{
		File:   file,
		Offset: tok.Position.Offset,
		Line:   tok.Position.Line,
		Column: tok.Position.Column,
	}
}
