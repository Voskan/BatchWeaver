package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Voskan/BatchWeaver/internal/editor"
	"github.com/Voskan/BatchWeaver/internal/lsp/jsonrpc"
	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

// analyzeNow runs a synchronous analysis over the current overlay for an
// on-demand (user-initiated) request. It returns nil when the server is not
// initialized.
func (s *Server) analyzeNow(ctx context.Context) (*editor.Result, error) {
	s.mu.Lock()
	svc := s.svc
	s.mu.Unlock()
	if svc == nil {
		return nil, fmt.Errorf("server not initialized")
	}
	result, cacheResult, err := svc.AnalyzeWithCache(ctx, s.docs.Overlay())
	if err == nil {
		s.logf("analysis cache source=%s hit=%t", cacheResult.Source, cacheResult.Hit)
	}
	return result, err
}

func (s *Server) onHover(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	var p protocol.TextDocumentPositionParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "hover: %v", err)
	}
	doc, ok := s.docs.Get(p.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	res, err := s.analyzeNow(ctx)
	if err != nil {
		s.logf("hover analysis: %v", err)
		return nil, nil //nolint:nilerr // analysis failure yields no hover, not an LSP error
	}
	s.mu.Lock()
	svc := s.svc
	s.mu.Unlock()
	return svc.Hover(res, p.TextDocument.URI, p.Position, doc.Mapper()), nil
}

func (s *Server) onCodeLens(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	var p protocol.CodeLensParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "codeLens: %v", err)
	}
	doc, ok := s.docs.Get(p.TextDocument.URI)
	if !ok {
		return []protocol.CodeLens{}, nil
	}
	res, err := s.analyzeNow(ctx)
	if err != nil {
		s.logf("codeLens analysis: %v", err)
		return []protocol.CodeLens{}, nil //nolint:nilerr // analysis failure yields no lenses, not an LSP error
	}
	s.mu.Lock()
	svc := s.svc
	s.mu.Unlock()
	lenses := svc.CodeLens(res, p.TextDocument.URI, doc.Mapper())
	if lenses == nil {
		lenses = []protocol.CodeLens{}
	}
	return lenses, nil
}

func (s *Server) onCodeAction(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	var p protocol.CodeActionParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "codeAction: %v", err)
	}
	doc, ok := s.docs.Get(p.TextDocument.URI)
	if !ok {
		return []protocol.CodeAction{}, nil
	}
	res, err := s.analyzeNow(ctx)
	if err != nil {
		s.logf("codeAction analysis: %v", err)
		return []protocol.CodeAction{}, nil //nolint:nilerr // analysis failure yields no actions, not an LSP error
	}
	s.mu.Lock()
	svc := s.svc
	s.mu.Unlock()
	actions := svc.CodeActions(res, p.TextDocument.URI, p.Range, doc.Mapper())
	if actions == nil {
		actions = []protocol.CodeAction{}
	}
	return actions, nil
}

// onExecuteCommand runs a BatchWeaver workspace command. Commands are read-only
// and return information; none mutate source. A preview returns text the client
// can display in a virtual document.
func (s *Server) onExecuteCommand(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	var p protocol.ExecuteCommandParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "executeCommand: %v", err)
	}
	res, err := s.analyzeNow(ctx)
	if err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInternalError, "%v", err)
	}
	switch p.Command {
	case editor.CmdScanWorkspace:
		return editor.ScanSummary(res), nil
	case editor.CmdShowGraph:
		return editor.OperationGraphText(res, firstStringArg(p.Arguments)), nil
	case editor.CmdPreview:
		return editor.PreviewText(res, firstStringArg(p.Arguments)), nil
	case editor.CmdProve:
		return editor.PreviewText(res, firstStringArg(p.Arguments)), nil
	case editor.CmdDoctor:
		return editor.DoctorText(res), nil
	default:
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "unknown command %q", p.Command)
	}
}

// firstStringArg extracts the first string argument, if present.
func firstStringArg(args []json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var s string
	_ = json.Unmarshal(args[0], &s)
	return s
}
