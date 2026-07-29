# Proof report schema

The proof report is emitted by `batchweaver prove --format json`. Its schema
version is `batchweaver.proof/v1alpha1`, independent from the analysis schema
version. It is alpha: fields may be added compatibly, and readers must reject
unsupported future major versions.

## Top-level fields

| Field | Meaning |
| --- | --- |
| `schema_version` | proof schema identifier |
| `tool_version` | BatchWeaver version |
| `go_version` | Go runtime version that produced the report |
| `timestamp` | omitted in `--reproducible` mode |
| `workspace` | portable workspace root |
| `analysis_schema` | schema of the consumed analysis snapshot |
| `analysis_digest` | digest binding the report to its analysis inputs |
| `contract_digest` | digest of the operation contracts in force |
| `assumption_digest` | digest of the applied assumption set |
| `declared_operations`, `operation_call_sites`, `candidates` | counts |
| `decision_counts` | histogram keyed by decision value |
| `strategy_counts` | histogram of eligible strategies |
| `candidate_proofs` | per-candidate proof records |
| `assumptions` | required assumptions (none applied automatically) |
| `diagnostics` | proof diagnostics |

## Candidate proof

| Field | Meaning |
| --- | --- |
| `id` | candidate ID |
| `proof_id` | deterministic proof certificate ID |
| `operation` | operation ID |
| `structure` | structural context label |
| `location` | portable source location |
| `candidate_digest` | digest of the analyzed candidate facts |
| `decision` | one of the five closed decisions |
| `reason_code` | machine-readable reason for a non-eligible decision |
| `allowed_strategies` | per-strategy eligibility with blocking obligations |
| `obligations` | ordered obligation results with status and evidence |
| `assumptions` | assumption IDs the decision depends on |
| `witnesses` | concrete failure or uncertainty traces |
| `limitations` | explicit non-guarantees |
| `invalidation` | digests whose change invalidates the certificate |

## Ordering and identity

Arrays whose order is not semantically meaningful are sorted by a stable key.
IDs are content-addressed over canonical inputs and never include host paths or
pointer identities. In reproducible mode the report is byte-identical across
machines and checkout paths for unchanged inputs.

## Decisions

`proven_eligible`, `proven_ineligible`, `requires_assumption`, `unknown`, and
`deferred`. A decision is always derived from named obligations; it is never a
score.
