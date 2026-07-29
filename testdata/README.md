# Test data

This directory holds shared test fixtures. The Go toolchain treats any directory
named `testdata` as data rather than importable code, so fixtures placed here are
ignored by builds and package listing.

Guidelines:

- Keep fixtures minimal and focused on the behavior under test.
- Never include secrets, credentials, or personal data.
- Prefer generating temporary files in tests with `t.TempDir()` when a fixture
  does not need to be version-controlled.
