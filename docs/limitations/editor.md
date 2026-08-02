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
(protocol, discovery, health, lifecycle, shared analysis cache) and `daemon` CLI; `batchweaver lsp`,
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

- **Analysis cache is opt-in by process:** when a compatible workspace daemon is
  running, CLI and LSP requests share its bounded memory/disk cache. Without a
  daemon they fall back to correct in-process analysis. No daemon is started
  implicitly, and no source, overlay, cache key, or telemetry is exported.
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
- **Pinned host compatibility:** CI runs a real gopls v0.21.1 proxy process and
  real minimum/current VS Code Extension Hosts. Other gopls and VS Code versions
  are not claimed until added to the blocking matrix. Community editor UIs remain
  protocol-supported rather than host-E2E-tested. Inlay hints are intentionally
  minimal/disabled by default to avoid noise.
