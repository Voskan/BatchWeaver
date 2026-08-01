# Fairness and quotas

The fair scheduler shares scheduling opportunities across anonymized classes
(operations and tenant classes) without starving any of them.

## Algorithms

Weighted fair queueing and deficit round robin are both supported. Classes carry
a weight, a priority, an optional reserved share, and a quota. A class below its
reserved share with backlog is preferred; otherwise the configured discipline
selects the next class.

## Quotas

Per-class quotas bound queued items, active items, concurrency, and payload
bytes. An admission that would exceed a quota is rejected with `BW8201`.

## Starvation and priority

Head-of-line wait beyond the starvation threshold is detected (`BW8202`) and can
drive priority aging. Reserved capacity protects high-priority and system traffic
without completely starving normal traffic.

## Identity

Fairness classes are anonymized (keyed-hash class labels). The runtime still
partitions by exact tenant internally; only class labels appear in metrics and
reports, so tenant identities are never exposed.
