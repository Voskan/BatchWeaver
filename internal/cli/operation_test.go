package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOperationListText(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := run(t, "operation", "list", "--file", exampleYAML)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr=%s", code, stderr)
	}
	if !strings.HasPrefix(stdout, "OPERATION") {
		t.Errorf("missing header:\n%s", stdout)
	}
	// Deterministic lexical order: orders.create, prices.get, users.get.
	iOrders := strings.Index(stdout, "orders.create")
	iPrices := strings.Index(stdout, "prices.get")
	iUsers := strings.Index(stdout, "users.get")
	if iOrders >= iPrices || iPrices >= iUsers {
		t.Errorf("operations not in lexical order:\n%s", stdout)
	}
}

func TestOperationListJSON(t *testing.T) {
	t.Parallel()
	code, stdout, _ := run(t, "operation", "list", "--file", exampleJSON, "--format", "json")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	var doc struct {
		SchemaVersion int `json:"schema_version"`
		Operations    []struct {
			ID string `json:"id"`
		} `json:"operations"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if doc.SchemaVersion != operationListSchemaVersion {
		t.Errorf("schema_version = %d", doc.SchemaVersion)
	}
	if len(doc.Operations) != 3 {
		t.Errorf("operations = %d, want 3", len(doc.Operations))
	}
}

func TestOperationListInvalidConfig(t *testing.T) {
	t.Parallel()
	code, stdout, _ := run(t, "operation", "list", "--file", invalidDir+"/bad-kind.yaml")
	if code != ExitConfigInvalid {
		t.Errorf("exit = %d, want %d", code, ExitConfigInvalid)
	}
	if stdout != "" {
		t.Errorf("stdout should be empty for invalid config:\n%s", stdout)
	}
}

func TestOperationNoSubcommand(t *testing.T) {
	t.Parallel()
	code, _, _ := run(t, "operation")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
}
