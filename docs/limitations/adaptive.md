# Adaptive Scheduling Limitations

## Implemented

Privacy-safe workload profiling; versioned cost model and objective policies; a
bounded, explainable controller (adaptive `max_wait`, `max_batch_size`,
concurrency, chunk size, and execution-mode recommendation) with cold/warm start,
shadow/active modes, canary and exploration limits, phase-change detection, decay,
SLO guardrails, and automatic rollback; a runtime hot-update path that clamps
adaptive settings to hard bounds and applies them atomically; multi-operation
wave planning with a dependency DAG, co-scheduling, provider-fusion hooks, and
critical-path analysis; recursive breadth-first batching for proven traversals;
fairness (weighted fair queueing and deficit round robin) with priority classes,
quotas, reserved capacity, and starvation detection; overload detection with
admission control, load shedding, and backpressure; deterministic offline replay,
policy simulation, counterfactuals, synthetic workload generators, and tuning
reports; `BW8xxx` diagnostics; CLI, configuration, docs, ADRs, and tests.

## Deliberately not implemented (out of scope)

- Model training through external hosted services; opaque reinforcement learning.
- Cross-process or distributed batching; distributed consensus; a global
  multi-datacenter scheduler; a remote SaaS control plane.
- Automatic infrastructure scaling; a Kubernetes operator; a service mesh.
- Arbitrary recursive program transformation without a semantic proof.
- Automatic production activation without explicit configuration.

## Honest notes on this build

- **Profile collection** in the CLI runs against a deterministic synthetic
  workload, because this build is not attached to a live service. In production
  the same collector is fed by the runtime; the collector, schema, persistence,
  privacy, cost model, controller, waves, recursion, fairness, overload, replay,
  and reports are all real and tested.
- **Metrics and tracing** are specified with bounded labels and documented names;
  concrete exporter bindings are added with the observability integration and add
  no third-party dependency here.
- **Diagnostic renumbering** — the specification's illustrative `BW7xxx` codes are
  renumbered into `BW8xxx` to keep diagnostic ranges distinct per stage; the
  mapping is documented in the tuning diagnostics reference.
- **No universal optimization guarantee.** The controller offers bounded,
  explainable, model-estimated improvements; it never claims universal optimality
  or a guaranteed performance gain.
