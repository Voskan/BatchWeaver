# Adopter report template

Copy this file, fill it in, and remove any line you do not want to answer. Every
field is designed to be answerable **without** disclosing source code, payloads,
credentials, tenant identifiers, or raw operation keys.

---

## Identity and consent

- Project or organization (or `anonymous adopter`):
- Contact (GitHub handle, optional):
- May we publish this report in the project's evidence set? `yes` / `no`
- May we name the project? `yes` / `no`

## Environment

| Field | Value |
| --- | --- |
| BatchWeaver version | |
| Go version | |
| OS and architecture | |
| Build mode (module, vendor, go.work) | |
| Database / cache / API clients and versions | |
| Repository size (approximate package count) | |

## Installation

- Install method (`go install`, release archive, source):
- Did `batchweaver version` report the expected version? `yes` / `no`
- Did `batchweaver doctor` report any problem?

## Results

| Step | Outcome | Notes |
| --- | --- | --- |
| `config validate` | pass / fail / not used | |
| `scan` | pass / fail | candidates found: |
| `prove` | pass / fail | proven / rejected counts: |
| `transform plan` | pass / fail | |
| `transform diff` | reviewed / not reviewed | was the diff understandable? |
| `batchweaver test` | pass / fail | did your suite behave identically? |
| materialization | not attempted / attempted | |
| rollback | not attempted / verified | |

**The most important question:** did your test suite produce the same results
through BatchWeaver as without it? If not, that is a correctness defect and we
want it above everything else in this report.

## Performance observation (optional)

Only report what you measured. Estimates are not useful.

| Field | Value |
| --- | --- |
| Workload described in one sentence | |
| Backend calls before / after | |
| Latency p50 / p95 before / after | |
| Measurement method | |
| Hardware and dataset scale | |

## Defects found

For each defect:

- What you expected:
- What happened:
- Diagnostic code, if any:
- **Sanitized** reproduction (a minimal synthetic example, never your real code):
- Is the behavior deterministic?

## Documentation feedback

- Which document did you need and could not find?
- Which explanation was wrong or misleading?
- Where did you get stuck?

## Redaction checklist

Confirm before sending:

- [ ] no source code from a private repository
- [ ] no request or response payloads
- [ ] no credentials, tokens, or connection strings
- [ ] no tenant identifiers or customer names
- [ ] no raw operation keys
- [ ] no internal hostnames or infrastructure detail
