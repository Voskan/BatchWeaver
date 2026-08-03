package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// signingWorkflowPath is the workflow whose OIDC identity binds every signature.
const signingWorkflowPath = ".github/workflows/release-signing.yml"

// TestSigningWorkflowIsIdentityBoundAndLeastPrivilege verifies the signing
// workflow's security contract: pinned actions, a protected environment,
// least-privilege elevated scopes, keyless signing with no private key, and
// attestation coverage for every artifact class.
func TestSigningWorkflowIsIdentityBoundAndLeastPrivilege(t *testing.T) {
	root := testRoot(t)
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(signingWorkflowPath)))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)

	for _, required := range []string{
		// Elevated identity is gated behind a protected environment.
		"environment: release-signing",
		// Least privilege: read by default, elevated only in the signing job.
		"permissions:\n  contents: read",
		"id-token: write",
		"attestations: write",
		// Keyless signing and hosted attestation, both pinned by commit.
		"actions/attest-build-provenance@0f67c3f4856b2e3261c31976d6725780e5e4c373",
		"sigstore/cosign-installer@6f9f17788090df1f26f669e9d70d6ae9567deba6",
		"cosign sign-blob --yes",
		// The workflow verifies before it publishes anything.
		"scripts/verify-release-signatures.sh",
		"persist-credentials: false",
		"retention-days: 90",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("signing workflow is missing %q", required)
		}
	}

	// Signatures and attestations must cover every artifact class.
	for _, subject := range []string{
		"dist/*.tar.gz", "dist/*.zip", "dist/*.vsix",
		"dist/SHA256SUMS", "dist/*.spdx.json", "dist/*.cdx.json",
		"dist/release-manifest.json",
	} {
		if !strings.Contains(workflow, subject) {
			t.Errorf("signing workflow does not cover %q", subject)
		}
	}

	// A private signing key must never appear in the workflow.
	for _, forbidden := range []string{
		"COSIGN_PRIVATE_KEY", "cosign.key", "--key ", "secrets.SIGNING_KEY",
	} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("signing workflow references a private key: %q", forbidden)
		}
	}
}

// TestVerifyScriptChecksEveryIdentityDimension verifies that the public
// verification command binds a signature to the subject digest, the repository,
// the signing workflow, the exact tag, and the OIDC issuer. Dropping any one of
// these would accept a signature produced somewhere else.
func TestVerifyScriptChecksEveryIdentityDimension(t *testing.T) {
	root := testRoot(t)
	path := filepath.Join(root, "scripts", "verify-release-signatures.sh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)

	for name, required := range map[string]string{
		"repository binding": "Voskan/BatchWeaver",
		"workflow binding":   ".github/workflows/release-signing.yml",
		"tag binding":        "refs/tags/",
		"issuer binding":     "https://token.actions.githubusercontent.com",
		"identity flag":      "--certificate-identity",
		"issuer flag":        "--certificate-oidc-issuer",
		"subject digest":     "sha256sum",
		"attestation check":  "gh attestation verify",
		"non-zero exit":      "exit 1",
	} {
		if !strings.Contains(script, required) {
			t.Errorf("verification script lacks %s (%q)", name, required)
		}
	}

	// The invariant that matters is the mode git records, because that is what a
	// consumer receives on checkout. Windows working trees do not carry Unix
	// permission bits, so read the mode from the index rather than the filesystem.
	out, err := exec.Command("git", "-C", root, "ls-files", "-s", "scripts/verify-release-signatures.sh").Output()
	if err != nil {
		t.Skipf("git is unavailable to check the recorded file mode: %v", err)
	}
	if !strings.HasPrefix(string(out), "100755") {
		t.Errorf("verification script must be recorded as executable (100755), got %q", strings.TrimSpace(string(out)))
	}
}

// TestVerifyScriptRejectsTamperedIdentity is the negative test for the identity
// contract. Each mutation below is a realistic forgery: a signature made by a
// different repository, from a different workflow, at a different ref, or
// attested by a different issuer. The verification command must not accept any
// of them, which it enforces by building the certificate identity from all of
// those values at once.
func TestVerifyScriptRejectsTamperedIdentity(t *testing.T) {
	root := testRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "scripts", "verify-release-signatures.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)

	// The identity string must be composed from repository, workflow, and tag
	// together. If any component were dropped, a signature from another source
	// would satisfy the check.
	const identity = `IDENTITY="https://github.com/${REPOSITORY}/${WORKFLOW}@refs/tags/${TAG}"`
	if !strings.Contains(script, identity) {
		t.Fatalf("identity must bind repository, workflow, and tag together; got no match for %s", identity)
	}

	for _, tampered := range []struct{ name, fragment string }{
		{"wrong repository accepted", `IDENTITY="https://github.com/${WORKFLOW}`},
		{"missing tag binding", `@refs/heads/`},
		{"identity check disabled", `--insecure-ignore-tlog`},
		{"issuer check skipped", `--certificate-oidc-issuer=""`},
		{"failures ignored", `exit 0 # ignore failures`},
	} {
		if strings.Contains(script, tampered.fragment) {
			t.Errorf("verification script would allow %s (%q)", tampered.name, tampered.fragment)
		}
	}

	// A failed verification must be fatal, never merely reported.
	if !strings.Contains(script, `if [ "$failed" -ne 0 ] || [ "$verified" -eq 0 ]; then`) {
		t.Error("verification script must fail when any artifact fails or nothing was verified")
	}
}

// TestSignatureGateIsDocumented requires the public verification guide to exist
// and to name the commands a consumer actually runs, so signature verification
// is reproducible outside this repository.
func TestSignatureGateIsDocumented(t *testing.T) {
	root := testRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "guides", "verify-release-artifacts.md"))
	if err != nil {
		t.Fatal(err)
	}
	guide := string(data)
	for _, required := range []string{
		"cosign verify-blob",
		"--certificate-identity",
		"--certificate-oidc-issuer",
		"gh attestation verify",
		"sha256sum",
		"scripts/verify-release-signatures.sh",
	} {
		if !strings.Contains(guide, required) {
			t.Errorf("verification guide is missing %q", required)
		}
	}
}
