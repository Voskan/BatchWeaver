package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdapterListCategoryNetwork(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "adapter", "list", "--category=network")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"graphql/gqlgen", "grpc-go", "http/openapi"} {
		if !strings.Contains(out, want) {
			t.Errorf("network list missing %q", want)
		}
	}
	if strings.Contains(out, "database/sql") {
		t.Errorf("network list should not include backend adapters:\n%s", out)
	}
}

func TestGraphQLInspect(t *testing.T) {
	t.Parallel()
	code, out, stderr := run(t, "graphql", "inspect", "--operation",
		"query Q { orders { id customer { name } } }")
	if code != ExitOK {
		t.Fatalf("exit = %d stderr=%s", code, stderr)
	}
	for _, want := range []string{"wave 0", "orders", "wave 1", "orders.customer"} {
		if !strings.Contains(out, want) {
			t.Errorf("graphql inspect missing %q:\n%s", want, out)
		}
	}
}

func TestGraphQLInspectReject(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "graphql", "inspect", "--operation", "query { orders {")
	if code != ExitConfigInvalid {
		t.Errorf("exit = %d, want %d", code, ExitConfigInvalid)
	}
	if !strings.Contains(out, "BW7106") {
		t.Errorf("expected parse diagnostic:\n%s", out)
	}
}

func TestHTTPVerify(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "http", "verify")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "PASS") || !strings.Contains(out, "bwcontract_") {
		t.Errorf("http verify output:\n%s", out)
	}
}

func TestGRPCInspect(t *testing.T) {
	t.Parallel()
	code, out, _ := run(t, "grpc", "inspect", "--scalar=/s/Get", "--batch=/s/Batch", "--key=id", "--response-key=id")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "partition") || !strings.Contains(out, "/s/Batch") {
		t.Errorf("grpc inspect output:\n%s", out)
	}
}

func TestOpenAPIValidate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	doc := `openapi: 3.1.0
paths:
  /users:batchGet:
    post:
      operationId: batchGetUsers
      x-batchweaver:
        scalar-operation-id: users.get
        mode: keyed
        response-key-path: /request_id
        maximum-items: 200
`
	f := filepath.Join(dir, "api.yaml")
	if err := os.WriteFile(f, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, stderr := run(t, "openapi", "validate", "--file", f)
	if code != ExitOK {
		t.Fatalf("exit = %d stderr=%s", code, stderr)
	}
	if !strings.Contains(out, "1 batch binding") {
		t.Errorf("validate output:\n%s", out)
	}
	code, out, _ = run(t, "openapi", "inspect", "--file", f)
	if code != ExitOK || !strings.Contains(out, "users.get") {
		t.Errorf("inspect output:\n%s", out)
	}
}
