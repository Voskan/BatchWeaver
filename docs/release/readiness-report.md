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
| Security | PASS | Local secret/workflow audit, vet, lint, govulncheck; hosted CodeQL remains CI evidence |
| Dependencies and licenses | PASS | Go inventory, npm lock/audit, bundled runtime notices, SBOM license metadata |
| Compatibility | PASS with limits | Exact statuses in `release/compatibility.json`; untested is never promoted |
| Performance budgets | PASS | 20 allocation samples and 25 bounded scale samples |
| Documentation | PASS | Markdown, YAML, internal links, examples through the Go suite |
| Artifacts and installation smoke | PASS | Digest, layout, native version/help, SBOM, provenance, and VSIX checks |
| Reproducibility | PASS in declared scope | Two isolated outputs compare byte-for-byte under fixed Go/Node toolchains |

Recommendation: the source is suitable for an unpublished beta
snapshot under the declared toolchains. Public publication is not ready until an
authorized maintainer verifies hosted branch protection, required CI checks, and
a real VS Code Extension Host smoke; see `KNOWN-ISSUES.md`.
