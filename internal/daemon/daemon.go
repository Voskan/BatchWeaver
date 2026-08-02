// Package daemon implements BatchWeaver's optional local workspace daemon: a
// per-workspace process that CLI, LSP, and editor integrations can share to
// avoid recomputing expensive analysis. The daemon owns a bounded,
// content-addressed memory/disk cache and exposes privacy-safe cache health. It
// is local-only: it listens on a Unix-domain socket associated with the
// workspace's ignored state directory and never opens a network port.
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
	"sort"
	"sync"
	"time"

	"github.com/Voskan/BatchWeaver/internal/analysis"
	"github.com/Voskan/BatchWeaver/internal/analysiscache"
	"github.com/Voskan/BatchWeaver/internal/lsp/jsonrpc"
	"github.com/Voskan/BatchWeaver/internal/proof"
	"github.com/Voskan/BatchWeaver/internal/transform"
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
	cache   *analysiscache.Cache
	close   sync.Once
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
	canonical, err := canonicalRoot(root)
	if err != nil {
		return nil, err
	}
	root = canonical
	if info, ok := readInfo(root); ok && pingInfo(info) == nil {
		return nil, fmt.Errorf("daemon: a live daemon (pid %d) already owns this workspace", info.PID)
	}
	if err := os.MkdirAll(stateDir(root), 0o700); err != nil {
		return nil, fmt.Errorf("daemon: create state dir: %w", err)
	}
	if err := os.Chmod(stateDir(root), 0o700); err != nil {
		return nil, fmt.Errorf("daemon: secure state dir: %w", err)
	}
	cache, err := analysiscache.New(filepath.Join(stateDir(root), "cache", "v1"), analysiscache.Config{})
	if err != nil {
		return nil, fmt.Errorf("daemon: create analysis cache: %w", err)
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
	s := &Server{root: root, started: time.Now(), ln: ln, cache: cache}
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
func (s *Server) handle(ctx context.Context, _ *jsonrpc.Conn, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	switch req.Method {
	case "daemon/health":
		return HealthResult{
			ProtocolVersion: ProtocolVersion,
			PID:             os.Getpid(),
			UptimeSeconds:   int64(time.Since(s.started).Seconds()),
			WorkspaceDigest: workspaceDigest(s.root),
			Cache:           s.cache.Snapshot(),
		}, nil
	case "analysis/analyze":
		return s.handleAnalysis(ctx, req)
	case "daemon/shutdown":
		go func() { time.Sleep(50 * time.Millisecond); s.Close() }()
		return map[string]bool{"ok": true}, nil
	default:
		return nil, jsonrpc.Errorf(jsonrpc.CodeMethodNotFound, "daemon: unknown method %q", req.Method)
	}
}

// Close stops the daemon and removes its discovery record.
func (s *Server) Close() {
	s.close.Do(func() {
		if s.ln != nil {
			_ = s.ln.Close()
		}
		_ = os.Remove(socketPath(s.root))
		_ = os.Remove(infoPath(s.root))
	})
}

// Socket returns the daemon's Unix-domain socket path.
func (s *Server) Socket() string { return socketPath(s.root) }

// HealthResult is the daemon/health response.
type HealthResult struct {
	ProtocolVersion string              `json:"protocol_version"`
	PID             int                 `json:"pid"`
	UptimeSeconds   int64               `json:"uptime_seconds"`
	WorkspaceDigest string              `json:"workspace_digest"`
	Cache           analysiscache.Stats `json:"cache"`
}

// AnalysisParams is the local daemon wire request. Overlay values are carried
// only over the workspace-owned Unix socket and are never persisted.
type AnalysisParams struct {
	Patterns     []string              `json:"patterns"`
	BuildContext analysis.BuildContext `json:"build_context"`
	Reproducible bool                  `json:"reproducible"`
	ToolVersion  string                `json:"tool_version"`
	Overlay      map[string][]byte     `json:"overlay,omitempty"`
}

// AnalysisResult is one cached or newly computed immutable snapshot.
type AnalysisResult struct {
	Snapshot *analysis.Snapshot   `json:"snapshot"`
	Cache    analysiscache.Result `json:"cache"`
}

func (s *Server) handleAnalysis(ctx context.Context, req *jsonrpc.Request) (any, *jsonrpc.Error) {
	var params AnalysisParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "daemon analysis: %v", err)
	}
	patterns := append([]string(nil), params.Patterns...)
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	sort.Strings(patterns)
	sourceDigest, err := analysiscache.SourceDigest(s.root, params.Overlay)
	if err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInvalidParams, "daemon analysis source: %v", err)
	}
	goos := params.BuildContext.GOOS
	if goos == "" {
		goos = envOr("GOOS", runtime.GOOS)
	}
	goarch := params.BuildContext.GOARCH
	if goarch == "" {
		goarch = envOr("GOARCH", runtime.GOARCH)
	}
	key := analysiscache.Key(analysiscache.KeyInput{
		Workspace: s.root, ToolVersion: params.ToolVersion, GoVersion: runtime.Version(),
		GOOS: goos, GOARCH: goarch, CGOEnabled: params.BuildContext.CGOEnabled,
		Tests: params.BuildContext.Tests, Reproducible: true, Tags: params.BuildContext.Tags,
		Patterns: patterns, SourceDigest: sourceDigest, AnalysisSchema: analysis.SchemaVersion,
		ProofSchema: proof.SchemaVersion, TransformSchema: transform.SchemaVersion,
		StrategyVersion: transform.StrategyVersion,
	})
	payload, cacheResult, err := s.cache.GetOrCompute(ctx, key, func() ([]byte, error) {
		snapshot, err := analysis.Analyze(ctx, analysis.Request{
			Patterns: patterns, BuildContext: params.BuildContext, Reproducible: true,
			ToolVersion: params.ToolVersion, Dir: s.root, Overlay: params.Overlay,
		})
		if err != nil {
			return nil, err
		}
		return json.Marshal(snapshot)
	})
	if err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInternalError, "daemon analysis: %v", err)
	}
	var snapshot analysis.Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return nil, jsonrpc.Errorf(jsonrpc.CodeInternalError, "daemon analysis snapshot: %v", err)
	}
	if !params.Reproducible {
		snapshot.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	return AnalysisResult{Snapshot: &snapshot, Cache: cacheResult}, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func canonicalRoot(root string) (string, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("daemon: resolve workspace: %w", err)
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("daemon: resolve workspace: %w", err)
	}
	return abs, nil
}

