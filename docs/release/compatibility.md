# Compatibility Report

The machine-readable source is `release/compatibility.json`. As of 2026-08-02,
Go 1.26.5 on macOS arm64 is the native release environment. Linux, macOS amd64,
Linux arm64, and Windows amd64 archive builds are cross-compiled with CGO
disabled and structurally verified. CI provides native Linux, macOS, and Windows
jobs, but this report does not reinterpret a skipped or unavailable runner as a
pass.

The Go release pin was verified against the
[official Go release history](https://go.dev/doc/devel/release): 1.26.5 is a
security patch release dated 2026-07-07. The gopls pin is v0.21.1 and was
verified against the [official golang/tools release](https://github.com/golang/tools/releases/tag/gopls/v0.21.1).
The VSIX packager pin follows its [official npm package](https://www.npmjs.com/package/@vscode/vsce),
whose current line requires Node 22. Pins are updated through a reviewed pull
request, never by an unreviewed "latest" lookup.

The source compatibility matrix now pins pgx v5.10.0, go-redis v9.21.0, gqlgen
v0.17.94, and grpc-go v1.83.0. pgx is exercised through pgxmock's implementation
of the public pgx.Rows contract; go-redis through the real client and miniredis;
gqlgen through its public extension interfaces; and grpc-go through a real
bufconn transport. Live PostgreSQL and multi-node Redis Cluster deployments are
still environment-specific acceptance tests and are not implied by the
hermetic matrix.
