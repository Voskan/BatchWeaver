# Profile privacy threat model

Workload profiles summarize production traffic and must not become a data leak.

## Assets

Operation keys, tenant identities, authorization tokens, request bodies, SQL
parameters, GraphQL variables, gRPC metadata values, HTTP headers and cookies,
raw URLs with identifiers, and source code.

## Guarantees

Profiles store none of the above. They contain only histograms, sketches,
counters, and bounded categorical counts keyed by anonymized, non-reversible
class labels. Partition and tenant identities are reduced to keyed-hash
(HMAC-SHA256) class labels with a per-session random salt in production, so labels
cannot be correlated across sessions or reversed to the raw identifier. Class
cardinality is bounded; excess collapses to a single overflow class. The
`raw_keys_stored` flag exists so audits can assert it is false.

## Threats and mitigations

- **Profile leakage** — profiles are safe to persist within a trust boundary; no
  raw identifier is present. There is no automatic remote upload.
- **Tenant inference** — random per-session salting and bounded cardinality
  prevent correlating class labels back to tenants across collections.
- **Sampled counts misread as exact** — sampling metadata records the strategy,
  rate, and scale factor; sampled counts are always labeled.
- **Corruption / tampering** — persisted profiles are checksummed and their
  embedded digest is recomputed and verified on load.

## Reproducible collection

Reproducible offline collection and tests use a deterministic salt seed so class
labels are stable. Production collection uses a random salt; labels remain
non-reversible either way.
