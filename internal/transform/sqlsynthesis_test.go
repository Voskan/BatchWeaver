package transform

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Voskan/BatchWeaver/internal/adapter"
)

func TestBuildSQLSynthesisPlanTypeChecksAndBuildsOverlay(t *testing.T) {
	root := t.TempDir()
	writeSQLFixture(t, filepath.Join(root, "go.mod"), "module example.com/sqlfixture\n\ngo 1.26\n")
	writeSQLFixture(t, filepath.Join(root, "query", "query.go"), "package query\n\nfunc Existing() string { return \"ok\" }\n")
	parsed, rejection := adapter.ParseExactKeySelect("SELECT u.tenant_id, u.id, p.display_name FROM users u LEFT JOIN profiles p ON p.user_id = u.id WHERE u.tenant_id = $1 AND u.id = $2")
	if rejection != nil {
		t.Fatal(rejection)
	}
	synthesis, rejection := adapter.SynthesizeExactKey(adapter.SynthInput{
		Query: parsed, KeyTypes: []string{"uuid", "bigint"},
		JoinCardinality: adapter.JoinCardinalityAtMostOne,
	})
	if rejection != nil {
		t.Fatal(rejection)
	}
	plan, err := BuildSQLSynthesisPlan(context.Background(), SQLPlanRequest{
		Workspace: root, PackageName: "query", PackagePath: "example.com/sqlfixture/query",
		Output: "query/batchweaver_sql_gen.go", Constant: "GetUsersBatchSQL",
		Operation: "users.get", Synthesis: synthesis,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Transformations) != 1 || plan.Transformations[0].Strategy != StrategyBoundedJoinSQLSynthesis || plan.Validation.TypeCheck != ValidationPassed {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if sourceMap := BuildSourceMap(plan); len(sourceMap.Segments) != 1 || sourceMap.Segments[0].Role != RoleSQLSynthesis {
		t.Fatalf("unexpected source map: %+v", sourceMap)
	}
	if err := VerifySQLSynthesisPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := SavePlan(root, plan); err != nil {
		t.Fatal(err)
	}
	overlay, count, err := WriteOverlay(root, plan)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("overlay files = %d, want 1", count)
	}
	command := exec.CommandContext(context.Background(), "go", "test", "-overlay="+overlay, "./...")
	command.Dir = root
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("overlay build: %v\n%s", err, output)
	}
}

func TestVerifySQLSynthesisPlanRejectsGeneratedMutation(t *testing.T) {
	root := t.TempDir()
	writeSQLFixture(t, filepath.Join(root, "go.mod"), "module example.com/sqlfixture\n\ngo 1.26\n")
	writeSQLFixture(t, filepath.Join(root, "query.go"), "package sqlfixture\n")
	parsed, _ := adapter.ParseExactKeySelect("SELECT id FROM users WHERE id = $1")
	synthesis, _ := adapter.SynthesizeExactKey(adapter.SynthInput{Query: parsed, KeyType: "bigint"})
	plan, err := BuildSQLSynthesisPlan(context.Background(), SQLPlanRequest{
		Workspace: root, PackageName: "sqlfixture", PackagePath: "example.com/sqlfixture",
		Output: "batchweaver_sql_gen.go", Constant: "GetBatchSQL", Operation: "get", Synthesis: synthesis,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan.Files[0].transformed = append(plan.Files[0].transformed, []byte("// mutation\n")...)
	if err := VerifySQLSynthesisPlan(plan); err == nil {
		t.Fatal("generated source mutation survived verification")
	}
}

func TestBuildSQLSynthesisPlanRejectsMutatedOrInvalidInput(t *testing.T) {
	root := t.TempDir()
	writeSQLFixture(t, filepath.Join(root, "go.mod"), "module example.com/sqlfixture\n\ngo 1.26\n")
	writeSQLFixture(t, filepath.Join(root, "query.go"), "package sqlfixture\n")
	parsed, _ := adapter.ParseExactKeySelect("SELECT id FROM users WHERE id = $1")
	synthesis, _ := adapter.SynthesizeExactKey(adapter.SynthInput{Query: parsed, KeyType: "bigint"})
	synthesis.Query = strings.ReplaceAll(synthesis.Query, "ORDER BY bw_requested.bw_ord", "")
	_, err := BuildSQLSynthesisPlan(context.Background(), SQLPlanRequest{
		Workspace: root, PackageName: "sqlfixture", PackagePath: "example.com/sqlfixture",
		Output: "batchweaver_sql_gen.go", Constant: "GetBatchSQL", Operation: "get", Synthesis: synthesis,
	})
	if err == nil || !strings.Contains(err.Error(), CodeSQLTransformPlan) {
		t.Fatalf("mutated plan error = %v", err)
	}
	_, err = BuildSQLSynthesisPlan(context.Background(), SQLPlanRequest{Workspace: root, PackageName: "bad-name"})
	if err == nil || !strings.Contains(err.Error(), CodeSQLTransformInput) {
		t.Fatalf("invalid input error = %v", err)
	}
	_, err = BuildSQLSynthesisPlan(context.Background(), SQLPlanRequest{
		Workspace: root, PackageName: "sqlfixture", PackagePath: "example.com/sqlfixture",
		Output: "query.go", Constant: "GetBatchSQL", Operation: "get",
		Synthesis: func() adapter.SynthPlan {
			valid, _ := adapter.SynthesizeExactKey(adapter.SynthInput{Query: parsed, KeyType: "bigint"})
			return valid
		}(),
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing target error = %v", err)
	}
}

func writeSQLFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
