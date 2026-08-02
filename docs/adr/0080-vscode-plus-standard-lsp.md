# 80. VS Code extension plus standard LSP portability

Date: 2026-08-02

## Status

Accepted

## Context

A first-class VS Code experience is valuable, but BatchWeaver must not be
VS-Code-only.

## Decision

Ship a VS Code extension (source) for the richest experience, and support all
other editors through standard LSP plus documented configurations, labeled
"community" where not officially maintained. Do not build editor-specific
integrations beyond VS Code.

## Consequences

Editor portability is preserved; the VS Code extension adds status bar, output,
and virtual-document conveniences on top of the standard protocol.
