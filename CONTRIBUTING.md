# Contributing to BatchWeaver

Thank you for your interest in contributing. BatchWeaver is an early-stage
compiler and runtime project, and contributions of all sizes are welcome.

## Prerequisites

- Go 1.26.5. The release policy does not claim compatibility with other
  toolchains until they are tested; `GOTOOLCHAIN=auto` can fetch the pin.
- `make` (optional but recommended; it drives the local quality gates).

## Clone, build, and test

```bash
git clone https://github.com/Voskan/BatchWeaver.git
cd BatchWeaver

make build   # build bin/batchweaver
make test    # run unit tests
make check   # run all mandatory local gates
```

## Formatting

- All Go code must be formatted with `gofmt`. Run `make fmt` before committing,
  or `make fmt-check` to verify.
- Follow the settings in `.editorconfig` for indentation and whitespace.

## Tests

- New non-trivial code must be covered by tests.
- Tests must be deterministic and must not depend on network access, the user's
  home directory, local GitHub authentication, timezone, or locale.
- Use `t.TempDir()` for filesystem work and avoid sleeps for synchronization.
- Tests must pass under the race detector: `make test-race`.
- Public API changes must update the reviewed baseline intentionally; schema and
  semantic-safety changes require compatibility and differential tests.

## Documentation

- Every exported Go identifier must have a documentation comment beginning with
  the identifier name.
- Update relevant documents under `docs/` when behavior or structure changes.
- Keep documentation accurate: do not describe functionality that is not
  implemented as if it exists.

## Commit messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/).
Examples:

```text
feat(runtime): add batch scope lifecycle
fix(cli): return usage exit code for unknown commands
docs: clarify project root discovery
```

## Architectural changes

- For large or cross-cutting changes, open an issue to discuss the approach
  before writing significant code.
- Major decisions must be recorded as an Architecture Decision Record under
  [docs/adr/](docs/adr/). See the existing ADRs for the template.

## Compatibility

Before the 1.0 release the public API may change. Breaking changes must be
documented in [CHANGELOG.md](CHANGELOG.md), and changes to compiler behavior or
generated-code format require a changelog entry.

## Local agent and editor files

Do not commit local tooling, editor, or coding-agent state. The repository's
`.gitignore` excludes common cases; keep it that way and never add such files.

## Code review principles

- Prefer clarity over cleverness.
- Keep the dependency direction explicit and acyclic.
- Avoid premature abstraction and speculative interfaces.
- Keep the public API surface small and deliberate.

## Security issues

Do not report vulnerabilities through public issues or pull requests. Follow the
process in [SECURITY.md](SECURITY.md).
