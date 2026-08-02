# Daemon protocol reference

Protocol version: `batchweaver.daemon/v1alpha1`. Transport: JSON-RPC 2.0 over a
Unix-domain socket, local only, `0600` permissions, in the OS temp directory
under a per-workspace digest name. Discovery record:
`.batchweaver/daemon/info.json` (git-ignored).

## Methods

| Method | Request | Response |
| --- | --- | --- |
| `daemon/health` | none | `{protocol_version, pid, uptime_seconds, workspace_digest}` |
| `daemon/shutdown` | none | `{ok: true}` (daemon exits shortly after) |

## Discovery states

- **not running:** no info file.
- **running:** info file present and health responds.
- **stale:** info file present but the daemon is unreachable (dead PID or removed
  socket); `daemon clean` removes stale records.
- **incompatible:** info file records a different protocol version.

The daemon never opens a network port and never kills unrelated processes.
