package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Voskan/BatchWeaver/diagnostics"
)

const (
	exampleYAML = "../examples/configuration/batchweaver.yaml"
	exampleJSON = "../examples/configuration/batchweaver.json"
)

func loadPath(t *testing.T, path string) LoadResult {
	t.Helper()
	res, err := Load(context.Background(), LoadOptions{Path: path})
	if err != nil {
		t.Fatalf("Load(%q) error: %v", path, err)
	}
	return res
}

func TestLoadExampleYAMLValid(t *testing.T) {
	t.Parallel()
	res := loadPath(t, exampleYAML)
	if res.HasErrors() {
		t.Fatalf("example YAML has errors:\n%s", renderDiags(res.Diagnostics))
	}
	if res.Catalog.Len() != 3 {
		t.Errorf("operations = %d, want 3", res.Catalog.Len())
	}
	ids := res.Catalog.IDs()
	want := []string{"orders.create", "prices.get", "users.get"}
	for i := range want {
		if string(ids[i]) != want[i] {
			t.Errorf("id[%d] = %q, want %q", i, ids[i], want[i])
		}
	}
	if res.Digest == "" || !strings.HasPrefix(res.Digest, "sha256:") {
		t.Errorf("digest = %q", res.Digest)
	}
}

func TestYAMLAndJSONSameDigest(t *testing.T) {
	t.Parallel()
	y := loadPath(t, exampleYAML)
	j := loadPath(t, exampleJSON)
	if y.HasErrors() || j.HasErrors() {
		t.Fatalf("example configs have errors")
	}
	if y.Digest != j.Digest {
		t.Errorf("YAML and JSON digests differ:\n yaml=%s\n json=%s", y.Digest, j.Digest)
	}
}

func TestRenderJSONDeterministic(t *testing.T) {
	t.Parallel()
	res := loadPath(t, exampleYAML)
	a, err := RenderJSON(res.Config)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	b, _ := RenderJSON(res.Config)
	if string(a) != string(b) {
		t.Errorf("RenderJSON not deterministic")
	}
	if len(a) == 0 || a[len(a)-1] != '\n' {
		t.Errorf("RenderJSON must end with a newline")
	}
}

func writeConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestUnknownFieldSuggestion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeConfig(t, dir, "batchweaver.yaml", `version: 1
operations:
  users.get:
    kind: read-only
    scheduler:
      min_sze: 10
`)
	res := loadPath(t, path)
	found := false
	for _, d := range res.Diagnostics.Diagnostics() {
		if d.Code == "BWCFG021" && strings.Contains(d.Message, "min_sze") {
			found = true
			if !strings.Contains(d.Details, "min_size") {
				t.Errorf("unknown field diagnostic missing suggestion: %+v", d)
			}
		}
	}
	if !found {
		t.Errorf("expected BWCFG021 unknown field diagnostic")
	}
}

func TestDuplicateKeyRejected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeConfig(t, dir, "batchweaver.yaml", `version: 1
version: 2
`)
	res := loadPath(t, path)
	if !hasCode(res.Diagnostics, "BWCFG002") {
		t.Errorf("expected duplicate-key diagnostic BWCFG002; got:\n%s", renderDiags(res.Diagnostics))
	}
}

func TestMissingVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeConfig(t, dir, "batchweaver.yaml", "operations: {}\n")
	res := loadPath(t, path)
	if !hasCode(res.Diagnostics, "BWCFG015") {
		t.Errorf("expected missing-version diagnostic BWCFG015")
	}
}

func TestIncludeAndMerge(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, "shared.yaml", `version: 1
operations:
  users.get:
    kind: read-only
    scalar: {symbol: github.com/example/app/users.GetUser}
    batch: {symbol: github.com/example/app/users.GetUsersBatch}
`)
	main := writeConfig(t, dir, "batchweaver.yaml", `version: 1
include:
  - shared.yaml
operations:
  prices.get:
    kind: read-only
    scalar: {symbol: github.com/example/app/pricing.GetPrice}
    batch: {symbol: github.com/example/app/pricing.GetPricesBatch}
`)
	res := loadPath(t, main)
	if res.HasErrors() {
		t.Fatalf("include merge errors:\n%s", renderDiags(res.Diagnostics))
	}
	if res.Catalog.Len() != 2 {
		t.Errorf("merged operations = %d, want 2", res.Catalog.Len())
	}
	if len(res.Files) != 2 {
		t.Errorf("files = %v, want 2 files", res.Files)
	}
}

