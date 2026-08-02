# Editor Integration Limitations

## Implemented

A standalone LSP 3.17 server (`internal/lsp`) over a small internal JSON-RPC
implementation with hand-written protocol types and no gopls internal imports; a
canonical UTF-16/byte position mapper; a document store with versions,
out-of-order rejection, incremental edits, and unsaved-buffer overlays fed to
analysis; an editor service producing deterministic diagnostics (including the
`BW1001` batching opportunity), hover, code lenses, and preview code actions; an
optional gopls proxy that launches gopls, forwards traffic with automatic ID
namespacing, merges `initialize` capabilities and hover/code-action/code-lens
results, and keeps diagnostics separated by source; a local workspace daemon
(protocol, discovery, health, lifecycle) and `daemon` CLI; `batchweaver lsp`,
`batchweaver editor doctor`, and `daemon` commands; a VS Code extension source
tree (sidecar/proxy, commands, settings, status bar, output channel, workspace
trust); editor setup guides for VS Code, Neovim, Emacs/Eglot, Helix, and Zed; an
editor support matrix; docs, ADRs 0072–0081; and tests (protocol round-trip,
UTF-16 mapping, document sync, capability merge, daemon lifecycle, gopls-internal
import ban) including race and fuzz coverage.

## Deliberately not implemented (out of scope)

- Changes to, imports from, or a fork of the upstream gopls codebase; a fake
  "gopls plugin" interface.
- Automatic application of source edits; automatic materialization; automatic
  benchmark/test execution on keystrokes.
- Remote hosted LSP, cloud telemetry, source upload, or a browser IDE.
- Publishing the VS Code extension to the Marketplace.

## Honest notes on this build

- **Analysis runs in-process** in the LSP server and CLI; routing analysis
  through the workspace daemon's shared cache is a documented follow-up. The
  daemon protocol, discovery, health, and lifecycle are real and tested.
- **Transformation preview** shows the operation binding, structural context, and
  candidate evidence, and points to `batchweaver prove` / `transform diff` for the
  exact deterministic diff and proof certificate; wiring the full diff and
  versioned `WorkspaceEdit` apply flow through the LSP is the next increment.
- **Diagnostic renumbering:** the specification's illustrative `BW8xxx` LSP codes
  are renumbered to `BW9xxx` to keep diagnostic ranges distinct per stage
  (documented in the LSP diagnostics reference).
- **VS Code extension** is delivered as source. `npm ci`, lint, typecheck,
  compile, and packaging require network access and are run in CI; the
  `package-lock.json` is generated there rather than committed in this phase.
- **Real-gopls compatibility matrix and VS Code headless E2E** are CI-gated; this
  build verifies proxy capability merge with deterministic unit tests and a manual
  gopls launch smoke test. Inlay hints are intentionally minimal/disabled by
  default to avoid noise.
