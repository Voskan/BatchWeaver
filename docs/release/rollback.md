# Release Rollback Plan

Stop further publication, preserve evidence, and mark the affected release and
artifacts as compromised or withdrawn. Do not silently replace an immutable tag
or checksum. Publish a corrected release with a new version, corrected checksums,
updated provenance, and a compatibility/security notice. Deprecate or unlist an
extension/package only through its channel's reviewed process.

For a security incident, use the private advisory path in `SECURITY.md`, assess
credential rotation and history exposure, coordinate a fix and disclosure, and
credit reporters according to their wishes. Tell users exactly which versions
are affected and how to verify the replacement. Destructive public rollback is
never automated by the snapshot tooling.
