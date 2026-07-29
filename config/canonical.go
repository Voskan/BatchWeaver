package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"sort"

	"github.com/Voskan/BatchWeaver/operation"
)

// Digest returns a deterministic SHA-256 digest of the normalized configuration,
// formatted as "sha256:<lowercase-hex>". It covers the schema version, top-level
// semantic settings, extension data, and every operation spec in ID order. It
// excludes source file paths, source positions, and include order, so the same
// semantic configuration expressed in YAML or JSON yields the same digest.
func Digest(cfg Config) string {
	h := sha256.New()
	fmt.Fprintf(h, "version=%d\n", cfg.Version)
	fmt.Fprintf(h, "compiler.mode=%s\n", cfg.Compiler.Mode)
	fmt.Fprintf(h, "runtime.default_scope=%s\n", cfg.Runtime.DefaultScope)
	fmt.Fprintf(h, "security.cross_scope_batching=%t\n", cfg.Security.CrossScopeBatching)
	fmt.Fprintf(h, "security.raw_key_observability=%t\n", cfg.Security.RawKeyObservability)
	fmt.Fprintf(h, "observability.metrics=%t\n", cfg.Observability.Metrics)
	fmt.Fprintf(h, "observability.tracing=%t\n", cfg.Observability.Tracing)
	fmt.Fprintf(h, "observability.logging=%s\n", cfg.Observability.Logging)

	exts := append([]operation.Extension(nil), cfg.Extensions...)
	sort.Slice(exts, func(i, j int) bool { return exts[i].Namespace < exts[j].Namespace })
	for _, e := range exts {
		fmt.Fprintf(h, "ext:%s=%s\n", e.Namespace, base64.StdEncoding.EncodeToString(e.Data))
	}

	for _, sp := range cfg.Catalog.List() { // catalog.List is ID-sorted
		_, _ = io.WriteString(h, sp.Canonical())
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
