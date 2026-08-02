# 72. Standalone LSP server, not a gopls internal plugin

Date: 2026-08-02

## Status

Accepted

## Context

gopls has no stable public plugin API; its implementation packages are internal
and its CLI is experimental. BatchWeaver needs editor features for Go.

## Decision

Implement a standalone BatchWeaver language server that speaks the public LSP
wire protocol, with a small internal JSON-RPC implementation and hand-written
protocol types. Do not import `golang.org/x/tools/gopls/internal/*` or copy its
protocol structs. A CI/lint test enforces the import ban.

## Consequences

BatchWeaver is independent of gopls internals and stable across gopls releases.
It reimplements only the protocol surface it needs, at the cost of maintaining
those types.
