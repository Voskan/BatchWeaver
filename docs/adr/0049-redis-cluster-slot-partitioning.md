# ADR 0049: Redis cluster slot partitioning

- Status: Accepted
- Date: 2026-07-29

## Context

Multi-key Redis commands must not cross cluster hash slots.

## Decision

- The adapter computes Redis cluster slots with CRC-16/XMODEM and hash-tag handling, and groups keys by slot so each multi-key command stays within one slot.
- Cross-slot batches are split by slot; the client remains responsible for routing and MOVED/ASK handling.
- The slot algorithm is pure and verified against known vectors.

## Consequences

Cluster-safe grouping without reimplementing the cluster protocol.
