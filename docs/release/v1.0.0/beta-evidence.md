# Beta Evidence for the Stable-Release Decision

## Evidence window

- Audit date: 2026-08-02
- Public repository: `https://github.com/Voskan/BatchWeaver`
- Default branch: `main`
- Candidate branch: `release/v0.1.0-beta.1`
- Candidate pull request: #7
- Selected candidate: `v0.1.0-beta.1`

## Public state verified

The repository is public and its owner, remote URL, and default branch agree.
The public GitHub API returned no tags and no releases. GitHub Pages returned no
configured public site. The repository has Issues enabled and Discussions
disabled. The only open issue-shaped item was PR #7; there were no public bug,
compatibility, performance, editor, adapter, installation, or documentation
reports.

The Go module proxy returned an empty version list. No pkg.go.dev version can be
treated as published until an immutable semantic-version tag is fetched by the
proxy.

## CI evidence

At candidate commit `eee33b88761af4ac309298eae035d015547287f1`:

- CodeQL run `30739695827` passed.
- Linux and macOS builds, validation, coverage, VS Code checks, and release
  assurance passed within CI run `30739695835`.
- The Windows job failed on two LF-versus-CRLF baseline comparisons. The root
  cause is reproduced and corrected by enforcing LF for `.txt` files; hosted
  verification is pending.
- Dependency Review run `30739695826` failed because the repository Dependency
  Graph is disabled. This is a repository setting, not a lockfile finding.

## Feedback classification

| Source | Classification | Result |
| --- | --- | --- |
| Public issues | verified-report | No user reports exist |
| PR #7 Windows log | verified-reproduction | Cross-platform text checkout defect |
| PR #7 dependency log | verified-report | Dependency Graph disabled |
| Public releases/assets | needs-more-information | No release exists |
| Downstream integrations | needs-more-information | None publicly evidenced |
| Private security reports | needs-more-information | Not accessible in this session |
| Discussions | unsupported-use-case | Repository Discussions disabled |

No identity, adoption, integration, or benchmark claim is inferred from stars,
forks, download counts, or the absence of reports.

## Exit-criteria implication

The evidence window has not begun in a form capable of supporting stable v1.
The first defensible next step is a gated beta publication followed by public
installation and compatibility verification. Stable release remains blocked.
