# Editor workspace trust

BatchWeaver treats an unopened or untrusted workspace as hostile until the user
grants trust.

## Untrusted workspace

Allowed:

- syntax-level and configuration text diagnostics that require no command
  execution.

Blocked by default:

- starting the BatchWeaver server, gopls, or the workspace daemon;
- running the Go command, tests, benchmarks, or code generators;
- transformation materialization;
- external schema fetches and adapter verification servers.

The VS Code extension enforces this through `capabilities.untrustedWorkspaces`
(limited) and starts the server only after trust is granted.

## Consent

Long-running or mutating actions — deep analysis, proof verification, tests,
benchmarks, profile collection, materialization — require an explicit user
request. Nothing of the sort runs automatically on file open.
