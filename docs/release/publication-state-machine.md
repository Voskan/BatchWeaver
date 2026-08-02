# Prerelease Publication State Machine

States advance monotonically:

```text
planned → gates-passed → release-commit-created → tag-created → tag-pushed
→ repository-public → draft-release-created → assets-uploading → assets-complete
→ prerelease-published → docs-deployed → public-verified
```

Any state may move to `blocked`; a published version may move to `rolled-back`
without deleting its immutable evidence. The ignored `.release/state.json` and
audit log record exact commit, tag, commands, API summaries, digests, and blockers
without credentials.

Recovery inspects the remote tag and release before mutation, verifies every
existing asset digest, resumes only missing steps, and refuses to overwrite a
different artifact. A network failure never triggers a new version, repository,
or duplicate release automatically.
