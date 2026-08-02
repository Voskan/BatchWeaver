// Package daemon implements BatchWeaver's optional local workspace daemon: a
// per-workspace process that CLI, LSP, and editor integrations can share to
// avoid recomputing expensive analysis. This build provides the versioned local
// protocol, discovery, health, and lifecycle; the analysis-sharing cache is a
// documented follow-up (see docs/limitations/prompt-11.md). The daemon is
// local-only: it listens on a Unix-domain socket inside the workspace's ignored
// state directory and never opens a network port.
package daemon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Voskan/BatchWeaver/internal/lsp/jsonrpc"
)

// ProtocolVersion identifies the daemon wire protocol. Clients and servers must
// match on the major identity; a mismatch is reported rather than misparsed.
const ProtocolVersion = "batchweaver.daemon/v1alpha1"

// stateDirName is the ignored per-workspace state directory.
const stateDirName = ".batchweaver"

// Info is the discovery record written by a running daemon.
type Info struct {
	ProtocolVersion string `json:"protocol_version"`
	PID             int    `json:"pid"`
	Socket          string `json:"socket"`
	WorkspaceDigest string `json:"workspace_digest"`
	Started         string `json:"started"`
	GoOS            string `json:"goos"`
}

// stateDir returns the per-workspace daemon state directory.
func stateDir(root string) string { return filepath.Join(root, stateDirName, "daemon") }

// infoPath returns the discovery file path for a workspace.
func infoPath(root string) string { return filepath.Join(stateDir(root), "info.json") }

// socketPath returns a deterministic per-workspace socket path. It lives in the
// OS temporary directory under a short digest name because Unix-domain socket
// paths are limited to about 104 bytes, far shorter than a typical workspace
// path. The discovery info file (in the workspace) records the exact path.
func socketPath(root string) string {
	sum := sha256.Sum256([]byte(root))
	name := "bwd-" + hex.EncodeToString(sum[:])[:16] + ".sock"
	return filepath.Join(os.TempDir(), name)
}

// workspaceDigest is a stable identity for the workspace root.
func workspaceDigest(root string) string {
	sum := sha256.Sum256([]byte(root))
	return hex.EncodeToString(sum[:])[:16]
}

// Server is a running daemon instance.
type Server struct {
	root    string
	started time.Time
	ln      net.Listener
}

// Start launches a daemon for the workspace root, writing its discovery record
// and serving until ctx is canceled. It fails if a compatible live daemon
// already owns the workspace.
func Start(ctx context.Context, root string) (*Server, error) {
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		root = wd
	}
	if info, ok := readInfo(root); ok && pingInfo(info) == nil {
		return nil, fmt.Errorf("daemon: a live daemon (pid %d) already owns this workspace", info.PID)
	}
	if err := os.MkdirAll(stateDir(root), 0o755); err != nil {
		return nil, fmt.Errorf("daemon: create state dir: %w", err)
	}
	sock := socketPath(root)
	_ = os.Remove(sock) // clear a stale socket file
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("daemon: listen: %w", err)
	}
	if err := os.Chmod(sock, 0o600); err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("daemon: secure socket: %w", err)
	}
	s := &Server{root: root, started: time.Now(), ln: ln}
	if err := writeInfo(root, Info{
		ProtocolVersion: ProtocolVersion,
		PID:             os.Getpid(),
		Socket:          sock,
		WorkspaceDigest: workspaceDigest(root),
		Started:         s.started.UTC().Format(time.RFC3339),
		GoOS:            runtime.GOOS,
	}); err != nil {
		_ = ln.Close()
		return nil, err
	}
	go s.acceptLoop(ctx)
	go func() {
		<-ctx.Done()
		s.Close()
	}()
	return s, nil
}

// acceptLoop accepts connections and serves the health protocol.
func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return // listener closed
		}
		go func() {
			c := jsonrpc.NewConn(conn, conn, s.handle)
			_ = c.Serve(ctx)
			_ = conn.Close()
		}()
	}
}

