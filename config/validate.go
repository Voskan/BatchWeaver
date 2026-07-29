package config

import (
	"fmt"

	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/Voskan/BatchWeaver/internal/configdecode"
)

// validateSemantic performs cross-configuration semantic checks that are not
// captured by per-operation validation, such as enforcing that operations do
// not request cross-scope batching while the security default forbids it. It
// returns additional diagnostics.
func validateSemantic(cfg Config) diagnostics.Collection {
	var diags diagnostics.Collection
	for _, sp := range cfg.Catalog.List() {
		if !cfg.Security.CrossScopeBatching && sp.PartitionContract().CrossRootScopeAllowed() {
			diags.Add(diagnostics.Diagnostic{
				Code:     configdecode.CodeSecurity,
				Severity: diagnostics.SeverityError,
				Message:  fmt.Sprintf("operation %q requests cross-root-scope batching but security.cross_scope_batching is disabled", sp.ID()),
				Source:   "config",
				Range:    sp.Source().Range,
			})
		}
	}
	return diags
}
