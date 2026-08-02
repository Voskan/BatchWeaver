# Package Manager and Container Decision

The beta's supported channels are Go module installation and verified GitHub
archives. Homebrew and Scoop metadata are not published because final public
asset URLs/digests and tap or bucket ownership do not yet exist. Draft metadata
would be misleading until it can pass install, version, checksum, and uninstall
tests against an immutable release.

No container image is planned for the CLI beta. A standalone static binary is
simpler, smaller, and matches the current use case. A future daemon distribution
would require a separate threat, configuration, persistence, and lifecycle
review before any image publication.
