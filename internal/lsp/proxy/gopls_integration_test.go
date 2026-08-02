package proxy

import (
	"context"
	"encoding/json"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Voskan/BatchWeaver/internal/lsp/jsonrpc"
	"github.com/Voskan/BatchWeaver/internal/lsp/protocol"
)

func TestRealGoplsCompatibility(t *testing.T) {
	gopls := os.Getenv("BATCHWEAVER_TEST_GOPLS")
	if gopls == "" {
		t.Skip("set BATCHWEAVER_TEST_GOPLS to run the real-process compatibility test")
	}
	root := t.TempDir()
	writeGoplsFixture(t, filepath.Join(root, "go.mod"), "module example.com/goplscompat\n\ngo 1.26\n")
	writeGoplsFixture(t, filepath.Join(root, "main.go"), "package goplscompat\n\nfunc Value() int { return 1 }\n")

	serverSide, clientSide := net.Pipe()
	defer func() { _ = serverSide.Close() }()
	defer func() { _ = clientSide.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p := New(Options{GoplsCommand: gopls, ToolVersion: "compatibility-test"})
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx, serverSide, serverSide) }()
	client := jsonrpc.NewConn(clientSide, clientSide, nil)
	go func() { _ = client.Serve(ctx) }()

	rootURI := (&url.URL{Scheme: "file", Path: filepath.ToSlash(root)}).String()
	raw, err := client.Call(ctx, "initialize", protocol.InitializeParams{RootURI: rootURI})
	if err != nil {
		t.Fatalf("initialize real gopls proxy: %v", err)
	}
	var initialized map[string]any
	if err := json.Unmarshal(raw, &initialized); err != nil {
		t.Fatal(err)
	}
	serverInfo, _ := initialized["serverInfo"].(map[string]any)
	if serverInfo["name"] != "batchweaver-proxy+gopls" {
		t.Fatalf("server info = %#v", serverInfo)
	}
	capabilities, _ := initialized["capabilities"].(map[string]any)
	if capabilities["completionProvider"] == nil || capabilities["executeCommandProvider"] == nil {
		t.Fatalf("merged capabilities omit gopls or BatchWeaver providers: %#v", capabilities)
	}
	if err := client.Notify(ctx, "initialized", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Call(ctx, "shutdown", nil); err != nil {
		t.Fatalf("shutdown real gopls proxy: %v", err)
	}
	if err := client.Notify(ctx, "exit", nil); err != nil {
		t.Fatal(err)
	}
	// Editors close the stdio transport after exit; net.Pipe must model that EOF
	// explicitly so the client-facing serve loop can return.
	_ = clientSide.Close()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("real gopls proxy did not exit: %v", ctx.Err())
	}
}

func writeGoplsFixture(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
