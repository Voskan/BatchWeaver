# 73. Optional gopls-compatible proxy

Date: 2026-08-02

## Status

Accepted

## Context

Some editors support only one Go language server, so BatchWeaver cannot run as a
sidecar there.

## Decision

Provide an optional proxy mode: BatchWeaver launches the user's gopls, forwards
standard Go traffic to it, and merges BatchWeaver features on top, speaking only
the public LSP protocol. Forwarded requests are re-issued through the destination
connection so request-ID namespacing is automatic. gopls is never downloaded,
patched, or imported.

## Consequences

Single-server editors get both feature sets. The proxy must correctly route
server-initiated requests, cancellation, and diagnostics, which is covered by
tests.
