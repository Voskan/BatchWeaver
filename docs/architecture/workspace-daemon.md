# Workspace daemon

The workspace daemon (`internal/daemon`) is an optional per-workspace process for
sharing expensive analysis between the CLI, the LSP server, and editor
integrations. It provides the versioned local protocol
(`batchweaver.daemon/v1alpha1`), discovery, health, lifecycle, and a shared
content-addressed analysis cache. CLI and LSP clients use the cache when a
compatible daemon is running and safely fall back to local analysis otherwise.

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

## Cache model

The memory tier is a 32-entry, 128 MiB LRU. The persistent tier is bounded to
512 MiB and stored below `.batchweaver/daemon/cache/v1` with directory mode
`0700` and files mode `0600`. Concurrent requests for one key are single-flight.
Disk envelopes contain a schema, a content-addressed identity, an integrity
digest, and the serialized immutable analysis snapshot. Invalid or corrupt
entries are deleted and recomputed.

Keys include a digest of relevant Go/module/workspace/config files and
authoritative unsaved overlays, plus the canonical workspace identity,
BatchWeaver and Go versions, target GOOS/GOARCH/CGO/tests/tags, package patterns,
and analysis, proof, transformation, and strategy schema versions. Source and
overlay bytes are hashed but never stored in the cache. Symlinks and overlay
paths escaping the workspace are rejected.

`batchweaver daemon status` reports only counters and occupancy. `batchweaver
scan --cache-status` reports `local`, `memory`, `disk`, `shared`, or `compute`
and a hit boolean to stderr. These are local diagnostics, not telemetry.

Stopping and restarting the daemon preserves validated disk entries. Source,
config, build-context, toolchain, package-pattern, or schema changes naturally
produce new keys. `daemon clean` removes discovery, socket, and cache state only
when no live daemon owns the workspace.
