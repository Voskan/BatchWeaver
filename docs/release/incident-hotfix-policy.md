# Incident, Hotfix, and Supersession Policy

A new prerelease is justified for incorrect transformation, partition leakage,
cancellation/deadline violation, corruption, vulnerability, broken installation,
or critical compatibility failure. Never change an existing tag or silently
replace assets.

Mark the affected release unsafe or superseded, identify versions/strategies,
recommend feature disablement, scalar fallback, source revert, and cache/proof
invalidation, retain forensic artifacts, coordinate security disclosure, and
publish a higher version. Retraction in `go.mod` is reserved for serious defects
and must state the reason. Deleting evidence is not rollback.
