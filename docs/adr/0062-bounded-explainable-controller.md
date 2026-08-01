# 62. Bounded, explainable adaptive controller

Date: 2026-08-01

## Status

Accepted

## Context

Automatic scheduler tuning can improve batching efficiency, but an opaque
optimizer that changes production behavior without guardrails, rollback, or
explanation is unacceptable in a correctness-first system.

## Decision

The adaptive controller is a bounded, explainable, cost-model-driven component.
Every recommendation is derived from a versioned cost model over a workload
profile, is clamped to authoritative hard bounds, and records evidence, reasons,
confidence, and safety state in a content-addressed decision. There is no
reinforcement learning and no hidden state that changes behavior without a
recorded decision.

## Consequences

Recommendations are reproducible and auditable. The controller cannot exceed a
configured bound or apply a change that fails a safety gate. The cost of this
conservatism is that the controller does not chase every theoretical optimum; it
prefers a bounded, defensible improvement.
