# Proving candidates

The `prove` command evaluates semantic batching eligibility for every candidate
the analysis stage discovered. It does not modify source.

## Prove a workspace

```bash
batchweaver prove ./...
```

The summary reports candidate counts, the histogram of proof outcomes, and the
per-strategy eligible counts. A reproducible machine report is available with:

```bash
batchweaver prove ./... --format=json --reproducible > batchweaver-proof.json
```

## Inspect a candidate

```bash
batchweaver candidate inspect <candidate-id>
```

For a proven-eligible candidate this lists the allowed strategies, the proof
certificate ID, and the satisfied obligations. For a rejected or unknown
candidate it names the failed or unknown obligation, prints the witness, and
reports the independent runtime-coalescing eligibility.

## Explain one obligation

```bash
batchweaver proof explain <proof-id> --obligation=error-order
```

`--obligation` accepts an obligation ID (`BW-PROOF-ERROR-001`) or a short alias
(`error-order`, `key`, `receiver`, `context`, `target`, `order`).

## Other commands

- `batchweaver proof inspect <proof-id>` — inspect by certificate ID.
- `batchweaver proof graph <proof-id> [--format=dot|json]` — evidence graph.
- `batchweaver assumption list ./...` — assumptions candidates require.
- `batchweaver strategy list` and `batchweaver strategy inspect <id>` — the
  strategy registry and each strategy's required obligations.

## Filters and limits

`prove` accepts `--strategy`, `--decision`, and `--candidate` filters (which list
matching candidates), `--max-candidates` to bound work, and `--fail-on-unproven`
to exit nonzero unless every candidate is proven eligible. Package-loading flags
are inherited from the analysis stage.

## Interpreting outcomes

A candidate can be eligible for one strategy and unknown or ineligible for
another. `unknown` means the analysis was not precise enough to decide, not that
the candidate is unsafe; `proven_ineligible` means a concrete semantic conflict
exists. See [explaining a rejection](explaining-a-rejection.md).
