# LSP server

The standalone server (`internal/lsp/server`) speaks Language Server Protocol
3.17 over a small internal JSON-RPC implementation (`internal/lsp/jsonrpc`) with
hand-written protocol types (`internal/lsp/protocol`). It imports no third-party
LSP library and no gopls internal packages (see
[ADR 0072](../adr/0072-standalone-lsp-server.md)).

## Transport

stdio is the only transport. In stdio mode nothing is written to stdout except
framed protocol messages; logs go to stderr. Messages use `Content-Length`
framing; a single message is bounded (64 MiB) to resist hostile payloads.
Incoming requests are handled concurrently, replies are matched by ID, and the
serve loop drains in-flight handlers before closing on EOF so no reply is lost.

## Lifecycle and capabilities

The server handles `initialize`, `initialized`, `shutdown`, and `exit`, and
advertises only implemented capabilities: incremental text sync, hover, code
action, code lens, execute command, and multi-root workspace folders, with
`utf-16` position encoding.

## Documents and overlays

Open buffers are authoritative over disk. The document store keeps versions,
rejects out-of-order changes, applies incremental edits, and exposes a
`go/packages` overlay of unsaved bytes. Analysis is debounced so typing does not
trigger repeated package loads, and a generation counter prevents a stale run
from publishing diagnostics.
