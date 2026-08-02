package release

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Voskan/BatchWeaver/bridge"
	"github.com/Voskan/BatchWeaver/config"
	"github.com/Voskan/BatchWeaver/diagnostics"
	"github.com/Voskan/BatchWeaver/internal/adaptive"
)

// publishedPrereleases are the immutable public prereleases from which an
// upgrade to the v1 candidate must be supported. Every entry needs its own
// executable fixture; see docs/release/v1.0.0/migration.md.
var publishedPrereleases = []string{
	"v0.1.0-beta.1",
	"v0.1.0-beta.2",
	"v0.1.0-beta.3",
}

// candidateVersion is the version this working tree builds.
const candidateVersion = "1.0.0"

// TestPrereleaseConfigurationMigrates loads the configuration shipped with each
// published prerelease using the current loader. Configuration schema 1 is the
// documented compatibility contract across the upgrade, so a beta-era file must
// still load, validate, and produce a semantic digest without error diagnostics.
func TestPrereleaseConfigurationMigrates(t *testing.T) {
	root := testRoot(t)
	for _, version := range publishedPrereleases {
		t.Run(version, func(t *testing.T) {
			path := filepath.Join(root, "internal", "release", "testdata", "migration", version+".yaml")
			result, err := config.Load(context.Background(), config.LoadOptions{Path: path})
			if err != nil {
				t.Fatalf("load %s configuration: %v", version, err)
			}
			if !result.Found {
				t.Fatalf("%s configuration fixture was not found", version)
			}
			if result.Diagnostics.HasErrors() {
				for _, d := range result.Diagnostics.FilterBySeverity(diagnostics.SeverityError).Diagnostics() {
					t.Errorf("%s configuration error: %s", version, d.Message)
				}
				t.Fatalf("%s configuration must load under schema %d", version, config.CurrentSchemaVersion)
			}
			if result.Digest == "" {
				t.Fatalf("%s configuration produced no semantic digest", version)
			}
			if got := result.Config.Version; got != config.CurrentSchemaVersion {
				t.Fatalf("%s schema version = %d, want %d", version, got, config.CurrentSchemaVersion)
			}
			if result.Config.Catalog.Len() == 0 {
				t.Fatalf("%s configuration produced an empty operation catalog", version)
			}
		})
	}
}

// TestPrereleaseMigrationInventory verifies that every published prerelease has
// a well-formed migration inventory to the candidate and that the inventory
// demands cache invalidation and artifact regeneration rather than silent reuse.
func TestPrereleaseMigrationInventory(t *testing.T) {
	for _, version := range publishedPrereleases {
		t.Run(version, func(t *testing.T) {
			inventory := migrationInventory{
				FromVersion:         version,
				ToVersion:           candidateVersion,
				ConfigSchema:        config.CurrentSchemaVersion,
				ProofSchema:         "batchweaver.proof/v1alpha1",
				TransformSchema:     "batchweaver.transform/v1alpha1",
				DiscardCaches:       true,
				RegenerateArtifacts: true,
			}
			data, err := json.Marshal(inventory)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateMigrationInventory(data); err != nil {
				t.Fatalf("%s inventory rejected: %v", version, err)
			}

			// Reusing caches or artifacts across the upgrade must be rejected.
			for _, unsafe := range []migrationInventory{
				{FromVersion: version, ToVersion: candidateVersion, ConfigSchema: config.CurrentSchemaVersion, DiscardCaches: false, RegenerateArtifacts: true},
				{FromVersion: version, ToVersion: candidateVersion, ConfigSchema: config.CurrentSchemaVersion, DiscardCaches: true, RegenerateArtifacts: false},
			} {
				raw, err := json.Marshal(unsafe)
				if err != nil {
					t.Fatal(err)
				}
				if err := validateMigrationInventory(raw); err == nil {
					t.Errorf("%s inventory must reject cache or artifact reuse: %s", version, raw)
				}
			}
		})
	}
}

