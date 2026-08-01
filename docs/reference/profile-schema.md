# Profile schema reference

Schema version: `batchweaver.profile/v1alpha1`.

## ProfileBundle

| Field | Meaning |
| --- | --- |
| `schema_version` | Profile schema version. |
| `id` | Short content-addressed bundle ID. |
| `toolchain` | Go version, build version, commit, GOOS, GOARCH. |
| `runtime_abi` | Runtime bridge ABI the profile was collected under. |
| `config_digest` | Digest of the effective configuration. |
| `window` | Observation window (metadata; excluded from the digest). |
| `operations` | Per-operation profiles, sorted by operation ID. |
| `redaction` | Redaction summary; `raw_keys_stored` is always false. |
| `digest` | Content-addressed digest over identity and all distributions. |

## OperationProfile

Holds `arrivals`, `queue`, `batches`, `backend`, `deadlines`, `errors`,
`duplicates`, `payloads`, `partitions`, `fallbacks`, `chunks`, `fairness`,
`execution_modes`, `sampling`, and a per-operation `digest` and
`operation_digest` (used for compatibility checks).

## Histograms

Distributions serialize as `HistogramData`: relative `accuracy`, `count`, `sum`,
`min`, `max`, `zero_count`, an `overflow` flag, and parallel ascending
`bucket_index` / `bucket_counts` arrays. Encoding is canonical and byte-stable.

## Compatibility

A profile is compatible when its schema, runtime ABI, config digest, toolchain,
and per-operation digests match the requirement. Excessive age marks it stale;
stale profiles are usable for offline comparison but not active warm start unless
explicitly permitted.
