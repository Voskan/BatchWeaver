// Package jsonrpc implements the minimal JSON-RPC 2.0 surface BatchWeaver's
// language server needs, framed with the Language Server Protocol's
// Content-Length header convention.
//
// It is a small internal implementation (no third-party dependency and no code
// copied from gopls) covering exactly what the server and proxy require:
// Content-Length framing, requests, responses, notifications, concurrent request
// handling, per-request cancellation, ordered writes, bounded message size, and
// graceful shutdown on EOF. See docs/architecture/lsp-server.md and
// docs/adr/0072-standalone-lsp-server.md for the rationale.
package jsonrpc
