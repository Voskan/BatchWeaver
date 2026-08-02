# 77. Versioned WorkspaceEdit preconditions

Date: 2026-08-02

## Status

Accepted

## Context

Applying an edit computed from a stale snapshot could corrupt source.

## Decision

Applying a transformation requires a current proof, unchanged documents, a
type-checking package, client support for versioned workspace edits, and explicit
user action. Edits carry version/digest/plan/proof preconditions; when the client
cannot enforce versions, the server revalidates immediately and warns.

## Consequences

Stale edits are refused. This build previews transformations and points to the
CLI for the exact diff; the versioned apply flow through the LSP is the next
increment.
