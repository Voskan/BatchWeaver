# SLO guardrails

SLO guardrails are the limits an adaptive change must respect. They are
authoritative over the controller's objective.

Before an active change, a modeled added-latency budget must be satisfied. After
an active change, a rollback monitor watches measured metrics for the rollback
window and restores the prior settings if a guardrail is breached: p95 queue
delay, timeout rate, error-rate regression (relative to the pre-change baseline),
backend throttling, fairness regression, verification mismatch, or contract
violation. A rollback records `BW8006` and the prior policy.

Guardrails may be hard (block or roll back a change) or soft (only lower
confidence).
