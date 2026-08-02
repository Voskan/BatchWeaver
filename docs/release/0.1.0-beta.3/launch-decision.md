# Launch Decision: v0.1.0-beta.3

Decision: **authorized supply-chain hotfix beta**.

Beta.3 is required because public verification found that GitHub flattened
beta.2 report asset paths while its checksums retained directory prefixes. The
new version preserves earlier release immutability, changes no public API,
schema, or ABI, and enforces a flat, unique public asset layout with regression
coverage. Stable v1 remains blocked by the stable-release decision.
