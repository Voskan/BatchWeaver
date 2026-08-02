# Security Policy

## Supported versions

Before the first beta is published, only `main` receives security fixes. After
publication, only the latest non-withdrawn prerelease is supported unless an
advisory says otherwise.

## Private reporting

Use **Report a vulnerability** on this repository's Security page when enabled.
If private reporting is unavailable, use a private contact channel published on
the repository owner's verified GitHub profile. Do not send secrets until the
recipient and private or encrypted channel are verified.

Do not open a public issue for a vulnerability, partition or authorization leak,
data corruption, proof bypass, or exploitable transformation mismatch. Include
affected versions, impact, a sanitized reproduction, environment details, and a
possible mitigation. A public exploit reproduction is never required.

Never include keys, tokens, credentials, production source, request data, SQL
parameters, GraphQL variables, headers, metadata, or tenant identifiers.

## Handling and disclosure

The maintainer will acknowledge when available, reproduce privately, assign
severity, use an embargoed fix branch, add a regression test, prepare a GHSA/CVE
when appropriate, publish a new immutable prerelease, notify affected users,
then disclose according to risk. No fixed response SLA is promised.

P0 incidents include cross-tenant or authorization execution, semantic data
corruption, proof bypass, credential disclosure, or a release installation that
executes unverified content. Maintainers may mark a release unsafe, disable the
affected strategy, invalidate proofs and caches, recommend scalar fallback and
revert, retain forensic artifacts, and publish a new version. Released tags or
assets are never silently replaced or rewritten.
