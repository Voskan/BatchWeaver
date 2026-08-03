package release

import "testing"

// TestLicenseExpressionPolicy pins the SPDX evaluation rules. The important
// case is a disjunction: a dual-licensed dependency such as
// "MIT OR GPL-3.0-or-later" lets the consumer choose the permissive option, so
// rejecting it because the string contains "GPL" would refuse a license the
// project is entitled to take.
func TestLicenseExpressionPolicy(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want bool
		why  string
	}{
		// Plain permissive identifiers.
		{"MIT", true, "permissive"},
		{"Apache-2.0", true, "permissive"},
		{"BSD-3-Clause", true, "permissive"},
		{"ISC", true, "permissive"},
		{"Unlicense", true, "public domain dedication"},

		// Copyleft alone is refused.
		{"GPL-3.0-or-later", false, "strong copyleft"},
		{"GPL-2.0-only", false, "strong copyleft"},
		{"AGPL-3.0", false, "network copyleft"},
		{"SSPL-1.0", false, "server side public license"},
		{"LGPL-3.0", false, "weak copyleft is out of policy for bundled code"},

		// Disjunction: acceptable when any operand is.
		{"MIT OR GPL-3.0-or-later", true, "consumer may choose MIT"},
		{"GPL-3.0-or-later OR MIT", true, "order must not matter"},
		{"(MIT OR GPL-3.0-or-later)", true, "parenthesized disjunction"},
		{"GPL-2.0-only OR AGPL-3.0", false, "every option is copyleft"},

		// Conjunction: every operand must be acceptable.
		{"MIT AND Apache-2.0", true, "both permissive"},
		{"MIT AND GPL-3.0-only", false, "conjunction imposes copyleft"},

		// Mixed precedence: AND binds tighter than OR.
		{"MIT AND GPL-3.0-only OR Apache-2.0", true, "the Apache-2.0 branch is acceptable"},
		{"GPL-3.0-only OR (MIT AND Apache-2.0)", true, "the permissive branch is acceptable"},
		{"(GPL-3.0-only OR AGPL-3.0) AND MIT", false, "the copyleft conjunct governs"},

		// Exceptions attach to their license identifier.
		{"GPL-2.0 WITH Classpath-exception-2.0", false, "still governed by GPL"},
		{"MIT WITH some-exception", true, "still governed by MIT"},

		// Malformed input must never be accepted silently.
		{"", false, "empty"},
		{"MIT OR", false, "dangling operator"},
		{"(MIT", false, "unbalanced parenthesis"},
		{"OR MIT", false, "leading operator"},
	} {
		if got := licenseExpressionAcceptable(tc.expr); got != tc.want {
			t.Errorf("licenseExpressionAcceptable(%q) = %v, want %v (%s)", tc.expr, got, tc.want, tc.why)
		}
	}
}

// TestValidateNPMLicensesAcceptsDualLicensedDependency covers the exact
// dependency that the production campaign rejected: jszip reaches the tree
// through the VS Code packaging tool and is dual licensed.
func TestValidateNPMLicensesAcceptsDualLicensedDependency(t *testing.T) {
	packages := []npmPackage{
		{Name: "jszip", Version: "3.10.1", License: "(MIT OR GPL-3.0-or-later)", Path: "node_modules/jszip"},
		{Name: "typescript", Version: "5.9.3", License: "Apache-2.0", Path: "node_modules/typescript"},
	}
	if err := validateNPMLicenses(packages); err != nil {
		t.Errorf("dual-licensed dependency rejected: %v", err)
	}
}

// TestValidateNPMLicensesRejectsCopyleft keeps the policy meaningful: a
// dependency offered only under copyleft is still refused.
func TestValidateNPMLicensesRejectsCopyleft(t *testing.T) {
	packages := []npmPackage{
		{Name: "copyleft-only", Version: "1.0.0", License: "GPL-3.0-or-later", Path: "node_modules/copyleft-only"},
	}
	if err := validateNPMLicenses(packages); err == nil {
		t.Error("copyleft-only dependency was accepted")
	}
}

// TestUnevaluableLicenseRequiresRecordedReview covers the npm conventions that
// carry no machine-checkable grant. They are allowed only for dependencies a
// maintainer has reviewed and recorded, so the audit cannot be silenced by an
// upstream package simply omitting SPDX metadata.
func TestUnevaluableLicenseRequiresRecordedReview(t *testing.T) {
	reviewed := []npmPackage{
		{Name: "@vscode/vsce-sign", Version: "2.0.9", License: "SEE LICENSE IN LICENSE.txt", Path: "node_modules/@vscode/vsce-sign"},
		{Name: "@vscode/vsce-sign-darwin-arm64", Version: "2.0.2", License: "", Path: "node_modules/@vscode/vsce-sign-darwin-arm64"},
		{Name: "@esbuild/darwin-arm64", Version: "0.28.1", License: "", Path: "node_modules/@esbuild/darwin-arm64"},
	}
	if err := validateNPMLicenses(reviewed); err != nil {
		t.Errorf("reviewed build-time dependency rejected: %v", err)
	}

	for _, unreviewed := range []npmPackage{
		{Name: "mystery", Version: "1.0.0", License: "SEE LICENSE IN LICENSE.txt", Path: "node_modules/mystery"},
		{Name: "proprietary", Version: "1.0.0", License: "UNLICENSED", Path: "node_modules/proprietary"},
		{Name: "custom-ref", Version: "1.0.0", License: "LicenseRef-Custom", Path: "node_modules/custom-ref"},
		{Name: "no-metadata", Version: "1.0.0", License: "", Path: "node_modules/no-metadata"},
	} {
		if err := validateNPMLicenses([]npmPackage{unreviewed}); err == nil {
			t.Errorf("unevaluable license accepted without a recorded review: %s", unreviewed.Name)
		}
	}
}
