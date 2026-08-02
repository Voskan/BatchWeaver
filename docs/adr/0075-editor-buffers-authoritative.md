# 75. Editor buffers are authoritative snapshots

Date: 2026-08-02

## Status

Accepted

## Context

Developers act on unsaved code; analyzing on-disk content would be misleading.

## Decision

While a document is open, its editor buffer is authoritative. BatchWeaver builds a
`go/packages` overlay from unsaved bytes and feeds the same overlay to analysis,
proof, preview, and type checking. Nothing is written to disk without an explicit
save or apply.

## Consequences

Results reflect exactly what the developer sees. A single snapshot backs each
request for internal consistency.
