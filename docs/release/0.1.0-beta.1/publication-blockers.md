# Publication Gate Closure for v0.1.0-beta.1

Decision: **ready**. The pre-publication blockers were resolved as follows:

1. Authenticated GitHub identity `Voskan` has admin permission for exactly
   `Voskan/BatchWeaver`.
2. Main protection, required checks, Actions policy, environments, Pages, and
   private vulnerability reporting are configured and verified.
3. The hosted Windows fixture defect is fixed; Linux, macOS, Windows, Validate,
   coverage, VS Code, release assurance, CodeQL, and Dependency Review pass.
4. The VSIX installs and activates in Visual Studio Code 1.131.0 on macOS arm64.
   That smoke found and led to a regression-tested duplicate-command fix.
5. Clean-checkout artifacts reproduce byte-for-byte in the declared scope.
6. Checksums-only beta publication is accepted with unsigned tag/artifact and
   local-provenance limitations stated in the release notes.
7. Community labels and Discussions are configured.

The remaining public-link, Go-proxy, archive-download, and release-VSIX checks
are intentionally post-publication because their endpoints do not exist before
the immutable tag and release. A failed check requires a new hotfix version; it
never authorizes moving or deleting the published tag.

Run `scripts/verify-github-release-gates.sh --pre-tag` before the tag and
`scripts/verify-github-release-gates.sh --publish` before the prerelease upload.
