# ADR 0028: Versioned transformation IR independent of AST/SSA

- Status: Accepted
- Date: 2026-07-29

## Context

The transformation stage must produce deterministic, cacheable, comparable plans
that reference stable identities from earlier stages, not ephemeral compiler
objects.

## Decision

- Define a versioned transformation IR (`batchweaver.transform/v1alpha1`) that
  references Prompt 04 identities and Prompt 05 certificates.
- Never serialize `token.Pos`, AST nodes, SSA values, `types.Object` pointers,
  process IDs, timestamps, or absolute machine paths in deterministic artifacts.
- Use workspace-relative slash paths and content-addressed IDs (plan,
  transformation, edit, materialization) derived from canonical content.

## Consequences

Plans are byte-identical across machines and runs for unchanged inputs, and the
IR is independent of the toolchain's internal representations.