// writeInfo atomically writes the discovery record.
func writeInfo(root string, info Info) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(stateDir(root), ".info-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, infoPath(root))
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
	canonical, err := canonicalRoot(root)
	if err != nil {
		return Info{}, nil, err
	}
	root = canonical
	info, ok := readInfo(root)
	if !ok {
		return Info{}, nil, ErrNotRunning
	}
	if info.ProtocolVersion != ProtocolVersion {
		return info, nil, fmt.Errorf("daemon: incompatible protocol %q (want %q)", info.ProtocolVersion, ProtocolVersion)
	}
	if info.WorkspaceDigest != workspaceDigest(root) || info.Socket != socketPath(root) {
		return info, nil, fmt.Errorf("%w: discovery record does not belong to this workspace", ErrStale)
	}
	h, err := ping(info.Socket)
	if err != nil {
		return info, nil, fmt.Errorf("%w: %w", ErrStale, err)
	}
	return info, h, nil
}

// Stop asks the workspace daemon to shut down.
func Stop(root string) error {
	canonical, err := canonicalRoot(root)
	if err != nil {
		return err
	}
	root = canonical
	info, ok := readInfo(root)
	if !ok {
		return ErrNotRunning
	}
	if info.WorkspaceDigest != workspaceDigest(root) || info.Socket != socketPath(root) {
		return fmt.Errorf("%w: discovery record does not belong to this workspace", ErrStale)
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
	canonical, err := canonicalRoot(root)
	if err != nil {
		return err
	}
	root = canonical
	if _, _, err := Status(root); err == nil {
		return errors.New("daemon: a live daemon is running; stop it before cleaning")
	}
	_ = os.Remove(socketPath(root))
	_ = os.Remove(infoPath(root))
	_ = os.RemoveAll(filepath.Join(stateDir(root), "cache"))
	return nil
}

// Analyze routes one request through the compatible daemon for root.
func Analyze(ctx context.Context, root string, params AnalysisParams) (*AnalysisResult, error) {
	canonical, err := canonicalRoot(root)
	if err != nil {
		return nil, err
	}
	info, ok := readInfo(canonical)
	if !ok {
		return nil, ErrNotRunning
	}
	if info.ProtocolVersion != ProtocolVersion || info.WorkspaceDigest != workspaceDigest(canonical) || info.Socket != socketPath(canonical) {
		return nil, fmt.Errorf("%w: incompatible or isolated discovery record", ErrStale)
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", info.Socket)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStale, err)
	}
	defer func() { _ = conn.Close() }()
	client := jsonrpc.NewConn(conn, conn, nil)
	serveCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = client.Serve(serveCtx) }()
	raw, err := client.Call(ctx, "analysis/analyze", params)
	if err != nil {
		return nil, err
	}
	var result AnalysisResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.Snapshot == nil {
		return nil, errors.New("daemon: analysis returned no snapshot")
	}
	return &result, nil
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
