# 79. No automatic mutating actions

Date: 2026-08-02

## Status

Accepted

## Context

An editor integration that silently changes source or runs code is dangerous.

## Decision

The server never writes source, runs project code, tests, benchmarks,
generators, or materialization implicitly. Every mutating or long-running action
requires an explicit user request, and an untrusted workspace blocks child
processes entirely.

## Consequences

The integration is safe by default. Developers stay in control of every change.
