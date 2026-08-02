# Analysis cache schema and policy

Schema: `batchweaver.analysis-cache/v1alpha1`.

The workspace daemon's cache is an optimization, never a source of truth. A
missing, stale, incompatible, corrupt, or evicted entry is recomputed from the
workspace and authoritative editor overlays.

## Key dimensions

Every key is a SHA-256 identity over:

- the canonical workspace identity and a content digest of relevant `.go`,
  `.mod`, `.sum`, `.work`, YAML, and JSON inputs;
- authoritative overlay path/content digests (overlay bytes are not persisted);
- BatchWeaver tool version and running Go toolchain version;
- GOOS, GOARCH, CGO, tests, build tags, and package patterns;
- reproducibility mode; and
- analysis, proof, transformation, and strategy schema versions.

Tags and package patterns are normalized before hashing. Workspace paths are
hashed and raw content is represented by digests. Symlinks and overlays outside
the canonical workspace are rejected rather than shared across isolation
boundaries.

## Disk envelope

The disk tier stores JSON envelopes below `.batchweaver/daemon/cache/v1`:

```json
{
  "schema": "batchweaver.analysis-cache/v1alpha1",
  "key": "sha256:<64 lowercase hexadecimal characters>",
  "digest": "sha256:<payload digest>",
  "payload": "<base64-encoded serialized analysis snapshot>"
}
```

The directory is `0700`, files are atomically replaced with mode `0600`, and
the default disk bound is 512 MiB. Envelope/schema/key/digest validation occurs
before reuse. Validation failure removes the entry and triggers recomputation.

## Memory, concurrency, and lifecycle

The default memory tier is a 32-entry, 128 MiB LRU. Concurrent misses for an
identical key are single-flight; canceled waiters do not block the leader.
Restarting a daemon drops memory state but can reuse validated disk entries.
`batchweaver daemon clean` removes the disk tier only when the workspace has no
live daemon.

## Observability and privacy

Health reports expose only hit, miss, disk-hit, eviction, and corruption counts
plus entry/byte occupancy. Per-request status exposes only `hit` and one of
`memory`, `disk`, `shared`, or `compute`. No remote telemetry is emitted, and no
source, overlay, raw cache key, path, payload value, or tenant identity is
included in observability output.
