package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

const (
	exampleYAML = "../../examples/configuration/batchweaver.yaml"
	exampleJSON = "../../examples/configuration/batchweaver.json"
	invalidDir  = "../../testdata/config/invalid"
)

func TestConfigValidateValid(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := run(t, "config", "validate", "--file", exampleYAML)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "Configuration is valid.") || !strings.Contains(stdout, "Operations: 3") {
		t.Errorf("stdout:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Digest: sha256:") {
		t.Errorf("missing digest:\n%s", stdout)
	}
}

func TestConfigValidateInvalid(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := run(t, "config", "validate", "--file", invalidDir+"/retry-nonidempotent.yaml")
	if code != ExitConfigInvalid {
		t.Fatalf("exit = %d, want %d", code, ExitConfigInvalid)
	}
	if stdout != "" {
		t.Errorf("stdout should be empty on invalid config, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "BWOP013") {
		t.Errorf("stderr missing diagnostic:\n%s", stderr)
	}
}

func TestConfigValidateJSON(t *testing.T) {
	t.Parallel()
	code, stdout, _ := run(t, "config", "validate", "--file", invalidDir+"/unknown-field.yaml", "--format", "json")
	if code != ExitConfigInvalid {
		t.Fatalf("exit = %d", code)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if doc["schema_version"] == nil {
		t.Errorf("json output missing schema_version")
	}
}

func TestConfigValidateNotFound(t *testing.T) {
	t.Parallel()
	code, _, _ := run(t, "config", "validate", "--file", "/no/such/batchweaver.yaml")
	if code != ExitConfigNotFound {
		t.Errorf("exit = %d, want %d", code, ExitConfigNotFound)
	}
}

func TestConfigValidateBadFormat(t *testing.T) {
	t.Parallel()
	code, _, stderr := run(t, "config", "validate", "--file", exampleYAML, "--format", "xml")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "unknown format") {
		t.Errorf("stderr:\n%s", stderr)
	}
}

func TestConfigPrintJSON(t *testing.T) {
	t.Parallel()
	code, stdout, _ := run(t, "config", "print", "--file", exampleYAML, "--format", "json")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("print output not valid JSON: %v", err)
	}
	if doc["version"].(float64) != 1 {
		t.Errorf("version = %v", doc["version"])
	}
}

func TestConfigPrintYAMLDeferred(t *testing.T) {
	t.Parallel()
	code, _, stderr := run(t, "config", "print", "--file", exampleYAML, "--format", "yaml")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "not yet implemented") {
		t.Errorf("stderr:\n%s", stderr)
	}
}

func TestConfigNoSubcommand(t *testing.T) {
	t.Parallel()
	code, _, stderr := run(t, "config")
	if code != ExitUsage {
		t.Errorf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, "subcommand") {
		t.Errorf("stderr:\n%s", stderr)
	}
}
