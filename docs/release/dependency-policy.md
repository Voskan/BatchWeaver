# Dependency and Tool Governance

Dependencies must have a clear use, maintained upstream, compatible license,
reviewed API surface, and bounded trust role. Pull requests adding or updating a
dependency include the reason, direct/transitive inventory change, license,
security scan, generated-code impact, and rollback plan. A newer version alone
is not sufficient reason to upgrade.

Go dependencies are pinned by `go.mod`/`go.sum`; editor dependencies and release
packaging tools are exact versions in `package.json` and the npm lockfile. GitHub
Actions use immutable 40-character commit pins with a reviewed version comment.
Dependabot proposes updates, but required tests and human review remain mandatory.

Abandoned or vulnerable dependencies are removed, replaced, temporarily forked
under documented governance, or covered by a time-bounded exception. Exceptions
identify an owner, impact, mitigation, tracking issue, and expiry. Release tools
must not pipe network content into a shell, modify global state, or access
release credentials during a snapshot build.
