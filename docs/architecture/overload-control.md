# Overload control

The overload detector protects the scheduler and its callers under load.

## Signals and states

Queue depth, queue age, memory, backend latency, timeout rate, throttle rate,
CPU, goroutine count, pool saturation, and rejection rate are combined against
configured watermarks into three states: `normal`, `elevated`, and `critical`.
No single signal forces a state.

## Admission control

Under elevated or critical load an explicit admission policy applies: `accept`,
`block`, `reject`, `fallback-direct`, `shed-low-priority`, or `flush-early`.
Critical requests (health checks, system traffic) are protected from shedding.

## Load shedding

Requests are never shed silently. A shed or rejected request carries a typed
diagnostic (`BW8302`, `BW8303`) and, where safe, is routed to the scalar fallback
rather than dropped.

## Backpressure

A minimal backpressure signal (queue full, overloaded, estimated wait,
retry-after, direct-mode recommendation) is exposed to callers and adapters so
they can slow down or route around an overloaded scheduler.
