# Stable Compatibility Report

The repeatable compatibility policy and blocking hosted matrix are established.
Stable-candidate compatibility evidence is not yet established because it must
be rerun for the exact final candidate commit. The current machine matrix is
`release/compatibility.json` and the human report is
[`docs/release/compatibility.md`](../compatibility.md).

Go 1.26.0 and 1.26.5 are blocking full-suite jobs. Linux, macOS, and Windows run
native integration subsets; all published OS/architecture targets cross-build
every package. Module, vendor, go.work, CGO/non-CGO, exact pgx/go-redis/gqlgen/
grpc-go pins, real gopls v0.21.1, and real VS Code 1.85.2/1.131.0 hosts produce
commit-bound JSON evidence. Broader client versions, live PostgreSQL/Redis
Cluster deployments, public candidate archives, and module-proxy installations
are not stable-tested.

Outcome: **blocked for v1**.
