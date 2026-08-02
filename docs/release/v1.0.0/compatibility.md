# Stable Compatibility Report

Stable compatibility is not established. The current machine matrix is
`release/compatibility.json` and the human report is
[`docs/release/compatibility.md`](../compatibility.md).

Go 1.26.5 is the only declared toolchain. Linux and macOS CI passed at the
audited candidate commit. Windows exposed a CRLF/LF golden-file defect; the
source correction requires a green hosted rerun. Cross-built archives are local
evidence only. Real pgx, go-redis, gqlgen, grpc-go, gopls/VS Code host, public
archive, and module-proxy installations are not stable-tested.

Outcome: **blocked for v1**.