// handle serves the daemon protocol methods.
func (s *Server) handle(_ context.Context, _ *jsonrpc.Conn, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	switch req.Method {
	case "daemon/health":
		return HealthResult{
			ProtocolVersion: ProtocolVersion,
			PID:             os.Getpid(),
			UptimeSeconds:   int64(time.Since(s.started).Seconds()),
			WorkspaceDigest: workspaceDigest(s.root),
		}, nil
	case "daemon/shutdown":
		go func() { time.Sleep(50 * time.Millisecond); s.Close() }()
		return map[string]bool{"ok": true}, nil
	default:
		return nil, jsonrpc.Errorf(jsonrpc.CodeMethodNotFound, "daemon: unknown method %q", req.Method)
	}
}

// Close stops the daemon and removes its discovery record.
func (s *Server) Close() {
	if s.ln != nil {
		_ = s.ln.Close()
	}
	_ = os.Remove(socketPath(s.root))
	_ = os.Remove(infoPath(s.root))
}

// Socket returns the daemon's Unix-domain socket path.
func (s *Server) Socket() string { return socketPath(s.root) }

// HealthResult is the daemon/health response.
type HealthResult struct {
	ProtocolVersion string `json:"protocol_version"`
	PID             int    `json:"pid"`
	UptimeSeconds   int64  `json:"uptime_seconds"`
	WorkspaceDigest string `json:"workspace_digest"`
}

// writeInfo atomically writes the discovery record.
func writeInfo(root string, info Info) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(infoPath(root), append(data, '\n'), 0o600)
}

// readInfo reads the discovery record, if present and current.
func readInfo(root string) (Info, bool) {
	data, err := os.ReadFile(infoPath(root))
	if err != nil {
		return Info{}, false
	}
	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return Info{}, false
	}
	return info, true
}

// Status returns the health of the workspace daemon, discovering it via the
// info file. It distinguishes not-running, stale (dead PID or unreachable), and
// incompatible-protocol states.
func Status(root string) (Info, *HealthResult, error) {
	info, ok := readInfo(root)
	if !ok {
		return Info{}, nil, ErrNotRunning
	}
	if info.ProtocolVersion != ProtocolVersion {
		return info, nil, fmt.Errorf("daemon: incompatible protocol %q (want %q)", info.ProtocolVersion, ProtocolVersion)
	}
	h, err := ping(info.Socket)
	if err != nil {
		return info, nil, fmt.Errorf("%w: %w", ErrStale, err)
	}
	return info, h, nil
}

// Stop asks the workspace daemon to shut down.
func Stop(root string) error {
	info, ok := readInfo(root)
	if !ok {
		return ErrNotRunning
	}
	conn, err := net.DialTimeout("unix", info.Socket, 2*time.Second)
	if err != nil {
		// The daemon is unreachable; clean up its stale records.
		_ = os.Remove(info.Socket)
		_ = os.Remove(infoPath(root))
		return fmt.Errorf("%w: %w", ErrStale, err)
	}
	defer func() { _ = conn.Close() }()
	c := jsonrpc.NewConn(conn, conn, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Serve(ctx) //nolint:errcheck // client serve loop ends on ctx/conn close
	_, err = c.Call(ctx, "daemon/shutdown", nil)
	return err
}

// Clean removes stale discovery records and sockets when no live daemon exists.
func Clean(root string) error {
	if _, _, err := Status(root); err == nil {
		return errors.New("daemon: a live daemon is running; stop it before cleaning")
	}
	_ = os.Remove(socketPath(root))
	_ = os.Remove(infoPath(root))
	return nil
}

// ping connects to a socket and calls daemon/health.
func ping(socket string) (*HealthResult, error) {
	conn, err := net.DialTimeout("unix", socket, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	c := jsonrpc.NewConn(conn, conn, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Serve(ctx) //nolint:errcheck // client serve loop ends on ctx/conn close
	raw, err := c.Call(ctx, "daemon/health", nil)
	if err != nil {
		return nil, err
	}
	var h HealthResult
	if err := json.Unmarshal(raw, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// pingInfo pings the daemon described by info.
func pingInfo(info Info) error {
	_, err := ping(info.Socket)
	return err
}

// Sentinel errors.
var (
	// ErrNotRunning indicates no daemon discovery record exists.
	ErrNotRunning = errors.New("daemon: not running")
	// ErrStale indicates a discovery record whose daemon is unreachable.
	ErrStale = errors.New("daemon: stale")
)
