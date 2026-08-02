package server

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/Voskan/BatchWeaver/internal/lsp/jsonrpc"
	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

func TestInitializeCapabilities(t *testing.T) {
	srvSide, cliSide := net.Pipe()
	defer func() { _ = srvSide.Close() }()
	defer func() { _ = cliSide.Close() }()

	s := New(Options{ToolVersion: "test"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go s.Run(ctx, srvSide, srvSide) //nolint:errcheck

	client := jsonrpc.NewConn(cliSide, cliSide, nil)
	go client.Serve(ctx) //nolint:errcheck

	raw, err := client.Call(ctx, "initialize", protocol.InitializeParams{RootURI: "file:///tmp"})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	var res protocol.InitializeResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatal(err)
	}
	if res.ServerInfo == nil || res.ServerInfo.Name != ServerName {
		t.Errorf("serverInfo = %+v", res.ServerInfo)
	}
	if !res.Capabilities.HoverProvider {
		t.Error("expected hoverProvider")
	}
	if res.Capabilities.PositionEncoding != "utf-16" {
		t.Errorf("positionEncoding = %q", res.Capabilities.PositionEncoding)
	}
	if res.Capabilities.TextDocumentSync == nil || res.Capabilities.TextDocumentSync.Change != protocol.SyncIncremental {
		t.Error("expected incremental text sync")
	}
	if res.Capabilities.ExecuteCommandProvider == nil || len(res.Capabilities.ExecuteCommandProvider.Commands) == 0 {
		t.Error("expected execute-command commands")
	}
}

func TestUnknownMethod(t *testing.T) {
	srvSide, cliSide := net.Pipe()
	defer func() { _ = srvSide.Close() }()
	defer func() { _ = cliSide.Close() }()
	s := New(Options{ToolVersion: "test"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go s.Run(ctx, srvSide, srvSide) //nolint:errcheck
	client := jsonrpc.NewConn(cliSide, cliSide, nil)
	go client.Serve(ctx) //nolint:errcheck
	if _, err := client.Call(ctx, "textDocument/nonexistent", struct{}{}); err == nil {
		t.Fatal("expected method-not-found error")
	}
}
