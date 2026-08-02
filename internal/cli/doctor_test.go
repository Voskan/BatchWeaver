package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorBundleExcludesSensitiveValues(t *testing.T) {
	secrets := []string{"github_pat_secret-value", "postgres://user:pass@db", "Bearer production-token", "/Users/private/repository", "SELECT secret FROM customer"}
	for i, value := range secrets {
		t.Setenv("BATCHWEAVER_SECRET_TEST_"+string(rune('A'+i)), value)
	}
	path := filepath.Join(t.TempDir(), "doctor.tar.gz")
	var stdout, stderr bytes.Buffer
	code := New(&stdout, &stderr).Run(context.Background(), []string{"doctor", "--bundle", path})
	if code != ExitOK {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	_, err = tr.Next()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(tr)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range secrets {
		if strings.Contains(string(data), secret) {
			t.Fatalf("bundle leaked sensitive value")
		}
	}
	if !strings.Contains(string(data), `"network_access": "none"`) {
		t.Fatalf("missing privacy contract: %s", data)
	}
}

func TestDoctorRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doctor.tar.gz")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := New(&stdout, &stderr).Run(context.Background(), []string{"doctor", "--bundle", path})
	if code != ExitError || string(mustRead(t, path)) != "keep" {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
