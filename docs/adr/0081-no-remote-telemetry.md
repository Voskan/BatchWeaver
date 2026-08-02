# 81. No remote telemetry

Date: 2026-08-02

## Status

Accepted

## Context

Editor integrations often collect usage telemetry, which is a privacy risk.

## Decision

BatchWeaver's editor layer implements no remote telemetry and uploads no source.
Traces are local and redacted, never including source or secrets by default.

## Consequences

Users' code and usage stay local. Any future telemetry would require an explicit,
separately reviewed opt-in.
