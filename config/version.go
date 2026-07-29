package config

// CurrentSchemaVersion is the configuration schema version this build produces
// and treats as canonical.
const CurrentSchemaVersion = 1

// supportedSchemaVersions is the set of schema versions this build can load.
// Future migrations will extend this set deliberately.
var supportedSchemaVersions = map[int]struct{}{
	1: {},
}

// SchemaVersionSupported reports whether v is a schema version this build can
// load.
func SchemaVersionSupported(v int) bool {
	_, ok := supportedSchemaVersions[v]
	return ok
}
