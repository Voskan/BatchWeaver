# Privacy and Diagnostic Data

BatchWeaver performs analysis locally and does not upload source code or private
runtime data by default. Prompt-free product telemetry is not implemented.
Project caches are local and should be treated as developer data. Workload
profiles contain bounded histograms and anonymized counts, not raw keys or
payloads, but still require an organization's own privacy review.

`batchweaver doctor --bundle` is allowlist-only. It excludes source, config and
environment values, tokens, URLs, request payloads, SQL, GraphQL variables,
headers, protocol metadata, tenant identifiers, logs, usernames, and home paths.
Users control whether a bundle is shared and must inspect it first.

Launch health uses public GitHub release/download events and voluntary feedback
only. No private user data is scraped or stored. This is a technical data-flow
statement, not a claim of legal compliance.
