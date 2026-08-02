package editor

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/internal/lsp/documents"
	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

// Command IDs surfaced to the editor. They are stable identifiers used by
// executeCommand and by code actions / lenses.
const (
	CmdScanWorkspace = "batchweaver.scanWorkspace"
	CmdPreview       = "batchweaver.previewTransformation"
	CmdProve         = "batchweaver.proveCandidate"
	CmdShowGraph     = "batchweaver.showOperationGraph"
	CmdDoctor        = "batchweaver.doctor"
)

// Commands returns the stable list of workspace commands the server advertises.
func Commands() []string {
	return []string{CmdScanWorkspace, CmdPreview, CmdProve, CmdShowGraph, CmdDoctor}
}

// Hover returns hover content for a scalar operation call at the given position,
// or nil when the position is not on a known call site.
func (s *Service) Hover(res *Result, uri protocol.DocumentURI, pos protocol.Position, mapper *documents.Mapper) *protocol.Hover {
	rel, ok := s.relForURI(uri)
	if !ok {
		return nil
	}
	cs, r, found := s.callSiteAt(res, rel, pos, mapper)
	if !found {
		return nil
	}
	op := res.opByID[cs.Operation]
	var b []string
	b = append(b, "**BatchWeaver operation:** `"+cs.Operation+"`")
	if op.ScalarSymbol != "" || op.BatchSymbol != "" {
		b = append(b, "", "**Binding:**", "- scalar: `"+dash(op.ScalarSymbol)+"`", "- batch: `"+dash(op.BatchSymbol)+"`")
	}
	if cs.KeyDependency != "" {
		b = append(b, "", "**Key dependency:** "+cs.KeyDependency)
	}
	if cs.Structural != "" {
		b = append(b, "**Structural context:** "+cs.Structural)
	}
	if cand, ok := res.candByCall[cs.ID]; ok {
		b = append(b, "", "**Candidate state:** "+cand.State)
	}
	b = append(b, "", "_No source changes have been applied._")
	return &protocol.Hover{
		Contents: protocol.MarkupContent{Kind: "markdown", Value: join(b)},
		Range:    &r,
	}
}

// callSiteAt returns the call site whose line matches the position line.
func (s *Service) callSiteAt(res *Result, rel string, pos protocol.Position, mapper *documents.Mapper) (analysis.CallSite, protocol.Range, bool) {
	for _, cs := range res.Snapshot.CallSites {
		pl, ok := parseLocation(cs.Location)
		if !ok || pl.rel != rel {
			continue
		}
		if uint32(pl.line-1) == pos.Line {
			return cs, rangeForLineCol(mapper, pl), true
		}
	}
	return analysis.CallSite{}, protocol.Range{}, false
}

// CodeLens returns lenses for operation declarations and candidate loops in the
// document. Lenses are lightweight (a command reference); nothing expensive runs
// merely because a lens is visible.
func (s *Service) CodeLens(res *Result, uri protocol.DocumentURI, mapper *documents.Mapper) []protocol.CodeLens {
	rel, ok := s.relForURI(uri)
	if !ok {
		return nil
	}
	var lenses []protocol.CodeLens

	for _, op := range res.Snapshot.Operations {
		for _, src := range op.Sources {
			pl, ok := parseLocation(src.Location)
			if !ok || pl.rel != rel {
				continue
			}
			proven, transformed := candidateCounts(res, op.ID)
			title := "BatchWeaver: " + strconv.Itoa(len(op.CallSiteIDs)) + " call sites · " +
				strconv.Itoa(proven) + " candidate · " + strconv.Itoa(transformed) + " transformable"
			lenses = append(lenses, protocol.CodeLens{
				Range:   rangeForLineCol(mapper, pl),
				Command: &protocol.Command{Title: title, Command: CmdShowGraph, Arguments: args(op.ID)},
			})
			break
		}
	}

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
		lenses = append(lenses, protocol.CodeLens{
			Range:   rangeForLineCol(mapper, pl),
			Command: &protocol.Command{Title: "BatchWeaver: preview batch transformation", Command: CmdPreview, Arguments: args(cand.ID)},
		})
	}

	sort.SliceStable(lenses, func(i, j int) bool {
		return lenses[i].Range.Start.Line < lenses[j].Range.Start.Line
	})
	return lenses
}

// candidateCounts returns the number of candidates and transformable candidates
// for an operation (a conservative, deterministic summary).
func candidateCounts(res *Result, opID string) (candidates, transformable int) {
	for _, cand := range res.Snapshot.Candidates {
		if cand.Operation != opID {
			continue
		}
		candidates++
		if cand.State == "eligible" || cand.State == "candidate" {
			transformable++
		}
	}
	return candidates, transformable
}

// CodeActions returns the actions available for a range: a preview action for
// each candidate whose first call site falls within the range, plus a workspace
// scan source action. Actions are command-backed and never mutate source.
func (s *Service) CodeActions(res *Result, uri protocol.DocumentURI, rng protocol.Range, mapper *documents.Mapper) []protocol.CodeAction {
	rel, ok := s.relForURI(uri)
	if !ok {
		return nil
	}
	var actions []protocol.CodeAction
	seen := map[string]bool{}
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
		line := uint32(max0(pl.line - 1))
		if line < rng.Start.Line || line > rng.End.Line {
			continue
		}
		if seen[cand.ID] {
			continue
		}
		seen[cand.ID] = true
		actions = append(actions, protocol.CodeAction{
			Title:   "BatchWeaver: Preview transformation for " + cand.Operation,
			Kind:    protocol.CodeActionRefactorRewriteBWName,
			Command: &protocol.Command{Title: "Preview transformation", Command: CmdPreview, Arguments: args(cand.ID)},
		})
	}
	actions = append(actions, protocol.CodeAction{
		Title:   "BatchWeaver: Scan workspace",
		Kind:    protocol.CodeActionSourceBatchWeaver,
		Command: &protocol.Command{Title: "Scan workspace", Command: CmdScanWorkspace},
	})
	return actions
}

// args marshals command arguments to raw JSON.
func args(vs ...string) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(vs))
	for _, v := range vs {
		raw, _ := json.Marshal(v)
		out = append(out, raw)
	}
	return out
}

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func join(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
