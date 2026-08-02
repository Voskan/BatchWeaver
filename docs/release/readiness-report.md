# Release Readiness Report

Local snapshot recommendation: `0.1.0-beta.1`. Source identity is the clean commit that
contains this report and is recorded exactly in the generated release manifest.
No tag or publication has occurred.

| Gate | Outcome | Evidence |
| --- | --- | --- |
| Working tree and generated files | PASS | Clean-checkout dry-run precondition; generated Mach-O files removed |
| Public API and schemas | PASS | Exported API baseline and strict manifest compatibility tests |
| Semantic differential | PASS | 256 deterministic seeds, fault cases, and short soak |
| Selected safety mutations | PASS | 12/12 modeled critical mutations killed |
| Fuzz smoke | PASS | Manifest seed corpus; longer fuzzing is scheduled |
| Security | BLOCKED for publication | Local secret/workflow audit, vet, lint, govulncheck and hosted CodeQL pass; private reporting and hosted settings remain unverified |
| Dependencies and licenses | BLOCKED for publication | Go inventory, npm lock/audit, notices, and SBOM metadata pass; Dependency Review cannot run until Dependency Graph is enabled |
| Compatibility | PASS with limits | Exact statuses in `release/compatibility.json`; untested is never promoted |
| Performance budgets | PASS | 20 allocation samples and 25 bounded scale samples |
| Documentation | PASS | Markdown, YAML, internal links, examples through the Go suite |
| Artifacts and installation smoke | PASS | Digest, layout, native version/help, SBOM, provenance, and VSIX checks |
| Reproducibility | PASS in declared scope | Two isolated outputs compare byte-for-byte under fixed Go/Node toolchains |

Recommendation: the source remains suitable for an unpublished beta snapshot
under the declared toolchains. Hosted Linux, macOS, and Windows builds now pass.
Public publication is not ready until Dependency Graph enables the mandatory
dependency review and an authorized maintainer verifies branch protection,
security reporting, required checks, publishing identity, and a real VS Code
Extension Host smoke; see `KNOWN-ISSUES.md`.
