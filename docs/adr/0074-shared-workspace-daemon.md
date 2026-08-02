# 74. Shared workspace daemon

Date: 2026-08-02

## Status

Accepted

## Context

CLI and LSP can duplicate expensive analysis for the same workspace.

## Decision

Provide an optional per-workspace daemon with a versioned local protocol over a
`0600` Unix-domain socket (never a network port), with discovery, health, and
lifecycle. This build ships the protocol and lifecycle; sharing the analysis
cache through it is a scheduled follow-up.

## Consequences

A single home for shared incremental state exists, with secure local-only access.
Until cache sharing lands, CLI and LSP analyze in-process.
