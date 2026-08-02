package gqlgen_test

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"

	batchgqlgen "github.com/Voskan/BatchWeaver/adapters/gqlgen"
	batchruntime "github.com/Voskan/BatchWeaver/runtime"
)

func TestScopeExtensionOwnsOperationScopeUntilIteratorEnds(t *testing.T) {
	engine, err := batchruntime.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = engine.Close(context.Background()) }()
	ext := batchgqlgen.ScopeExtension{Engine: engine}
	var scope *batchruntime.Scope
	handler := ext.InterceptOperation(context.Background(), func(ctx context.Context) graphql.ResponseHandler {
		var ok bool
		scope, ok = batchruntime.ScopeFromContext(ctx)
		if !ok {
			t.Fatal("operation context has no BatchWeaver scope")
		}
		var called bool
		return func(context.Context) *graphql.Response {
			if called {
				return nil
			}
			called = true
			return &graphql.Response{Data: []byte(`{"ok":true}`)}
		}
	})
	if response := handler(context.Background()); response == nil {
		t.Fatal("first response is nil")
	}
	if scope.State() != batchruntime.ScopeStateOpen {
		t.Fatalf("scope closed before response iterator ended: %s", scope.State())
	}
	if response := handler(context.Background()); response != nil {
		t.Fatal("second response is not nil")
	}
	if scope.State() != batchruntime.ScopeStateClosed {
		t.Fatalf("scope state = %s, want closed", scope.State())
	}
}

func TestScopeExtensionAttachesNormalizedFieldInfo(t *testing.T) {
	ext := batchgqlgen.ScopeExtension{}
	field := &ast.Field{Name: "user", Alias: "viewer", SelectionSet: ast.SelectionSet{
		&ast.Field{Name: "name"},
		&ast.Field{Name: "id"},
	}}
	ctx := graphql.WithFieldContext(context.Background(), &graphql.FieldContext{
		Object: "Query",
		Field:  graphql.CollectedField{Field: field, Selections: field.SelectionSet},
	})
	_, err := ext.InterceptField(ctx, func(ctx context.Context) (any, error) {
		info, ok := batchgqlgen.FieldInfoFromContext(ctx)
		if !ok {
			t.Fatal("field info missing")
		}
		if info.Object != "Query" || info.Field != "user" || info.SelectionDigest == "" || info.Partition == "" {
			t.Fatalf("unexpected info: %+v", info)
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestScopeExtensionRequiresEngine(t *testing.T) {
	ext := batchgqlgen.ScopeExtension{}
	if err := ext.Validate(nil); !errors.Is(err, batchgqlgen.ErrEngineRequired) {
		t.Fatalf("Validate error = %v", err)
	}
	response := ext.InterceptOperation(context.Background(), func(context.Context) graphql.ResponseHandler {
		t.Fatal("next called without engine")
		return nil
	})(context.Background())
	if response == nil || len(response.Errors) != 1 {
		t.Fatalf("unexpected error response: %+v", response)
	}
}

func TestScopeExtensionIsolatesAndDoesNotCloseParentScope(t *testing.T) {
	engine, err := batchruntime.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = engine.Close(context.Background()) }()
	parentCtx, parent, err := engine.NewScope(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = parent.Close(context.Background()) }()
	ext := batchgqlgen.ScopeExtension{Engine: engine}
	var child *batchruntime.Scope
	handler := ext.InterceptOperation(parentCtx, func(ctx context.Context) graphql.ResponseHandler {
		child, _ = batchruntime.ScopeFromContext(ctx)
		return func(context.Context) *graphql.Response { return nil }
	})
	_ = handler(context.Background())
	if child == nil || child.ID() == parent.ID() || child.State() != batchruntime.ScopeStateClosed {
		t.Fatalf("child scope was not isolated and closed: parent=%v child=%v", parent.ID(), child)
	}
	if parent.State() != batchruntime.ScopeStateOpen {
		t.Fatalf("parent scope was closed by operation: %s", parent.State())
	}
}

func TestScopeExtensionPropagatesOperationCancellation(t *testing.T) {
	engine, err := batchruntime.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = engine.Close(context.Background()) }()
	operationCtx, cancel := context.WithCancel(context.Background())
	var scopedCtx context.Context
	handler := (batchgqlgen.ScopeExtension{Engine: engine}).InterceptOperation(operationCtx, func(ctx context.Context) graphql.ResponseHandler {
		scopedCtx = ctx
		return func(context.Context) *graphql.Response { return nil }
	})
	cancel()
	if !errors.Is(scopedCtx.Err(), context.Canceled) {
		t.Fatalf("scoped context error = %v, want context.Canceled", scopedCtx.Err())
	}
	_ = handler(context.Background())
}
