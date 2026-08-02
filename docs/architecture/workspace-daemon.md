# Workspace daemon

The workspace daemon (`internal/daemon`) is an optional per-workspace process for
sharing expensive analysis between the CLI, the LSP server, and editor
integrations. This build provides the versioned local protocol
(`batchweaver.daemon/v1alpha1`), discovery, health, and lifecycle; routing CLI
and LSP analysis through the daemon's cache is a documented follow-up (see
[limitations](../limitations/editor.md)).

## Local-only and secure

The daemon listens on a Unix-domain socket, never a network port. The socket
lives in the OS temp directory under a short per-workspace digest name (Unix
socket paths are limited to ~104 bytes) with `0600` permissions; the discovery
record (`.batchweaver/daemon/info.json`, git-ignored) records the socket path,
protocol version, PID, and workspace digest.

## Discovery and lifecycle

`daemon start|status|stop|clean` manage the daemon. Start refuses to launch a
second daemon for a workspace that already has a live one; status distinguishes
not-running, stale (dead or unreachable), and incompatible-protocol; stop asks
the daemon to shut down and cleans stale records; clean removes stale sockets and
records when no live daemon exists. The daemon never kills unrelated processes.
