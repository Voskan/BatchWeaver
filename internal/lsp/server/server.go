// Package server implements BatchWeaver's standalone Language Server Protocol
// server. It speaks LSP 3.17 over a jsonrpc connection, treats open editor
// buffers as authoritative overlays, and publishes BatchWeaver diagnostics,
// hover, code lenses, and code actions derived from the editor service. It never
// writes source, runs project code implicitly, or imports gopls internals.
package server

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Voskan/BatchWeaver/internal/editor"
	"github.com/Voskan/BatchWeaver/internal/lsp/documents"
	"github.com/Voskan/BatchWeaver/internal/lsp/jsonrpc"
	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

// ServerName and version reported in initialize.
const (
	ServerName    = "batchweaver-lsp"
	LSPVersion    = "3.17"
	debounceDelay = 400 * time.Millisecond
)

// Options configure the server.
type Options struct {
	// ToolVersion is the BatchWeaver version reported to the client.
	ToolVersion string
	// Debounce overrides the analysis debounce delay (0 uses the default).
	Debounce time.Duration
	// Logf, if set, receives privacy-safe trace lines (never source or secrets).
	Logf func(format string, args ...any)
}

// Server is a single LSP session. It is created per connection.
type Server struct {
	opts  Options
	conn  *jsonrpc.Conn
	docs  *documents.Store
	svc   *editor.Service
	debnc time.Duration

	mu          sync.Mutex
	root        string
	initialized bool
	shutdown    bool
	gen         uint64 // analysis generation, bumped on every change
	timer       *time.Timer
}

// New returns a server with the given options.
func New(opts Options) *Server {
	d := opts.Debounce
	if d <= 0 {
		d = debounceDelay
	}
	return &Server{opts: opts, docs: documents.NewStore(), debnc: d}
}

// Run serves the LSP protocol over r/w until shutdown, EOF, or ctx cancellation.
func (s *Server) Run(ctx context.Context, r interface {
	Read([]byte) (int, error)
}, w interface {
	Write([]byte) (int, error)
}) error {
	s.conn = jsonrpc.NewConn(r, w, s.handle)
	// The serve loop ends on EOF (the client closes the stream after exit, per the
	// LSP spec) or ctx cancellation. Responses already queued are flushed first
	// because the client closes the stream only after receiving them.
	return s.conn.Serve(ctx)
}

// handle dispatches an incoming request or notification.
func (s *Server) handle(ctx context.Context, _ *jsonrpc.Conn, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	switch req.Method {
	case "initialize":
		return s.onInitialize(req)
	case "initialized", "$/setTrace", "workspace/didChangeConfiguration":
		return nil, nil
	case "shutdown":
		s.mu.Lock()
		s.shutdown = true
		s.mu.Unlock()
		return nil, nil
	case "exit":
		// The client closes the stream after exit; the serve loop then ends on EOF.
		return nil, nil
	case "textDocument/didOpen":
		return nil, s.onDidOpen(ctx, req)
	case "textDocument/didChange":
		return nil, s.onDidChange(ctx, req)
	case "textDocument/didClose":
		return nil, s.onDidClose(req)
	case "textDocument/didSave":
		s.scheduleAnalysis(ctx)
		return nil, nil
	case "textDocument/hover":
		return s.onHover(ctx, req)
	case "textDocument/codeLens":
		return s.onCodeLens(ctx, req)
	case "textDocument/codeAction":
		return s.onCodeAction(ctx, req)
	case "workspace/executeCommand":
		return s.onExecuteCommand(ctx, req)
	default:
		if req.IsNotification() {
			return nil, nil
		}
		return nil, jsonrpc.Errorf(jsonrpc.CodeMethodNotFound, "unsupported method %q", req.Method)
	}
}

