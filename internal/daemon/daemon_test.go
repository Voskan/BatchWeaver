package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDaemonLifecycle(t *testing.T) {
	root := t.TempDir()

	// Not running initially.
	if _, _, err := Status(root); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("expected not-running, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, err := Start(ctx, root)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Close()

	// Running: health responds.
	info, health, err := Status(root)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if health.ProtocolVersion != ProtocolVersion || info.PID == 0 {
		t.Errorf("unexpected health/info: %+v %+v", health, info)
	}

	// Starting a second daemon for the same workspace is refused.
	if _, err := Start(ctx, root); err == nil {
		t.Error("expected refusal to start a second daemon")
	}

	// Stop it and confirm it goes away.
	if err := Stop(root); err != nil {
		t.Fatalf("stop: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, _, err := Status(root); errors.Is(err, ErrNotRunning) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Error("daemon did not stop within deadline")
}

func TestDaemonAnalysisCacheInvalidatesOverlaySourceAndRestarts(t *testing.T) {
	root := t.TempDir()
	writeDaemonFixture(t, filepath.Join(root, "go.mod"), "module example.com/daemonfixture\n\ngo 1.26\n")
	goFile := filepath.Join(root, "service.go")
	writeDaemonFixture(t, goFile, "package daemonfixture\n\nfunc Value() int { return 1 }\n")
	ctx, cancel := context.WithCancel(context.Background())
	server, err := Start(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	params := AnalysisParams{Patterns: []string{"./..."}, Reproducible: true, ToolVersion: "test"}
	first, err := Analyze(context.Background(), root, params)
	if err != nil || first.Cache.Hit {
		t.Fatalf("first result=%+v err=%v", first, err)
	}
	second, err := Analyze(context.Background(), root, params)
	if err != nil || !second.Cache.Hit {
		t.Fatalf("second result=%+v err=%v", second, err)
	}
	params.Overlay = map[string][]byte{goFile: []byte("package daemonfixture\n\nfunc Value() int { return 2 }\n")}
	overlayMiss, err := Analyze(context.Background(), root, params)
	if err != nil || overlayMiss.Cache.Hit {
		t.Fatalf("overlay miss=%+v err=%v", overlayMiss, err)
	}
	overlayHit, err := Analyze(context.Background(), root, params)
	if err != nil || !overlayHit.Cache.Hit {
		t.Fatalf("overlay hit=%+v err=%v", overlayHit, err)
	}
	params.Overlay = nil
	writeDaemonFixture(t, goFile, "package daemonfixture\n\nfunc Value() int { return 3 }\n")
	sourceMiss, err := Analyze(context.Background(), root, params)
	if err != nil || sourceMiss.Cache.Hit {
		t.Fatalf("source miss=%+v err=%v", sourceMiss, err)
	}
	_, health, err := Status(root)
	if err != nil || health.Cache.Hits < 2 || health.Cache.Misses < 3 {
		t.Fatalf("health=%+v err=%v", health, err)
	}
	server.Close()
	cancel()

	restartCtx, restartCancel := context.WithCancel(context.Background())
	defer restartCancel()
	restarted, err := Start(restartCtx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	afterRestart, err := Analyze(context.Background(), root, params)
	if err != nil || !afterRestart.Cache.Hit || afterRestart.Cache.Source != "disk" {
		t.Fatalf("restart result=%+v err=%v", afterRestart, err)
	}
}

func TestDaemonRejectsOverlayOutsideWorkspace(t *testing.T) {
	root := t.TempDir()
	writeDaemonFixture(t, filepath.Join(root, "go.mod"), "module example.com/isolation\n\ngo 1.26\n")
	writeDaemonFixture(t, filepath.Join(root, "main.go"), "package isolation\n")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, err := Start(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	outside := filepath.Join(t.TempDir(), "outside.go")
	writeDaemonFixture(t, outside, "package outside\n")
	_, err = Analyze(context.Background(), root, AnalysisParams{
		ToolVersion: "test", Reproducible: true,
		Overlay: map[string][]byte{outside: []byte("package changed\n")},
	})
	if err == nil {
		t.Fatal("outside-workspace overlay was accepted")
	}
}

func TestCleanWhenNotRunning(t *testing.T) {
	root := t.TempDir()
	if err := Clean(root); err != nil {
		t.Errorf("clean on empty workspace: %v", err)
	}
}

func writeDaemonFixture(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
