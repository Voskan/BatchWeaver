package config

// CurrentSchemaVersion is the version of the BatchWeaver configuration schema
// that this build understands.
//
// It is incremented whenever the configuration format changes in a way that
// requires migration or compatibility handling. Loading logic introduced in
// later prompts uses it to detect and adapt to older or newer configuration
// files.
const CurrentSchemaVersion = 1