// onInitialize negotiates capabilities and records the workspace root.
func (s *Server) onInitialize(req *jsonrpc.Request) (any, *jsonrpc.Error) {
	var p protocol.InitializeParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "initialize: %v", err)
	}
	root := ""
	switch {
	case len(p.WorkspaceFolders) > 0:
		if path, err := documents.URIToPath(p.WorkspaceFolders[0].URI); err == nil {
			root = path
		}
	case p.RootURI != "":
		if path, err := documents.URIToPath(p.RootURI); err == nil {
			root = path
		}
	}
	s.mu.Lock()
	s.root = root
	s.svc = editor.NewService(root, s.opts.ToolVersion)
	s.initialized = true
	s.mu.Unlock()

	caps := protocol.ServerCapabilities{
		PositionEncoding: "utf-16",
		TextDocumentSync: &protocol.TextDocumentSyncOptions{
			OpenClose: true, Change: protocol.SyncIncremental, Save: true,
		},
		HoverProvider:          true,
		CodeActionProvider:     &protocol.CodeActionOptions{CodeActionKinds: []string{protocol.CodeActionRefactorRewriteBWName, protocol.CodeActionSourceBatchWeaver}},
		CodeLensProvider:       &protocol.CodeLensOptions{},
		ExecuteCommandProvider: &protocol.ExecuteCommandOptions{Commands: editor.Commands()},
		Workspace: &protocol.WorkspaceServerCapabilities{
			WorkspaceFolders: &protocol.WorkspaceFoldersServerCapabilities{Supported: true},
		},
	}
	return protocol.InitializeResult{
		Capabilities: caps,
		ServerInfo:   &protocol.ServerInfo{Name: ServerName, Version: s.opts.ToolVersion},
	}, nil
}

func (s *Server) onDidOpen(ctx context.Context, req *jsonrpc.Request) *jsonrpc.Error {
	var p protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "didOpen: %v", err)
	}
	if _, err := s.docs.Open(p.TextDocument); err != nil {
		s.logf("didOpen rejected: %v", err)
		return nil
	}
	s.scheduleAnalysis(ctx)
	return nil
}

func (s *Server) onDidChange(ctx context.Context, req *jsonrpc.Request) *jsonrpc.Error {
	var p protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "didChange: %v", err)
	}
	if _, err := s.docs.Change(p.TextDocument, p.ContentChanges); err != nil {
		s.logf("didChange rejected: %v", err)
		return nil
	}
	s.scheduleAnalysis(ctx)
	return nil
}

func (s *Server) onDidClose(req *jsonrpc.Request) *jsonrpc.Error {
	var p protocol.DidCloseTextDocumentParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "didClose: %v", err)
	}
	s.docs.Close(p.TextDocument.URI)
	// Clear diagnostics for the closed document.
	_ = s.conn.Notify(context.Background(), "textDocument/publishDiagnostics",
		protocol.PublishDiagnosticsParams{URI: p.TextDocument.URI, Diagnostics: []protocol.Diagnostic{}})
	return nil
}

// scheduleAnalysis debounces analysis so rapid edits do not trigger repeated
// expensive package loads. Each schedule bumps the generation so a stale run
// never publishes.
func (s *Server) scheduleAnalysis(ctx context.Context) {
	s.mu.Lock()
	s.gen++
	gen := s.gen
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(s.debnc, func() {
		s.runAnalysis(ctx, gen)
	})
	s.mu.Unlock()
}

// runAnalysis performs one analysis and publishes diagnostics if still current.
func (s *Server) runAnalysis(ctx context.Context, gen uint64) {
	s.mu.Lock()
	svc := s.svc
	current := s.gen == gen
	s.mu.Unlock()
	if svc == nil || !current {
		return
	}
	res, cacheResult, err := svc.AnalyzeWithCache(ctx, s.docs.Overlay())
	if err != nil {
		s.logf("analysis error: %v", err)
		return
	}
	s.logf("analysis cache source=%s hit=%t", cacheResult.Source, cacheResult.Hit)
	s.mu.Lock()
	stillCurrent := s.gen == gen
	s.mu.Unlock()
	if !stillCurrent {
		return // a newer snapshot superseded this one; do not publish stale results
	}
	for _, uri := range s.docs.OpenURIs() {
		doc, ok := s.docs.Get(uri)
		if !ok {
			continue
		}
		diags := svc.Diagnostics(res, uri, doc.Mapper())
		if diags == nil {
			diags = []protocol.Diagnostic{}
		}
		_ = s.conn.Notify(context.Background(), "textDocument/publishDiagnostics",
			protocol.PublishDiagnosticsParams{URI: uri, Version: doc.Version, Diagnostics: diags})
	}
}

func (s *Server) logf(format string, args ...any) {
	if s.opts.Logf != nil {
		s.opts.Logf(format, args...)
	}
}