func TestIncludeDuplicateOperationConflict(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, "shared.yaml", `version: 1
operations:
  users.get:
    kind: read-only
    scalar: {symbol: github.com/example/app/users.GetUser}
    batch: {symbol: github.com/example/app/users.GetUsersBatch}
`)
	main := writeConfig(t, dir, "batchweaver.yaml", `version: 1
include:
  - shared.yaml
operations:
  users.get:
    kind: idempotent-write
    scalar: {symbol: github.com/example/app/users.SetUser}
    batch: {symbol: github.com/example/app/users.SetUsersBatch}
`)
	res := loadPath(t, main)
	if !hasCode(res.Diagnostics, "BWCFG019") {
		t.Errorf("expected duplicate-operation conflict BWCFG019; got:\n%s", renderDiags(res.Diagnostics))
	}
}

func TestIncludeReplaceAllowed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, "shared.yaml", `version: 1
operations:
  users.get:
    kind: read-only
    scalar: {symbol: github.com/example/app/users.GetUser}
    batch: {symbol: github.com/example/app/users.GetUsersBatch}
`)
	main := writeConfig(t, dir, "batchweaver.yaml", `version: 1
include:
  - shared.yaml
operations:
  users.get:
    replace: true
    kind: read-only
    scalar: {symbol: github.com/example/app/users.GetUserV2}
    batch: {symbol: github.com/example/app/users.GetUsersBatchV2}
`)
	res := loadPath(t, main)
	if res.HasErrors() {
		t.Fatalf("replace should be allowed:\n%s", renderDiags(res.Diagnostics))
	}
	spec, _ := res.Catalog.Get("users.get")
	if !strings.Contains(spec.ScalarSymbol().String(), "GetUserV2") {
		t.Errorf("replace did not take effect: %s", spec.ScalarSymbol())
	}
}

func TestIncludeCycle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, "a.yaml", "version: 1\ninclude:\n  - b.yaml\n")
	writeConfig(t, dir, "b.yaml", "version: 1\ninclude:\n  - a.yaml\n")
	res := loadPath(t, filepath.Join(dir, "a.yaml"))
	if !hasCode(res.Diagnostics, "BWCFG012") {
		t.Errorf("expected include-cycle diagnostic BWCFG012; got:\n%s", renderDiags(res.Diagnostics))
	}
}

func TestRemoteIncludeForbidden(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeConfig(t, dir, "batchweaver.yaml", "version: 1\ninclude:\n  - https://evil.example/x.yaml\n")
	res := loadPath(t, path)
	if !hasCode(res.Diagnostics, "BWCFG014") {
		t.Errorf("expected remote-include diagnostic BWCFG014")
	}
}

func TestDiscovery(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, dir, "batchweaver.yaml", "version: 1\noperations: {}\n")
	res, err := Load(context.Background(), LoadOptions{WorkingDirectory: sub, Discover: true, RepositoryRoot: dir})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !res.Found {
		t.Errorf("discovery did not find upward config")
	}
}

func TestDiscoveryNotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res, err := Load(context.Background(), LoadOptions{WorkingDirectory: dir, Discover: true, RepositoryRoot: dir})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !hasCode(res.Diagnostics, "BWCFG017") {
		t.Errorf("expected not-found diagnostic BWCFG017")
	}
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Load(ctx, LoadOptions{Path: exampleYAML})
	if err == nil {
		t.Errorf("canceled context should return an error")
	}
}

func hasCode(c diagnostics.Collection, code diagnostics.Code) bool {
	for _, d := range c.Diagnostics() {
		if d.Code == code {
			return true
		}
	}
	return false
}

func renderDiags(c diagnostics.Collection) string {
	var b strings.Builder
	_ = diagnostics.NewTextFormatter().Format(&b, c)
	return b.String()
}
