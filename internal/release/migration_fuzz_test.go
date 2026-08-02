package release

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/Voskan/BatchWeaver/config"
)

type migrationInventory struct {
	FromVersion         string `json:"from_version"`
	ToVersion           string `json:"to_version"`
	ConfigSchema        int    `json:"config_schema"`
	ProofSchema         string `json:"proof_schema"`
	TransformSchema     string `json:"transform_schema"`
	DiscardCaches       bool   `json:"discard_caches"`
	RegenerateArtifacts bool   `json:"regenerate_artifacts"`
}

func validateMigrationInventory(data []byte) error {
	var inventory migrationInventory
	if err := strictJSON(data, &inventory); err != nil {
		return err
	}
	if inventory.FromVersion == "" || inventory.ToVersion == "" || !config.SchemaVersionSupported(inventory.ConfigSchema) {
		return errInvalidMigration
	}
	if !inventory.DiscardCaches || !inventory.RegenerateArtifacts {
		return errInvalidMigration
	}
	return nil
}

var errInvalidMigration = &migrationError{"invalid migration inventory"}

type migrationError struct{ message string }

func (e *migrationError) Error() string { return e.message }

func strictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errInvalidMigration
		}
		return err
	}
	return nil
}

func FuzzMigrationInventory(f *testing.F) {
	f.Add([]byte(`{"from_version":"0.1.0-beta.3","to_version":"1.0.0","config_schema":1,"proof_schema":"batchweaver.proof/v1alpha1","transform_schema":"batchweaver.transform/v1alpha1","discard_caches":true,"regenerate_artifacts":true}`))
	f.Add([]byte(`{"from_version":"0.1.0-beta.1","to_version":"1.0.0","config_schema":0}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		_ = validateMigrationInventory(data)
	})
}
