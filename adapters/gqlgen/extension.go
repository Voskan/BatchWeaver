package gqlgen

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"

	batchruntime "github.com/Voskan/BatchWeaver/runtime"
)

// ErrEngineRequired is returned when a ScopeExtension has no runtime engine.
var ErrEngineRequired = errors.New("batchweaver gqlgen: runtime engine is required")

// FieldInfo describes the gqlgen field currently being resolved. Selection is
// alias-independent and contains no argument values or response payloads.
type FieldInfo struct {
	Object          string
	Field           string
	Path            string
	SelectionDigest string
	Partition       string
}

type fieldInfoKey struct{}

// FieldInfoFromContext returns field metadata attached by ScopeExtension.
func FieldInfoFromContext(ctx context.Context) (FieldInfo, bool) {
	info, ok := ctx.Value(fieldInfoKey{}).(FieldInfo)
	return info, ok
}

// PartitionFromContext returns the conservative field partition token. It can
// be composed with application tenant/authorization partitions in a runtime
// binding. An empty string means the resolver is outside this extension.
func PartitionFromContext(ctx context.Context) string {
	info, ok := FieldInfoFromContext(ctx)
	if !ok {
		return ""
	}
	return info.Partition
}

// ScopeExtension creates one runtime scope per gqlgen operation and attaches
// field metadata through a public FieldInterceptor.
type ScopeExtension struct {
	Engine       *batchruntime.Engine
	OnCloseError func(error)
}

// ExtensionName implements graphql.HandlerExtension.
func (ScopeExtension) ExtensionName() string { return "BatchWeaver" }

// Validate implements graphql.HandlerExtension.
func (e ScopeExtension) Validate(graphql.ExecutableSchema) error {
	if e.Engine == nil {
		return ErrEngineRequired
	}
	return nil
}

// InterceptOperation implements graphql.OperationInterceptor. The returned
// response iterator owns the scope and closes it exactly once after the final
// response, including subscriptions with multiple responses.
func (e ScopeExtension) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	if e.Engine == nil {
		return graphql.OneShot(graphql.ErrorResponse(ctx, "%s", ErrEngineRequired))
	}
	scopedCtx, scope, err := e.Engine.NewScope(ctx)
	if err != nil {
		return graphql.OneShot(graphql.ErrorResponse(ctx, "BatchWeaver scope: %v", err))
	}
	var once sync.Once
	closeScope := func() {
		once.Do(func() {
			if closeErr := scope.Close(context.WithoutCancel(scopedCtx)); closeErr != nil && e.OnCloseError != nil {
				e.OnCloseError(closeErr)
			}
		})
	}
	var inner graphql.ResponseHandler
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				closeScope()
				panic(recovered)
			}
		}()
		inner = next(scopedCtx)
	}()
	if inner == nil {
		closeScope()
		return func(context.Context) *graphql.Response { return nil }
	}
	return func(context.Context) *graphql.Response {
		response := inner(scopedCtx)
		if response == nil {
			closeScope()
		}
		return response
	}
}

// InterceptField implements graphql.FieldInterceptor using public gqlgen field
// context data. It does not change resolver results or GraphQL error semantics.
func (ScopeExtension) InterceptField(ctx context.Context, next graphql.Resolver) (any, error) {
	field := graphql.GetFieldContext(ctx)
	if field == nil || field.Field.Field == nil {
		return next(ctx)
	}
	selection := digestSelection(field.Field.Selections)
	info := FieldInfo{
		Object:          field.Object,
		Field:           field.Field.Name,
		Path:            field.Path().String(),
		SelectionDigest: selection,
		Partition:       field.Object + ":" + field.Field.Name + ":" + selection,
	}
	return next(context.WithValue(ctx, fieldInfoKey{}, info))
}

func digestSelection(selection ast.SelectionSet) string {
	var tokens []string
	collectSelection(selection, "", &tokens)
	sort.Strings(tokens)
	hash := sha256.Sum256([]byte(strings.Join(tokens, "\x00")))
	return "sha256:" + hex.EncodeToString(hash[:8])
}

func collectSelection(selection ast.SelectionSet, prefix string, out *[]string) {
	for _, selected := range selection {
		switch node := selected.(type) {
		case *ast.Field:
			path := node.Name
			if prefix != "" {
				path = prefix + "." + path
			}
			*out = append(*out, path)
			collectSelection(node.SelectionSet, path, out)
		case *ast.InlineFragment:
			collectSelection(node.SelectionSet, prefix+"<"+node.TypeCondition+">", out)
		case *ast.FragmentSpread:
			*out = append(*out, prefix+"..."+node.Name)
		}
	}
}

var _ graphql.HandlerExtension = ScopeExtension{}
var _ graphql.OperationInterceptor = ScopeExtension{}
var _ graphql.FieldInterceptor = ScopeExtension{}
