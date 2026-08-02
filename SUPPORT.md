# Beta Support

BatchWeaver is prerelease software maintained on a best-effort basis. There is
no guaranteed SLA. Maintainers aim to elevate transformation-safety and P0
reports promptly and triage ordinary beta reports during regular maintenance.

Use structured GitHub Issues for reproducible bugs, compatibility reports,
performance regressions, documentation defects, and beta feedback. Use
Discussions only if repository governance later enables it. Open-ended support
questions are not guaranteed a response.

Run `batchweaver doctor` or create an allowlist-only bundle with:

```bash
batchweaver doctor --bundle doctor.tar.gz
```

Inspect every bundle before sharing. It excludes source, configuration and
environment values, credentials, request data, SQL, GraphQL variables, headers,
metadata, tenant identifiers, logs, usernames, and local paths by design.

For suspected vulnerabilities, tenant/authorization leakage, or an exploitable
transformation correctness issue, do not open a public issue. Follow
[SECURITY.md](SECURITY.md).

Only the latest non-withdrawn beta is actively supported. Older prereleases may
receive migration guidance but not fixes. Never include credentials or
production data in a public report.
