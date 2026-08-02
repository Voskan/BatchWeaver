# BatchWeaver Documentation

Use this index to find the right level of detail. Start with a tutorial if you
are evaluating BatchWeaver, use a guide for a specific task, and use reference
pages when you need exact contracts or compatibility details.

The public documentation portal provides curated entry points for
[all documentation](https://voskan.github.io/BatchWeaver/docs.html),
[runnable examples](https://voskan.github.io/BatchWeaver/examples.html), the
[Go API and CLI](https://voskan.github.io/BatchWeaver/api.html), and the current
[implementation and stable-release status](https://voskan.github.io/BatchWeaver/status.html).

## Start here

- [Verified batching tutorial](tutorials/verified-batching.md) — evaluate the
  complete scan, proof, preview, overlay-test, and rollback workflow.
- [Use BatchWeaver as a Go module](guides/use-as-go-module.md) — import the typed
  library API and understand pkg.go.dev publication.
- [Runtime API](guides/runtime-api.md) — explicitly coalesce compatible calls.
- [Scan a workspace](guides/scan.md) — produce a deterministic candidate report.
- [Plan a transformation](guides/plan-a-transformation.md) — review a proven
  transformation without changing source.

## Learn the model

- [Batching model](concepts/batching-model.md)
- [Operation contracts](concepts/operation-contracts.md)
- [Proof obligations](concepts/proof-obligations.md)
- [Isolation and partitioning](concepts/isolation-and-partitioning.md)
- [Result semantics](concepts/result-semantics.md)
- [Cancellation and deadlines](reference/cancellation-and-deadlines.md)
- [Transformation strategies](concepts/transformation-strategies.md)

## Architecture

- [System overview](architecture/overview.md)
- [Package boundaries](architecture/package-boundaries.md)
- [Static analysis](architecture/static-analysis.md)
- [Semantic proof engine](architecture/semantic-proof-engine.md)
- [Transformation pipeline](architecture/transformation-pipeline.md)
- [Build overlays](architecture/build-overlays.md)
- [Runtime engine](architecture/runtime-engine.md)
- [Generated bridge ABI](architecture/generated-bridge-abi.md)
- [Adapter SDK](architecture/adapter-sdk.md)
- [Editor service](architecture/editor-service.md)
- [Adaptive scheduler](architecture/adaptive-scheduler.md)

The [ADR index](adr/README.md) records significant design decisions and their
tradeoffs.

## Task guides

### Compiler and runtime

- [Prove candidates](guides/proving-candidates.md)
- [Explain a rejection](guides/explaining-a-rejection.md)
- [Build with overlays](guides/build-with-overlays.md)
- [Enable runtime lowering](guides/enable-runtime-lowering.md)
- [Materialize and revert](guides/materialize-and-revert.md)

### Adapters and protocols

- [Create an adapter](guides/create-an-adapter.md)
- [Configure database/sql](guides/configure-database-sql.md)
- [Configure gqlgen](guides/configure-gqlgen.md)
- [Configure gRPC](guides/configure-grpc.md)
- [Configure OpenAPI](guides/configure-openapi.md)
- [Verify adapters](guides/verify-an-adapter.md)

### Editor and operations

- [VS Code](guides/vscode.md)
- [Neovim](guides/neovim.md)
- [Emacs/Eglot](guides/emacs-eglot.md)
- [Helix](guides/helix.md)
- [Zed](guides/zed.md)
- [Production rollout](adoption/production-rollout.md)
- [Troubleshooting](guides/editor-troubleshooting.md)

## Reference

- [Configuration schema](reference/configuration.md)
- [Typed declarations](reference/typed-declarations.md)
- [Provider contract](reference/provider-contract.md)
- [Diagnostic codes](reference/diagnostic-codes.md)
- [Exit codes](reference/exit-codes.md)
- [Runtime ABI](reference/runtime-abi.md)
- [Proof schema](reference/proof-schema.md)
- [Transformation schema](reference/transformation-schema.md)
- [Daemon protocol](reference/daemon-protocol.md)
- [Editor support matrix](reference/editor-support-matrix.md)

## Limitations

- [Semantic proof](limitations/semantic-proof-limitations.md)
- [Transformation](limitations/transformation.md)
- [Runtime lowering](limitations/runtime-lowering.md)
- [Backend adapters](limitations/backend-adapters.md)
- [Network adapters](limitations/network-adapters.md)
- [Adaptive scheduling](limitations/adaptive.md)
- [Editor integration](limitations/editor.md)
- [Repository-wide known issues](../KNOWN-ISSUES.md)

## Releases and maintenance

- [Compatibility](release/compatibility.md)
- [Release policy](release/release-policy.md)
- [Release checklist](release/release-checklist.md)
- [Rollback](release/rollback.md)
- [Security posture](release/security-report.md)
- [Performance evidence](release/performance-report.md)
- [Stable-release evidence and decision](release/v1.0.0/stable-release-decision.md)

## Contribute

- [Development setup](development/getting-started.md)
- [Quality gates](development/quality-gates.md)
- [Repository conventions](development/repository-conventions.md)
- [Contributing policy](../CONTRIBUTING.md)
- [Security policy](../SECURITY.md)
