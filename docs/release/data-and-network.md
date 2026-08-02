# Data and Network Inventory

Runtime state and memoized values are process-local and scoped. Compiler caches,
profiles, daemon discovery, and release outputs are local files under explicit
workspace or output paths. Profiles contain aggregates and hashed identifiers,
not raw keys. Users control retention by deleting those paths; `daemon clean`
and `release clean --output` provide bounded cleanup.

Potential network operations are Go module download, npm install, vulnerability
database refresh, external documentation link checking, and an explicitly
authorized future publication. Remote OpenAPI references are rejected. No
runtime telemetry, source upload, profile upload, or background release check is
implemented.
