# Release Readiness Report

Release decision: **ready for public `v0.1.0-beta.3` publication**. Source
identity is the clean tagged commit and is recorded exactly in the generated
release manifest.

| Gate | Outcome | Evidence |
| --- | --- | --- |
| Working tree and generated files | PASS | Clean-checkout dry-run precondition; generated Mach-O files removed |
| Public API and schemas | PASS | Exported API baseline and strict manifest compatibility tests |
| Semantic differential | PASS | 256 deterministic seeds, fault cases, and short soak |
| Selected safety mutations | PASS | 12/12 modeled critical mutations killed |
| Fuzz smoke | PASS | Manifest seed corpus; longer fuzzing is scheduled |
| Security | PASS | Local secret/workflow audit, vet, lint, govulncheck and hosted CodeQL pass; private reporting, push protection, and protected-main settings verified |
| Dependencies and licenses | PASS | Go inventory, npm lock/audit, notices, SBOM metadata, Dependency Graph, Dependabot security updates, and hosted Dependency Review pass |
| Compatibility | PASS with limits | Exact statuses in `release/compatibility.json`; untested is never promoted |
| Performance budgets | PASS | 20 allocation samples and 25 bounded scale samples |
| Documentation | PASS | Markdown, YAML, internal links, examples through the Go suite |
| Artifacts and installation smoke | PASS | Digest, layout, native version/help, SBOM, provenance, VSIX install, and real Extension Host activation checks |
| Reproducibility | PASS in declared scope | Two isolated outputs compare byte-for-byte under fixed Go/Node toolchains |

Recommendation: publish the exact `v0.1.0-beta.3` commit as a prerelease under
the declared toolchains. Hosted Linux, macOS, Windows, CodeQL, and Dependency
Review checks pass. Repository identity, admin authority, protected-main checks,
private security reporting, Pages, community settings, and a real VS Code
Extension Host smoke are verified. Public URL and clean external installation
checks remain mandatory post-publication verification steps.
