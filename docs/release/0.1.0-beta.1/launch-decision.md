# Launch Decision: v0.1.0-beta.1

Decision: **authorized for public beta publication**.

The first public version is `v0.1.0-beta.1`. Beta is more truthful
than RC because the public API may still change and the adoption and broader
compatibility evidence period has only begun.

The source repository is publicly readable at
`https://github.com/Voskan/BatchWeaver`. Authenticated admin identity,
branch protection, required checks, Actions policy, Pages, private vulnerability
reporting, community settings, Dependency Review, and a real VS Code host smoke
are verified. The machine-readable gate decision is
`release/gates-v0.1.0-beta.1.json`.

Supported artifacts remain Linux amd64/arm64, macOS amd64/arm64, and Windows
amd64. Go 1.26.5 is the only supported toolchain. Concrete pgx, go-redis,
gqlgen, and grpc-go bindings are absent. Rollback uses scalar fallback,
transformation revert, versioned cache cleanup, and a new immutable hotfix tag.
