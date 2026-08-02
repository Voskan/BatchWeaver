# 78. Virtual documents for proof, diff, graph, and report views

Date: 2026-08-02

## Status

Accepted

## Context

Some views (proof certificates, diffs, graphs, reports) have no physical file.

## Decision

Surface them as read-only, snapshot-bound, size-limited, privacy-safe content,
always available as text/DOT/JSON so no browser is required. A local webview is
optional and, if used, loopback-only with a strict CSP and no remote assets.

## Consequences

Rich views are available without writing files or requiring a browser.
