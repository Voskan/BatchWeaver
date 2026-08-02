# Publication Blockers for v0.1.0-beta.1

No tag, GitHub prerelease, Pages deployment, or public asset upload may occur
until all items below are resolved:

1. Authenticate GitHub CLI and verify `gh api user` is the expected release owner.
2. Verify admin/release permissions for exactly `Voskan/BatchWeaver`.
3. Verify `main` protection, required checks, fork permissions, environments,
   Pages, Actions policy, secret scope, and private vulnerability reporting.
4. Fix the Windows-specific test failure observed in hosted CI run 30737635451,
   then observe every required check green at the exact release commit.
5. Install and activate the VSIX in a disposable real VS Code Extension Host.
6. Rebuild from the exact release commit and reproduce/verify all assets.
7. Review whether checksums-only publication is accepted or configure an
   authenticated signing/attestation identity.
8. Create/verify community labels and decide whether to enable Discussions.

The local history-aware Gitleaks v8.30.1 scan covered 22 commits and found no
leaks; repeat it at the final release commit.

After authentication, run `scripts/verify-github-release-gates.sh`. Only an
authorized maintainer may then create the release commit and immutable tag.
