package proxy

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Voskan/BatchWeaver/internal/editor"
	"github.com/Voskan/BatchWeaver/internal/lsp/documents"
	"github.com/Voskan/BatchWeaver/internal/lsp/jsonrpc"
	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

// onInitialize forwards initialize to gopls, then merges BatchWeaver
// capabilities into gopls's response so both feature sets are advertised.
func (p *Proxy) onInitialize(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	var ip protocol.InitializeParams
	if err := json.Unmarshal(req.Params, &ip); err == nil {
		root := ""
		if len(ip.WorkspaceFolders) > 0 {
			if path, err := documents.URIToPath(ip.WorkspaceFolders[0].URI); err == nil {
				root = path
			}
		} else if ip.RootURI != "" {
			if path, err := documents.URIToPath(ip.RootURI); err == nil {
				root = path
			}
		}
		p.mu.Lock()
		p.root = root
		p.svc = editor.NewService(root, p.opts.ToolVersion)
		p.mu.Unlock()
	}

	raw, err := p.gopls.Call(ctx, "initialize", req.Params)
	if err != nil {
		var rpcErr *jsonrpc.Error
		if errors.As(err, &rpcErr) {
			return nil, rpcErr
		}
		return nil, jsonrpc.Errorf(jsonrpc.CodeInternalError, "gopls initialize: %v", err)
	}
	merged := mergeInitializeResult(raw)
	return merged, nil
}

// mergeInitializeResult overlays BatchWeaver capabilities onto gopls's
// InitializeResult, preserving every gopls field.
func mergeInitializeResult(goplsRaw json.RawMessage) json.RawMessage {
	var result map[string]any
	if err := json.Unmarshal(goplsRaw, &result); err != nil || result == nil {
		result = map[string]any{}
	}
	caps, _ := result["capabilities"].(map[string]any)
	if caps == nil {
		caps = map[string]any{}
	}
	// BatchWeaver-owned additions. gopls already provides hover/codeAction/codeLens
	// for Go; we advertise them so our merged results are requested, and add our
	// commands to the executeCommand list.
	caps["hoverProvider"] = true
	if _, ok := caps["codeLensProvider"]; !ok {
		caps["codeLensProvider"] = map[string]any{}
	}
	if _, ok := caps["codeActionProvider"]; !ok {
		caps["codeActionProvider"] = true
	}
	caps["executeCommandProvider"] = mergeCommands(caps["executeCommandProvider"], editor.Commands())
	result["capabilities"] = caps

	// Combine server info names so the client shows both.
	result["serverInfo"] = map[string]any{"name": "batchweaver-proxy+gopls"}
	out, _ := json.Marshal(result)
	return out
}

// mergeCommands unions gopls's executeCommand command list with BatchWeaver's.
func mergeCommands(existing any, add []string) map[string]any {
	seen := map[string]bool{}
	var cmds []string
	if m, ok := existing.(map[string]any); ok {
		if list, ok := m["commands"].([]any); ok {
			for _, c := range list {
				if s, ok := c.(string); ok && !seen[s] {
					seen[s] = true
					cmds = append(cmds, s)
				}
			}
		}
	}
	for _, c := range add {
		if !seen[c] {
			seen[c] = true
			cmds = append(cmds, c)
		}
	}
	return map[string]any{"commands": cmds}
}

// isBatchWeaverCommand reports whether an executeCommand targets a BatchWeaver command.
func (p *Proxy) isBatchWeaverCommand(params json.RawMessage) bool {
	var pm protocol.ExecuteCommandParams
	if json.Unmarshal(params, &pm) != nil {
		return false
	}
	for _, c := range editor.Commands() {
		if pm.Command == c {
			return true
		}
	}
	return false
}

// executeBatchWeaverCommand runs a BatchWeaver command locally over the tracked
// overlay.
func (p *Proxy) executeBatchWeaverCommand(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	var pm protocol.ExecuteCommandParams
	if err := json.Unmarshal(req.Params, &pm); err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "%v", err)
	}
	res, svc, ok := p.analyze(ctx)
	if !ok {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInternalError, "not initialized")
	}
	_ = svc
	arg := ""
	if len(pm.Arguments) > 0 {
		_ = json.Unmarshal(pm.Arguments[0], &arg)
	}
	switch pm.Command {
	case editor.CmdScanWorkspace:
		return editor.ScanSummary(res), nil
	case editor.CmdShowGraph:
		return editor.OperationGraphText(res, arg), nil
	case editor.CmdPreview, editor.CmdProve:
		return editor.PreviewText(res, arg), nil
	case editor.CmdDoctor:
		return editor.DoctorText(res), nil
	default:
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "unknown command %q", pm.Command)
	}
}

