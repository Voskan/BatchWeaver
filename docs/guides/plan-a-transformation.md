# Plan a transformation

`transform plan` evaluates proof certificates and produces a deterministic plan
without modifying source.

## Plan

```bash
batchweaver transform plan ./...
```

The summary reports the number of planned transformations, skipped candidates by
reason, files replaced through the overlay, net line changes, validation
outcomes, and the plan ID. A machine-readable plan is available with
`--format=json`.

Filters (`--candidate`, `--operation`, `--file`, `--max-transformations`) select a
subset without changing proof semantics. Candidates excluded by filters are
distinct from candidates rejected by the safety rules.

## Diff

```bash
batchweaver transform diff <plan-id>
```

A deterministic unified diff with `a/` and `b/` paths and no timestamps.

## Inspect

```bash
batchweaver transform inspect <plan-id> [--candidate=<id>]
```

Shows the transformation, candidate, certificate, strategy, source range, phases,
generated symbols, and assumptions.

## Verify

```bash
batchweaver transform verify <plan-id>
```

Re-runs analysis, proof, anchoring, generation, parse, and type check, and
confirms the deterministic plan digest matches the stored plan.

## Clean

```bash
batchweaver transform clean
```

Removes the cached plans under `.batchweaver/`.
