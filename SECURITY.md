# Security Policy

## Supported versions

BatchWeaver has not had a stable release yet. Until a `1.0` release is
published, only the latest state of the `main` branch is supported for security
fixes.

| Version | Supported |
| ------- | --------- |
| `main` (pre-release) | ✅ |
| Any tagged release | ❌ (none exist yet) |

## Reporting a vulnerability

Please report suspected vulnerabilities **privately** through GitHub Security
Advisories:

1. Go to the repository's **Security** tab.
2. Choose **Report a vulnerability** to open a private advisory.

Do not disclose vulnerabilities publicly — including in issues, pull requests,
or discussions — before they have been coordinated and addressed.

## What to include

A useful report typically contains:

- a description of the vulnerability and its impact;
- the affected component (for example the CLI, a specific package, or generated
  code);
- steps to reproduce, ideally a minimal example;
- the BatchWeaver commit, Go version, and operating system;
- any relevant diagnostic output, with secrets removed.

## Scope

Because BatchWeaver is a compiler and runtime, security concerns include, among
others:

- correctness of transformations that could change program behavior in a way
  that affects security;
- handling of untrusted input during analysis and build;
- safety of generated code;
- the runtime's handling of concurrency, context cancellation, and resource use.

## Secret handling

Never include secrets — tokens, keys, credentials, or private data — in reports,
issues, logs, or test fixtures. Redact them before sharing. The project does not
commit secrets and expects the same of contributions.
