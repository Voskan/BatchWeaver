# Adopter guide

This guide is for teams evaluating BatchWeaver on a real codebase and willing to
report what actually happened. Reports are the only way the project can claim
downstream evidence; maintainer fixtures cannot substitute for them.

Participation is opt-in, and you decide what to share. **Never send source code,
request payloads, credentials, tenant identifiers, or raw operation keys.** The
report template below is structured so you never need to.

## What we ask for

1. an evaluation on a real repository, not a toy example;
2. one structured report using the template below;
3. a sanitized reproduction for any defect you find.

You are not asked to run BatchWeaver in production, and you are not asked to
adopt it permanently.

## Evaluation sequence

Nothing here modifies your source. Every step is read-only until you explicitly
materialize.

```bash
# 1. Install the pinned stable release.
go install github.com/Voskan/BatchWeaver/cmd/batchweaver@v1.0.1
batchweaver version
batchweaver doctor

# 2. Validate configuration, if you have one.
batchweaver config validate --file batchweaver.yaml

# 3. Discover candidates. Read-only.
batchweaver scan ./...

# 4. Prove which candidates are actually safe. Read-only.
batchweaver prove ./...

# 5. Review the exact deterministic change. Still read-only.
batchweaver transform plan ./...
batchweaver transform diff ./...

# 6. Run your own test suite through the overlay, without editing source.
batchweaver test -- -race ./...
```

Stop at any step. If step 6 fails, that result is the most valuable thing you
can report.

## Rollback

Materialization is a separate, explicit command with a backup manifest. If you
materialize and want to undo it:

```bash
batchweaver transform revert
```

The supported fallback is always plain scalar execution using your original
source. See [rollback](../release/rollback.md).

## Reporting

Copy [the report template](adopter-report-template.md), fill it in, and attach
it to a
[GitHub Discussion](https://github.com/Voskan/BatchWeaver/discussions) or an
[issue](https://github.com/Voskan/BatchWeaver/issues/new/choose).

For a suspected security problem, use
[private vulnerability reporting](https://github.com/Voskan/BatchWeaver/security/advisories/new)
instead of a public thread.

## What we publish

With your permission we publish: your project name or "anonymous adopter", the
Go/platform/client matrix, and the outcome of each step. We never publish source,
payloads, credentials, tenant identifiers, raw keys, or internal hostnames.

You may withdraw consent at any time and the report is removed from the evidence
set.
