# Adaptive scheduler

The adaptive scheduler is BatchWeaver's production optimization and control
layer. It lives in `internal/adaptive` and is deliberately conservative: it can
recommend scheduler settings, or apply settings that stay within caller-configured
hard bounds, but it can never bypass a semantic proof, combine partitions, change
transaction identity, exceed a backend limit, or override an operator's emergency
disablement.

## Components

- **Workload profiler** — collects privacy-safe, bounded statistics into a
  versioned [profile bundle](workload-profiles.md).
- **Cost model** — a versioned, explicitly weighted [objective](cost-model.md)
  over profile evidence.
- **Controller** — searches bounded settings for the minimum-cost policy, clamps
  to hard bounds, gates on confidence and SLO guardrails, and records an
  explainable decision.
- **Rollback monitor** — restores prior settings if measured metrics breach a
  guardrail within the rollback window.
- **Wave planner** and **recursive batcher** — coordinate work across operations
  and across breadth-first frontiers.
- **Fairness scheduler** and **overload detector** — protect tenants and the
  system under contention and load.

## Core safety rule

Adaptive behavior may change runtime settings only when the operation contract,
semantic proof, adapter and runtime ABIs, and profile are valid and current; hard
bounds and SLO guardrails permit the change; the exploration budget and
tenant/partition policy permit it; and no error-severity diagnostic remains.

## Runtime integration

The controller never mutates the runtime directly. An accepted decision's settings
are applied through `runtime.BoundOperation.ApplyAdaptiveSettings`, which clamps
every field to the binding's configured limits and installs them atomically:
in-flight batches keep the settings they were dispatched under, and the next
scheduling decision observes the new snapshot. `ClearAdaptiveSettings` restores
the configured defaults and is the runtime side of an emergency freeze.
