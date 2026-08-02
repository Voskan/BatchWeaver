package editor

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/internal/lsp/documents"
	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

// DiagnosticSource is the LSP diagnostic source string for BatchWeaver findings.
const DiagnosticSource = "batchweaver"

// docsBaseURL is the base for diagnostic code documentation links.
const docsBaseURL = "https://github.com/Voskan/BatchWeaver/blob/main/docs/reference/"

// Service analyzes unsaved editor buffers and derives LSP features. It is
// stateless aside from the workspace root and tool version; each Analyze call
// runs an independent, snapshot-bound analysis over the provided overlay.
type Service struct {
	root        string
	toolVersion string
}

// NewService returns an editor service rooted at root.
func NewService(root, toolVersion string) *Service {
	return &Service{root: root, toolVersion: toolVersion}
}

// Root returns the workspace root.
func (s *Service) Root() string { return s.root }

// Result is a snapshot-bound analysis result plus indexes for feature lookups.
// It is immutable; the same overlay yields an equal Result.
type Result struct {
	Snapshot   *analysis.Snapshot
	callByID   map[string]analysis.CallSite
	opByID     map[string]analysis.Operation
	candByCall map[string]analysis.Candidate
}

// Analyze runs a BatchWeaver analysis over the overlay (unsaved buffers) and
// returns a snapshot-bound Result. Patterns default to the whole module.
func (s *Service) Analyze(ctx context.Context, overlay map[string][]byte) (*Result, error) {
	snap, err := analysis.Analyze(ctx, analysis.Request{
		Patterns:     []string{"./..."},
		Dir:          s.root,
		Overlay:      overlay,
		Reproducible: true,
		ToolVersion:  s.toolVersion,
	})
	if err != nil {
		return nil, err
	}
	r := &Result{
		Snapshot:   snap,
		callByID:   make(map[string]analysis.CallSite, len(snap.CallSites)),
		opByID:     make(map[string]analysis.Operation, len(snap.Operations)),
		candByCall: make(map[string]analysis.Candidate),
	}
	for _, c := range snap.CallSites {
		r.callByID[c.ID] = c
	}
	for _, o := range snap.Operations {
		r.opByID[o.ID] = o
	}
	for _, cand := range snap.Candidates {
		for _, cs := range cand.CallSites {
			if _, exists := r.candByCall[cs]; !exists {
				r.candByCall[cs] = cand
			}
		}
	}
	return r, nil
}

// parsedLocation is a decoded "relpath:line:col" analysis location.
type parsedLocation struct {
	rel  string
	line int
	col  int
}

// parseLocation decodes an analysis location string. It tolerates Windows drive
// letters by splitting on the last two colons.
func parseLocation(loc string) (parsedLocation, bool) {
	if loc == "" {
		return parsedLocation{}, false
	}
	// Split off column and line from the right so a "C:\..." prefix survives.
	i := strings.LastIndex(loc, ":")
	if i < 0 {
		return parsedLocation{}, false
	}
	colStr := loc[i+1:]
	rest := loc[:i]
	j := strings.LastIndex(rest, ":")
	if j < 0 {
		return parsedLocation{}, false
	}
	lineStr := rest[j+1:]
	rel := rest[:j]
	line, err1 := strconv.Atoi(lineStr)
	col, err2 := strconv.Atoi(colStr)
	if err1 != nil || err2 != nil {
		return parsedLocation{}, false
	}
	return parsedLocation{rel: filepath.ToSlash(rel), line: line, col: col}, true
}

