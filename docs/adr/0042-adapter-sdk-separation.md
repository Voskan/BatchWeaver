# ADR 0042: Adapter SDK separates compile-time and runtime

- Status: Accepted
- Date: 2026-07-29

## Context

Backend adapters have two very different concerns: compile-time discovery/binding/generation and runtime batch execution.

## Decision

- The adapter SDK splits into a compile-time side (manifests, SQL parsing/synthesis, binding, diagnostics) and a runtime side (typed providers, chunking, mapping, verification).
- Compile-time code never opens a backend connection; runtime code never imports go/ast or go/types.
- Adapters connect to execution only through the Prompt 03 typed runtime contracts.

## Consequences

Adapters are testable in isolation and cannot accidentally couple analysis to live backends.
