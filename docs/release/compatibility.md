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

Concrete pgx, go-redis, gqlgen, and grpc-go client bindings are not present and
therefore have no supported client-version claim. Framework-neutral parsing,
partitioning, and contract verification are not a substitute for an integration
test against those clients.
