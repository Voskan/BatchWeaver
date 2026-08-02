# Stable Compatibility Report

Stable compatibility is not established. The current machine matrix is
`release/compatibility.json` and the human report is
[`docs/release/compatibility.md`](../compatibility.md).

Go 1.26.5 is the only declared toolchain. Linux, macOS, and Windows hosted
build/test jobs pass after correcting the Windows CRLF/LF fixture checkout.
Cross-built archives remain local evidence only. Concrete pgx, go-redis,
gqlgen, and grpc-go packages now have hermetic integration coverage, but broader
client versions, live PostgreSQL/Redis Cluster deployments, gopls/VS Code hosts,
public candidate archives, and module-proxy installations are not stable-tested.

Outcome: **blocked for v1**.
