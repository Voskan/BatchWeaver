package daemon

import (
	"context"
	"errors"
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

func TestCleanWhenNotRunning(t *testing.T) {
	root := t.TempDir()
	if err := Clean(root); err != nil {
		t.Errorf("clean on empty workspace: %v", err)
	}
}
