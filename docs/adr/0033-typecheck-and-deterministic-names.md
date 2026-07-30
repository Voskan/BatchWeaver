# ADR 0033: No transformation without type-check validation; deterministic names

- Status: Accepted
- Date: 2026-07-29

## Context

Generated code that does not compile, or whose names collide with existing
identifiers, is a correctness defect. Plans must be trustworthy before they are
exposed as buildable.

## Decision

- Every plan parses and type-checks all affected packages through an overlay
  before it is reported as buildable; a type error marks the plan failed with a
  `BW3402` diagnostic and no plan digest is exposed as usable.
- Generated identifiers (`bwKeys`, `bwValues`, `bwErr`, `bwIndex`) are allocated
  deterministically, scanning the enclosing function for collisions and appending
  numeric suffixes; names never change because unrelated files were added.
- Source maps map generated code back to candidate, certificate, transformation,
  and role, using workspace-relative paths.

## Consequences

A plan that validates is compilable and stable; generated names are reproducible
and collision-free.
