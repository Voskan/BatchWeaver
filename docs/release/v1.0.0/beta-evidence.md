# Beta Evidence for the Stable-Release Decision

## Evidence window

- Audit date: 2026-08-02
- Public repository: `https://github.com/Voskan/BatchWeaver`
- Default branch: `main`
- Candidate branch: `release/v0.1.0-beta.1`
- Candidate pull request: #7
- Prerelease sequence: `v0.1.0-beta.1`, `v0.1.0-beta.2`, `v0.1.0-beta.3`

## Public state verified

The repository is public and its owner, remote URL, and default branch agree.
Protected-main checks, private vulnerability reporting, Dependency Graph,
Dependabot security updates, Discussions, labels, Pages, and release authority
were authenticated and configured. The immutable beta tag, prerelease assets,
documentation site, and Go proxy path are verified after publication.

Public verification of beta.1 found that a CLI built through `go install`
reported `dev`; release archives were unaffected because they used injected
metadata. The immutable beta.1 tag was preserved. Beta.2 derives the version
from Go module build information and adds regression coverage. Public download
verification then found that GitHub flattened beta.2 report assets while its
checksum file retained directory prefixes. Beta.3 uses and enforces a flat,
unique public asset layout; earlier tags and releases remain immutable.

## CI evidence

During release-candidate verification:

- CodeQL run `30739695827` passed.
- Linux and macOS builds, validation, coverage, VS Code checks, and release
  assurance passed within CI run `30739695835`.
- The earlier Windows job failed on two LF-versus-CRLF baseline comparisons.
  The root cause was corrected by enforcing LF for `.txt` files. PR #7 CI run
  `30741225928` then passed Linux, macOS, and Windows build/test jobs.
- Dependency Review initially failed because Dependency Graph was disabled.
  After the owner enabled Dependency Graph, run `30741346250` passed at commit
  `5243b07ae939e9bf8d84102f8d04e6a8a1bc2eb7`.
- A real VS Code Extension Host smoke exposed duplicate command registration;
  the extension now delegates server-advertised commands to the language client
  and regression tests prevent reintroduction.

## Feedback classification

| Source | Classification | Result |
| --- | --- | --- |
| Public issues | verified-report | No user reports exist |
| PR #7 Windows log | verified-reproduction | Cross-platform text checkout defect |
| PR #7 dependency log | verified-report | Dependency Graph disabled |
| Public releases/assets | verified-report | First beta and declared assets published and post-verified |
| Downstream integrations | needs-more-information | None publicly evidenced |
| Private security reports | needs-more-information | Not accessible in this session |
| Discussions | verified-report | Repository Discussions enabled; no adoption inference made |

No identity, adoption, integration, or benchmark claim is inferred from stars,
forks, download counts, or the absence of reports.

## Exit-criteria implication

The evidence window begins with this gated beta and its public installation
verification. It is not yet long or broad enough to support stable v1. Stable
release remains blocked.
