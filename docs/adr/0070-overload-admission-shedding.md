# 70. Overload detection, admission control, and load shedding

Date: 2026-08-01

## Status

Accepted

## Context

Under overload, unbounded queueing degrades every caller. The scheduler needs an
explicit, observable response.

## Decision

An overload detector combines queue depth, queue age, memory, backend latency,
timeout, throttling, CPU, goroutine, and pool-saturation signals into a normal,
elevated, or critical state using configured watermarks. Admission control
applies an explicit policy (accept, block, reject, fallback-direct,
shed-low-priority, flush-early). Requests are never shed silently: a shed or
rejected request carries a typed diagnostic, and critical requests are protected.

## Consequences

Overload behavior is explicit, observable, and configurable. Backpressure is
exposed to callers and adapters through a minimal API.
