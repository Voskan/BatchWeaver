package editor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Voskan/BatchWeaver/internal/daemon"
)

func TestServiceUsesDaemonCacheAndInvalidatesUnsavedBuffer(t *testing.T) {
	root := t.TempDir()
	writeEditorCacheFixture(t, filepath.Join(root, "go.mod"), "module example.com/editorcache\n\ngo 1.26\n")
	goFile := filepath.Join(root, "service.go")
	writeEditorCacheFixture(t, goFile, "package editorcache\n\nfunc Value() int { return 1 }\n")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, err := daemon.Start(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	service := NewService(root, "test")
	if _, cacheResult, err := service.AnalyzeWithCache(context.Background(), nil); err != nil || cacheResult.Hit {
		t.Fatalf("first analysis cache=%+v err=%v", cacheResult, err)
	}
	if _, cacheResult, err := service.AnalyzeWithCache(context.Background(), nil); err != nil || !cacheResult.Hit {
		t.Fatalf("second analysis cache=%+v err=%v", cacheResult, err)
	}
	overlay := map[string][]byte{goFile: []byte("package editorcache\n\nfunc Value() int { return 2 }\n")}
	if _, cacheResult, err := service.AnalyzeWithCache(context.Background(), overlay); err != nil || cacheResult.Hit {
		t.Fatalf("changed buffer cache=%+v err=%v", cacheResult, err)
	}
	if _, cacheResult, err := service.AnalyzeWithCache(context.Background(), overlay); err != nil || !cacheResult.Hit {
		t.Fatalf("stable buffer cache=%+v err=%v", cacheResult, err)
	}
}

func writeEditorCacheFixture(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
