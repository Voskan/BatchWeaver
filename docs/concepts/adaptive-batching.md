# Adaptive batching

Adaptive batching is BatchWeaver choosing scheduler settings — `max_wait`,
`max_batch_size`, concurrency, chunk size, and execution mode — from measured
workload evidence rather than fixed configuration alone.

It is bounded and explainable. Every change is derived from a versioned cost
model, clamped to authoritative hard bounds, gated by SLO guardrails, and
recorded as a content-addressed decision with evidence, reasons, and confidence.
Adaptive batching never guarantees a universal performance improvement, never
exceeds a configured bound, and never bypasses a semantic proof.

Execution-mode availability (static prefetch, runtime coalescing, batch-of-one,
direct scalar, native batch, pipeline, fallback) is declared by the operation
contract and adapter, not inferred by the controller.
