# Workload profiles

A profile bundle is a versioned, deterministic, privacy-safe summary of an
operation's workload. Its schema version is `batchweaver.profile/v1alpha1`.

## Structure

A `ProfileBundle` carries toolchain identity, runtime ABI, config digest, an
observation window, a redaction summary, and one `OperationProfile` per
operation. Each operation profile holds arrival, queue, batch, backend,
deadline, error, duplicate, payload, partition, fallback, chunk, and fairness
sub-profiles, plus sampling metadata.

## Determinism

The bundle digest is content-addressed over identity fields and every
distribution and counter, but excludes wall-clock metadata (created time,
window), so two structurally identical bundles compare equal. Persisted bundles
are wrapped in a checksummed envelope; a corrupt or tampered file is rejected on
load, and the embedded digest is recomputed and verified.

## Distributions

Distributions are stored as bounded, mergeable, logarithmic-bucket histograms
(a simplified DDSketch) with a configurable relative accuracy. Bucket count is
bounded even under adversarial inputs; overflow is flagged so a clamp is never
mistaken for exact data.

## Privacy

Profiles never store raw keys, tokens, tenants, bodies, parameters, variables,
metadata values, headers, URLs, or source. See [profile privacy](../security/profile-privacy.md).
