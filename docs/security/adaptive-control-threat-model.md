# Adaptive control threat model

The adaptive controller can change runtime scheduling. This model covers the
risks of that capability and how they are contained.

## Threats and mitigations

- **Unbounded change** — hard bounds are authoritative and clamp every
  recommendation; the controller can never exceed a configured limit (`BW8004`).
- **SLO regression** — active changes must satisfy a modeled added-latency budget
  and are watched by a rollback monitor that restores prior settings on a
  measured breach within the rollback window (`BW8005`, `BW8006`).
- **Adversarial workload manipulation** — oscillating arrivals, fake duplicates,
  deadline abuse, tenant floods, huge payloads, and throttle storms are bounded
  by hard limits, phase-change detection (which lowers confidence and biases
  toward conservative settings), and exploration step limits with cooldowns.
- **Tuning poisoning via profiles** — profiles are compatibility-checked and
  age-checked; a stale or incompatible profile cannot drive active tuning
  (`BW8001`, `BW8002`), and low confidence downgrades a change to advisory
  (`BW8003`).
- **Denial of service via overload** — overload detection, admission control,
  and load shedding are explicit and observable; requests are never shed silently.
- **Loss of operator control** — an emergency freeze disables the controller and
  restores configured defaults immediately, independent of the profile store.
- **Report exposure** — reports and traces carry only anonymized classes, never
  raw tenants or keys.

## Non-goals

The controller does not train models through external services, does not perform
opaque reinforcement learning, and does not act across processes or datacenters.
