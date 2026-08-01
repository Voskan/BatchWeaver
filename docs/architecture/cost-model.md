# Cost model

The cost model (`batchweaver.cost/v1alpha1`) turns profile evidence and a
candidate policy into a modeled total cost in nanoseconds of equivalent effective
latency. It is an estimate, never a measurement.

## Objective

```text
total_cost =
    backend_fixed_cost / batch_size
  + per_item_backend_cost
  + queue_delay_penalty
  + deadline_risk_penalty
  + serialization_cost / batch_size
  + mapping_cost
  + chunking_cost
  + retry_cost
  + error_penalty
  + fairness_penalty
  + overload_penalty
```

The fixed backend cost is amortized across the batch, which is why batching
reduces cost up to the point where added queue delay and deadline risk dominate.

## Objective policies

Every policy maps to explicit, documented weights: `latency`, `throughput`,
`balanced`, `backend-cost`, `deadline-protection`, and `custom-weighted`. There
is no single universal objective and no hidden magic constants.

## Backend cost estimation

Fixed and per-item backend costs are estimated from the backend latency and
batch-size histograms: the low-percentile latency approximates the fixed
overhead, and the residual mean latency spread over the mean batch size
approximates the marginal per-item cost.

## Confidence

Confidence is a bounded `[0,1]` score from sample count, profile age (exponential
decay), histogram overflow, missing metrics, and stationarity. Low confidence
downgrades a recommendation to advisory (shadow only).