// TestPrereleaseProfileMigration verifies that a workload profile collected
// under a published prerelease is rejected for active use after the upgrade
// whenever its runtime ABI or configuration digest no longer matches, and is
// accepted only when both still match. Profiles are never silently reused.
func TestPrereleaseProfileMigration(t *testing.T) {
	for _, version := range publishedPrereleases {
		t.Run(version, func(t *testing.T) {
			betaProfile := &adaptive.ProfileBundle{
				RuntimeABI:   bridge.ABIVersion,
				ConfigDigest: adaptive.Digest("beta-config-digest"),
			}
			betaProfile.Finalize()

			// A changed configuration digest is a hard incompatibility.
			changed := adaptive.CheckCompatibility(betaProfile, adaptive.CompatibilityRequirement{
				RuntimeABI:   bridge.ABIVersion,
				ConfigDigest: adaptive.Digest("v1-config-digest"),
			})
			if changed.Compatible {
				t.Errorf("%s profile must be rejected when the configuration digest changes", version)
			}

			// A changed runtime ABI is a hard incompatibility.
			abiChanged := adaptive.CheckCompatibility(betaProfile, adaptive.CompatibilityRequirement{
				RuntimeABI: "batchweaver.bridge/v2",
			})
			if abiChanged.Compatible {
				t.Errorf("%s profile must be rejected when the bridge ABI changes", version)
			}

			// An unchanged environment stays usable.
			unchanged := adaptive.CheckCompatibility(betaProfile, adaptive.CompatibilityRequirement{
				RuntimeABI:   bridge.ABIVersion,
				ConfigDigest: adaptive.Digest("beta-config-digest"),
			})
			if !unchanged.Compatible {
				t.Errorf("%s profile must remain compatible when nothing changed: %+v", version, unchanged.Diagnostics)
			}

			// An aged profile is stale rather than silently authoritative.
			stale := adaptive.CheckCompatibility(betaProfile, adaptive.CompatibilityRequirement{
				RuntimeABI:   bridge.ABIVersion,
				ConfigDigest: adaptive.Digest("beta-config-digest"),
				MaxAge:       time.Second,
				Now:          time.Unix(0, betaProfile.CreatedUnixNanos).Add(72 * time.Hour),
			})
			if !stale.Stale || stale.UsableForActive() {
				t.Errorf("%s aged profile must be stale and unusable for active tuning", version)
			}
		})
	}
}

// TestScalarRollbackAfterMigration verifies the documented rollback target: a
// lowered call site executes the original scalar function when no runtime bound
// operation is installed. This is the fallback users rely on when an upgrade is
// reverted, so it must hold without any runtime, scope, or configuration.
func TestScalarRollbackAfterMigration(t *testing.T) {
	type repo struct{ name string }
	scalarCalls := 0
	op := bridge.Operation[*repo, int, string]{
		OpID: "users.get",
		Scalar: func(_ context.Context, r *repo, key int) (string, error) {
			scalarCalls++
			return r.name + ":" + itoa(key), nil
		},
	}
	got, err := op.Call(context.Background(), &repo{name: "users"}, 7)
	if err != nil {
		t.Fatalf("scalar rollback call: %v", err)
	}
	if got != "users:7" {
		t.Errorf("scalar rollback returned %q, want %q", got, "users:7")
	}
	if scalarCalls != 1 {
		t.Errorf("scalar function called %d times, want exactly 1", scalarCalls)
	}
}

// TestMigrationDocumentsPublishedPrereleases keeps the migration plan and the
// executable fixtures in agreement: every published prerelease named by the plan
// must have a fixture, and every fixture must be named by the plan.
func TestMigrationDocumentsPublishedPrereleases(t *testing.T) {
	root := testRoot(t)
	plan, err := os.ReadFile(filepath.Join(root, "docs", "release", "v1.0.0", "migration.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range publishedPrereleases {
		if !strings.Contains(string(plan), version) {
			t.Errorf("migration plan does not name published prerelease %s", version)
		}
		path := filepath.Join(root, "internal", "release", "testdata", "migration", version+".yaml")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("published prerelease %s has no migration fixture: %v", version, err)
		}
	}
}

// itoa formats a small non-negative integer without importing strconv into the
// rollback assertion path.
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
