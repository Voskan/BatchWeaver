package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/Voskan/BatchWeaver/internal/editor"
	"github.com/Voskan/BatchWeaver/internal/lsp/documents"
	"github.com/Voskan/BatchWeaver/internal/lsp/jsonrpc"
	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

// Options configure the proxy.
type Options struct {
	// GoplsCommand is the gopls executable to launch (default "gopls").
	GoplsCommand string
	// ToolVersion is the BatchWeaver version reported to the client.
	ToolVersion string
	// Logf receives privacy-safe trace lines on stderr; never source or secrets.
	Logf func(format string, args ...any)
}

// Proxy composes gopls with BatchWeaver behind one LSP endpoint.
type Proxy struct {
	opts   Options
	docs   *documents.Store
	svc    *editor.Service
	client *jsonrpc.Conn
	gopls  *jsonrpc.Conn
	cmd    *exec.Cmd

	mu     sync.Mutex
	root   string
	cancel context.CancelFunc
}

// New returns a proxy with the given options.
func New(opts Options) *Proxy {
	if opts.GoplsCommand == "" {
		opts.GoplsCommand = "gopls"
	}
	return &Proxy{opts: opts, docs: documents.NewStore()}
}

// Run starts gopls, wires both connections, and serves until either side closes.
func (p *Proxy) Run(ctx context.Context, r io.Reader, w io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p.mu.Lock()
	p.cancel = cancel
	p.mu.Unlock()

	cmd := exec.CommandContext(ctx, p.opts.GoplsCommand)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("proxy: gopls stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("proxy: gopls stdout: %w", err)
	}
	cmd.Stderr = writerFunc(func(b []byte) (int, error) {
		p.logf("gopls: %s", strings.TrimRight(string(b), "\n"))
		return len(b), nil
	})
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("proxy: start gopls %q: %w (is it installed and on PATH?)", p.opts.GoplsCommand, err)
	}
	p.cmd = cmd

	// gopls-facing connection: handles server-initiated requests from gopls by
	// forwarding them to the client.
	p.gopls = jsonrpc.NewConn(stdout, stdin, p.handleFromGopls)
	// client-facing connection: handles editor requests.
	p.client = jsonrpc.NewConn(r, w, p.handleFromClient)

	go func() {
		_ = p.gopls.Serve(ctx)
		cancel() // gopls exit tears down the session
	}()
	err = p.client.Serve(ctx)
	cancel()
	_ = cmd.Wait()
	return err
}

// handleFromClient routes an editor request. BatchWeaver-owned methods are
// handled locally; standard methods are forwarded to gopls; a few are merged.
func (p *Proxy) handleFromClient(ctx context.Context, _ *jsonrpc.Conn, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	// Feed text synchronization into the BatchWeaver document store as well as
	// forwarding to gopls, so BatchWeaver diagnostics track unsaved buffers.
	p.trackDocument(req)

	switch req.Method {
	case "initialize":
		return p.onInitialize(ctx, req)
	case "workspace/executeCommand":
		if p.isBatchWeaverCommand(req.Params) {
			return p.executeBatchWeaverCommand(ctx, req)
		}
		return p.forwardRequest(ctx, req)
	case "textDocument/hover":
		return p.mergeHover(ctx, req)
	case "textDocument/codeAction":
		return p.mergeCodeAction(ctx, req)
	case "textDocument/codeLens":
		return p.mergeCodeLens(ctx, req)
	case "shutdown":
		return p.forwardRequest(ctx, req)
	case "exit":
		_ = p.gopls.Notify(ctx, "exit", nil)
		p.mu.Lock()
		cancel := p.cancel
		p.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return nil, nil
	default:
		if req.IsNotification() {
			// Broadcast notifications (didOpen/didChange/didSave/…) to gopls.
			_ = p.gopls.Notify(ctx, req.Method, req.Params)
			if req.Method == "textDocument/didChange" || req.Method == "textDocument/didOpen" || req.Method == "textDocument/didSave" {
				go p.publishBatchWeaverDiagnostics(ctx)
			}
			return nil, nil
		}
		return p.forwardRequest(ctx, req)
	}
}

// handleFromGopls forwards gopls server-initiated requests and notifications to
// the client. Diagnostics keep gopls's own source so they never collide with
// BatchWeaver diagnostics.
func (p *Proxy) handleFromGopls(ctx context.Context, _ *jsonrpc.Conn, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	if req.IsNotification() {
		_ = p.client.Notify(ctx, req.Method, req.Params)
		return nil, nil
	}
	// Forward server-initiated request to the client and return its response.
	res, err := p.client.Call(ctx, req.Method, req.Params)
	if err != nil {
		var rpcErr *jsonrpc.Error
		if errors.As(err, &rpcErr) {
			return nil, rpcErr
		}
		return nil, jsonrpc.Errorf(jsonrpc.CodeInternalError, "%v", err)
	}
	return res, nil
}

// forwardRequest forwards a request to gopls and returns its response. The
// destination connection assigns its own request ID, so client/gopls IDs never
// collide.
func (p *Proxy) forwardRequest(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	res, err := p.gopls.Call(ctx, req.Method, req.Params)
	if err != nil {
		var rpcErr *jsonrpc.Error
		if errors.As(err, &rpcErr) {
			return nil, rpcErr
		}
		return nil, jsonrpc.Errorf(jsonrpc.CodeInternalError, "%v", err)
	}
	return res, nil
}

// trackDocument mirrors text synchronization into the BatchWeaver store.
func (p *Proxy) trackDocument(req *jsonrpc.Request) {
	switch req.Method {
	case "textDocument/didOpen":
		var pm protocol.DidOpenTextDocumentParams
		if json.Unmarshal(req.Params, &pm) == nil {
			_, _ = p.docs.Open(pm.TextDocument)
		}
	case "textDocument/didChange":
		var pm protocol.DidChangeTextDocumentParams
		if json.Unmarshal(req.Params, &pm) == nil {
			_, _ = p.docs.Change(pm.TextDocument, pm.ContentChanges)
		}
	case "textDocument/didClose":
		var pm protocol.DidCloseTextDocumentParams
		if json.Unmarshal(req.Params, &pm) == nil {
			p.docs.Close(pm.TextDocument.URI)
		}
	}
}

func (p *Proxy) logf(format string, args ...any) {
	if p.opts.Logf != nil {
		p.opts.Logf(format, args...)
	}
}

// writerFunc adapts a function to io.Writer for capturing gopls stderr.
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(b []byte) (int, error) { return f(b) }