// analyze runs a snapshot-bound analysis over the tracked overlay.
func (p *Proxy) analyze(ctx context.Context) (*editor.Result, *editor.Service, bool) {
	p.mu.Lock()
	svc := p.svc
	p.mu.Unlock()
	if svc == nil {
		return nil, nil, false
	}
	res, err := svc.Analyze(ctx, p.docs.Overlay())
	if err != nil {
		p.logf("proxy analysis error: %v", err)
		return nil, svc, false
	}
	return res, svc, true
}

// mergeHover returns gopls hover with a BatchWeaver section appended when the
// cursor is on a BatchWeaver operation call.
func (p *Proxy) mergeHover(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	goplsRaw, gerr := p.gopls.Call(ctx, req.Method, req.Params)
	var pm protocol.TextDocumentPositionParams
	_ = json.Unmarshal(req.Params, &pm)
	res, svc, ok := p.analyze(ctx)
	var bw *protocol.Hover
	if ok {
		if doc, has := p.docs.Get(pm.TextDocument.URI); has {
			bw = svc.Hover(res, pm.TextDocument.URI, pm.Position, doc.Mapper())
		}
	}
	if bw == nil {
		if gerr != nil {
			return nil, nil //nolint:nilerr // gopls hover failed and BatchWeaver has none: return no hover, not an error
		}
		return goplsRaw, nil
	}
	// Combine gopls markdown with the BatchWeaver section.
	var gh protocol.Hover
	if gerr == nil && len(goplsRaw) > 0 {
		_ = json.Unmarshal(goplsRaw, &gh)
	}
	combined := gh.Contents.Value
	if combined != "" {
		combined += "\n\n---\n\n"
	}
	combined += bw.Contents.Value
	return protocol.Hover{Contents: protocol.MarkupContent{Kind: "markdown", Value: combined}, Range: bw.Range}, nil
}

// mergeCodeAction concatenates gopls and BatchWeaver code actions.
func (p *Proxy) mergeCodeAction(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	goplsRaw, _ := p.gopls.Call(ctx, req.Method, req.Params)
	var actions []json.RawMessage
	if len(goplsRaw) > 0 {
		_ = json.Unmarshal(goplsRaw, &actions)
	}
	var pm protocol.CodeActionParams
	if json.Unmarshal(req.Params, &pm) == nil {
		if res, svc, ok := p.analyze(ctx); ok {
			if doc, has := p.docs.Get(pm.TextDocument.URI); has {
				for _, a := range svc.CodeActions(res, pm.TextDocument.URI, pm.Range, doc.Mapper()) {
					if raw, err := json.Marshal(a); err == nil {
						actions = append(actions, raw)
					}
				}
			}
		}
	}
	if actions == nil {
		actions = []json.RawMessage{}
	}
	return actions, nil
}

// mergeCodeLens concatenates gopls and BatchWeaver code lenses.
func (p *Proxy) mergeCodeLens(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	goplsRaw, _ := p.gopls.Call(ctx, req.Method, req.Params)
	var lenses []json.RawMessage
	if len(goplsRaw) > 0 {
		_ = json.Unmarshal(goplsRaw, &lenses)
	}
	var pm protocol.CodeLensParams
	if json.Unmarshal(req.Params, &pm) == nil {
		if res, svc, ok := p.analyze(ctx); ok {
			if doc, has := p.docs.Get(pm.TextDocument.URI); has {
				for _, l := range svc.CodeLens(res, pm.TextDocument.URI, doc.Mapper()) {
					if raw, err := json.Marshal(l); err == nil {
						lenses = append(lenses, raw)
					}
				}
			}
		}
	}
	if lenses == nil {
		lenses = []json.RawMessage{}
	}
	return lenses, nil
}

// publishBatchWeaverDiagnostics analyzes the tracked overlay and publishes
// BatchWeaver diagnostics under the "batchweaver" source, separate from gopls's.
func (p *Proxy) publishBatchWeaverDiagnostics(ctx context.Context) {
	res, svc, ok := p.analyze(ctx)
	if !ok {
		return
	}
	for _, uri := range p.docs.OpenURIs() {
		doc, has := p.docs.Get(uri)
		if !has {
			continue
		}
		diags := svc.Diagnostics(res, uri, doc.Mapper())
		if diags == nil {
			diags = []protocol.Diagnostic{}
		}
		_ = p.client.Notify(ctx, "textDocument/publishDiagnostics",
			protocol.PublishDiagnosticsParams{URI: uri, Version: doc.Version, Diagnostics: diags})
	}
}
