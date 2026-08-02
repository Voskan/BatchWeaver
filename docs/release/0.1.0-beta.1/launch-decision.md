# Launch Decision: v0.1.0-beta.1

Decision: **blocked before tag or publication**.

The first public version is selected as `v0.1.0-beta.1`. Beta is more truthful
than RC because the public API may still change and hosted installation,
Extension Host, compatibility, and adoption evidence are incomplete.

The source repository is publicly readable at
`https://github.com/Voskan/BatchWeaver`, but the release operator's authenticated
GitHub identity, administrative permissions, branch protection, Actions policy,
Pages configuration, and private vulnerability reporting could not be verified.
The machine-readable gate decision is `release/gates-v0.1.0-beta.1.json`.

Supported artifacts remain Linux amd64/arm64, macOS amd64/arm64, and Windows
amd64. Go 1.26.5 is the only supported toolchain. Concrete pgx, go-redis,
gqlgen, and grpc-go bindings are absent. Rollback uses scalar fallback,
transformation revert, versioned cache cleanup, and a new immutable hotfix tag.