// relForURI returns the workspace-relative, slash-separated path for a document
// URI, or ("", false) if it is outside the workspace.
func (s *Service) relForURI(uri protocol.DocumentURI) (string, bool) {
	path, err := documents.URIToPath(uri)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(s.root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// severityFor maps an analysis severity string to an LSP severity.
func severityFor(sev string) protocol.DiagnosticSeverity {
	switch sev {
	case "error":
		return protocol.SeverityError
	case "warning":
		return protocol.SeverityWarning
	case "info":
		return protocol.SeverityInformation
	default:
		return protocol.SeverityHint
	}
}

// Diagnostics returns the BatchWeaver diagnostics for one document, mapped to
// LSP ranges using the document's own mapper. It includes analysis diagnostics
// located in the file plus informational batching-opportunity diagnostics for
// discovered candidates. Ordering is deterministic.
func (s *Service) Diagnostics(res *Result, uri protocol.DocumentURI, mapper *documents.Mapper) []protocol.Diagnostic {
	rel, ok := s.relForURI(uri)
	if !ok {
		return nil
	}
	var out []protocol.Diagnostic

	for _, d := range res.Snapshot.Diagnostics {
		pl, ok := parseLocation(d.Location)
		if !ok || pl.rel != rel {
			continue
		}
		out = append(out, protocol.Diagnostic{
			Range:           rangeForLineCol(mapper, pl),
			Severity:        severityFor(d.Severity),
			Code:            d.Code,
			CodeDescription: codeDescription(d.Code),
			Source:          DiagnosticSource,
			Message:         d.Message,
		})
	}

	// One informational batching-opportunity diagnostic per candidate whose first
	// call site is in this file.
	for _, cand := range res.Snapshot.Candidates {
		if len(cand.CallSites) == 0 {
			continue
		}
		cs, ok := res.callByID[cand.CallSites[0]]
		if !ok {
			continue
		}
		pl, ok := parseLocation(cs.Location)
		if !ok || pl.rel != rel {
			continue
		}
		out = append(out, protocol.Diagnostic{
			Range:           rangeForLineCol(mapper, pl),
			Severity:        protocol.SeverityInformation,
			Code:            "BW1001",
			CodeDescription: codeDescription("BW1001"),
			Source:          DiagnosticSource,
			Message:         opportunityMessage(cand, res),
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Range.Start.Line != out[j].Range.Start.Line {
			return out[i].Range.Start.Line < out[j].Range.Start.Line
		}
		if out[i].Range.Start.Character != out[j].Range.Start.Character {
			return out[i].Range.Start.Character < out[j].Range.Start.Character
		}
		return out[i].Code < out[j].Code
	})
	return out
}

// opportunityMessage renders a batching-opportunity message for a candidate.
func opportunityMessage(cand analysis.Candidate, res *Result) string {
	op := res.opByID[cand.Operation]
	n := len(cand.CallSites)
	plural := "call sites"
	if n == 1 {
		plural = "call site"
	}
	msg := strconv.Itoa(n) + " scalar " + plural + " for operation " + cand.Operation +
		" may be batchable (" + cand.StructuralContext + ")."
	if op.BatchSymbol != "" {
		msg += " Batch provider: " + op.BatchSymbol + "."
	}
	msg += " No source changes have been applied."
	return msg
}

// codeDescription returns a documentation link for a diagnostic code family.
func codeDescription(code string) *protocol.CodeDescription {
	var page string
	switch {
	case strings.HasPrefix(code, "BW1"), strings.HasPrefix(code, "BW3"):
		page = "analysis-diagnostics.md"
	case strings.HasPrefix(code, "BW5"):
		page = "proof-diagnostics.md"
	case strings.HasPrefix(code, "BW6"), strings.HasPrefix(code, "BW7"):
		page = "protocol-diagnostics.md"
	case strings.HasPrefix(code, "BW8"):
		page = "lsp-diagnostics.md"
	default:
		return nil
	}
	return &protocol.CodeDescription{Href: docsBaseURL + page}
}

// rangeForLineCol maps a parsed location to an LSP range covering the token
// start. It uses the document mapper so UTF-16 offsets are correct.
func rangeForLineCol(mapper *documents.Mapper, pl parsedLocation) protocol.Range {
	if mapper == nil {
		p := protocol.Position{Line: uint32(max0(pl.line - 1)), Character: uint32(max0(pl.col - 1))}
		return protocol.Range{Start: p, End: p}
	}
	off := mapper.LineByteColToOffset(pl.line, pl.col)
	start := mapper.OffsetToPosition(off)
	// Highlight to end of line for visibility without guessing token width.
	endOff := off
	content := mapper.Content()
	for endOff < len(content) && content[endOff] != '\n' {
		endOff++
	}
	return protocol.Range{Start: start, End: mapper.OffsetToPosition(endOff)}
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
