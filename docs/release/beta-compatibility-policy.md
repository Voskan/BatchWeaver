# Beta Compatibility and Deprecation Policy

Before v1.0, exported Go APIs, configuration and artifact schemas, runtime ABI,
generated code, CLI output, and editor protocol behavior may change between
prereleases. Changes must be intentional, recorded in the changelog, reflected
in schema/ABI identifiers, and accompanied by migration or regeneration steps.

Tags and released assets are immutable. The latest non-withdrawn beta is the
supported prerelease; compatibility with earlier betas is best effort. Tool,
runtime, generated bridge, proof, and editor versions should match. Unsupported
schema versions fail explicitly rather than being guessed.

Where practical, a deprecation appears in one prerelease before removal. Safety,
security, corruption, or isolation defects may require immediate disablement or
removal with an advisory and hotfix. Stable v1.0 compatibility commitments are
not implied by this beta policy.
