# Tuning diagnostics reference

Adaptive scheduling diagnostics occupy the `BW8xxx` range, kept distinct from
analysis (`BW3xxx`), transform (`BW34xx`-`BW38xx`), runtime (`BW4xxx`), proof
(`BW5xxx`), backend adapter (`BW6xxx`), and network adapter (`BW7xxx`) codes.

The Prompt 10 specification illustrated these diagnostics in the `BW7xxx` range,
which network adapters already own. To preserve the repository rule that
diagnostic ranges are distinct per stage, this implementation renumbers them into
`BW8xxx`; the semantic meaning is unchanged.

| Code | Meaning |
| --- | --- |
| `BW8001` | Workload profile is incompatible. |
| `BW8002` | Workload profile is stale. |
| `BW8003` | Insufficient evidence for active tuning. |
| `BW8004` | Adaptive recommendation exceeds hard bound (clamped). |
| `BW8005` | SLO guardrail breached. |
| `BW8006` | Adaptive policy rolled back. |
| `BW8101` | Operation dependency graph contains an unsupported cycle. |
| `BW8102` | Recursive batching depth limit exceeded. |
| `BW8103` | Recursive traversal proof is stale. |
| `BW8201` | Tenant quota exceeded. |
| `BW8202` | Starvation detected. |
| `BW8301` | Scheduler overload detected. |
| `BW8302` | Request shed by policy. |
| `BW8303` | Admission control rejected request. |
| `BW8401` | Replay input is incomplete. |
| `BW8402` | Cost model confidence is low. |
