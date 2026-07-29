package config

import "errors"

// ErrInternal indicates a configuration-loading failure that prevents reliable
// diagnostic production, as opposed to an ordinary invalid-configuration case
// (which is reported through the diagnostics in a LoadResult). It is returned as
// the error from Load only for internal invariant failures.
var ErrInternal = errors.New("internal configuration error")
