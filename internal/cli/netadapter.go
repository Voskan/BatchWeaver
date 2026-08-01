package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	batchweaver "github.com/Voskan/BatchWeaver"
	"github.com/Voskan/BatchWeaver/internal/adapter"
)

// newGraphQLCommand returns the "graphql" command.
func newGraphQLCommand() *Command {
	return &Command{
		Name:    "graphql",
		Summary: "Inspect and graph GraphQL resolver-wave batching",
		Usage:   "graphql inspect --operation-file=f | graphql graph --operation-file=f [--format=dot|text]",
		Run:     runGraphQL,
	}
}

func runGraphQL(_ context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "graphql requires a subcommand: inspect or graph"}
	}
	sub, rest := args[0], args[1:]
	fs := flag.NewFlagSet("graphql "+sub, flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("operation-file", "", "path to a .graphql operation file")
	opStr := fs.String("operation", "", "inline GraphQL operation text")
	format := fs.String("format", "text", "graph format: text or dot")
	if err := fs.Parse(rest); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	src := *opStr
	if src == "" && *file != "" {
		b, err := os.ReadFile(*file)
		if err != nil {
			return &CommandError{Code: ExitError, Message: err.Error()}
		}
		src = string(b)
	}
	if src == "" {
		return &CommandError{Code: ExitUsage, Message: "provide --operation-file or --operation"}
	}
	doc, rej := adapter.ParseGraphQL(src)
	if rej != nil {
		fmt.Fprintf(app.Stdout(), "GraphQL parse rejected\n\nDiagnostic:\n  %s\n\nReason:\n  %s\n", rej.Code, rej.Reason)
		return &CommandError{Code: ExitConfigInvalid}
	}
	if len(doc.Operations) == 0 {
		return &CommandError{Code: ExitConfigInvalid, Message: "no operations found"}
	}
	op := doc.Operations[0]
	waves := adapter.ResolverWaves(doc, op)
	w := app.Stdout()
	switch sub {
	case "inspect":
		name := op.Name
		if name == "" {
			name = "(anonymous)"
		}
		fmt.Fprintf(w, "GraphQL operation:\n  %s\n\n", name)
		fmt.Fprint(w, "Execution scope:\n  one BatchWeaver scope per operation\n\n")
		fmt.Fprintln(w, "Resolver waves:")
		for _, wv := range waves {
			fmt.Fprintf(w, "  wave %d:\n", wv.Depth)
			for _, f := range wv.Fields {
				fmt.Fprintf(w, "    %s\n", f)
			}
		}
		return nil
	case "graph":
		if *format == "dot" {
			fmt.Fprintf(w, "digraph gql {\n  label=%q;\n  rankdir=LR;\n", opName(op))
			for _, wv := range waves {
				for _, f := range wv.Fields {
					fmt.Fprintf(w, "  %q [label=%q];\n", f, fmt.Sprintf("%s\\nwave %d", f, wv.Depth))
				}
			}
			fmt.Fprintln(w, "}")
			return nil
		}
		for _, wv := range waves {
			fmt.Fprintf(w, "wave %d: %s\n", wv.Depth, strings.Join(wv.Fields, ", "))
		}
		return nil
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown graphql subcommand %q", sub)}
	}
}

func opName(op adapter.GraphQLOperation) string {
	if op.Name == "" {
		return "(anonymous)"
	}
	return op.Name
}

// newOpenAPICommand returns the "openapi" command.
func newOpenAPICommand() *Command {
	return &Command{
		Name:    "openapi",
		Summary: "Validate and inspect OpenAPI batch bindings",
		Usage:   "openapi validate --file=f | openapi inspect --file=f",
		Run:     runOpenAPI,
	}
}

func runOpenAPI(_ context.Context, app *App, args []string) error {
	if len(args) == 0 {
		return &CommandError{Code: ExitUsage, Message: "openapi requires a subcommand: validate or inspect"}
	}
	sub, rest := args[0], args[1:]
	fs := flag.NewFlagSet("openapi "+sub, flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	file := fs.String("file", "", "path to an OpenAPI document (.json or .yaml)")
	if err := fs.Parse(rest); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	if *file == "" {
		return &CommandError{Code: ExitUsage, Message: "openapi requires --file"}
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		return &CommandError{Code: ExitError, Message: err.Error()}
	}
	isYAML := strings.HasSuffix(*file, ".yaml") || strings.HasSuffix(*file, ".yml")
	bindings, rej := adapter.LoadOpenAPI(data, isYAML)
	w := app.Stdout()
	if rej != nil {
		fmt.Fprintf(w, "OpenAPI rejected\n\nDiagnostic:\n  %s\n\nReason:\n  %s\n", rej.Code, rej.Reason)
		return &CommandError{Code: ExitConfigInvalid}
	}
	switch sub {
	case "validate":
		fmt.Fprintf(w, "OpenAPI document is valid; %d batch binding(s) discovered.\n", len(bindings))
		return nil
	case "inspect":
		if len(bindings) == 0 {
			fmt.Fprintln(w, "No x-batchweaver batch bindings declared.")
			return nil
		}
		for _, b := range bindings {
			fmt.Fprintf(w, "\nScalar operation:\n  %s\nBatch endpoint:\n  %s %s\nMode:\n  %s\nMax items:\n  %d\n",
				b.ScalarOperationID, b.Binding.Method, b.Binding.URL, b.Binding.Mode, b.Binding.MaxItems)
		}
		return nil
	default:
		return &CommandError{Code: ExitUsage, Message: fmt.Sprintf("unknown openapi subcommand %q", sub)}
	}
}

// newHTTPCommand returns the "http" command.
func newHTTPCommand() *Command {
	return &Command{
		Name:    "http",
		Summary: "Inspect and verify HTTP batch endpoints",
		Usage:   "http verify",
		Run:     runHTTP,
	}
}

func runHTTP(ctx context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "verify" {
		return &CommandError{Code: ExitUsage, Message: "usage: http verify"}
	}
	// Hermetic demonstration: a local batch server plus the HTTP provider,
	// verified against a scalar reference.
	data := map[int]string{1: "alice", 2: "bob", 3: "carol"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			Items []struct {
				RequestID string `json:"request_id"`
				Key       int    `json:"key"`
			} `json:"items"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &env)
		type ri struct {
			RequestID string  `json:"request_id"`
			Value     *string `json:"value"`
			Error     string  `json:"error"`
		}
		var out struct {
			Items []ri `json:"items"`
		}
		for _, it := range env.Items {
			if v, ok := data[it.Key]; ok {
				vv := v
				out.Items = append(out.Items, ri{RequestID: it.RequestID, Value: &vv})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer srv.Close()

	prov := adapter.HTTPProvider[int, string]{
		Client:  srv.Client(),
		Binding: adapter.HTTPBinding{Method: http.MethodPost, URL: srv.URL, Mode: adapter.EnvelopeKeyed, MaxItems: 100},
	}
	scalar := func(_ context.Context, k int) (string, error) {
		if v, ok := data[k]; ok {
			return v, nil
		}
		return "", adapter.ErrHTTPMissingItem
	}
	batch := func(ctx context.Context, keys []int) ([]batchweaver.Outcome[string], error) {
		items := make([]batchweaver.BatchItem[int], len(keys))
		for i, k := range keys {
			items[i] = batchweaver.NewBatchItem(batchweaver.RequestID(i+1), k)
		}
		if len(items) == 0 {
			return nil, nil
		}
		resp, err := prov.Execute(ctx, batchweaver.MustNewBatchRequest(items))
		if err != nil {
			return nil, err
		}
		return resp.Outcomes(), nil
	}
	vc := adapter.VerifyReadOnly(ctx, "users.get", "http/openapi", scalar, batch,
		func(a, b string) bool { return a == b },
		[]adapter.VerifyCase[int]{
			{Name: "unique keys", Keys: []int{1, 2, 3}},
			{Name: "duplicate keys", Keys: []int{1, 1}},
			{Name: "missing key", Keys: []int{1, 9}},
			{Name: "one key", Keys: []int{2}},
		})
	w := app.Stdout()
	fmt.Fprintln(w, "HTTP batch contract verification (hermetic demonstration)")
	fmt.Fprintf(w, "\nAdapter:\n  http/openapi\n\nCases:\n")
	for _, c := range vc.Cases {
		status := "PASS"
		if !c.Passed {
			status = "FAIL: " + c.Detail
		}
		fmt.Fprintf(w, "  %s: %s\n", c.Name, status)
	}
	fmt.Fprintf(w, "\nContract digest:\n  %s\n", vc.Digest)
	if !vc.Passed {
		return &CommandError{Code: ExitConfigInvalid}
	}
	return nil
}

// newGRPCCommand returns the "grpc" command.
func newGRPCCommand() *Command {
	return &Command{
		Name:    "grpc",
		Summary: "Inspect gRPC batch bindings and metadata policy",
		Usage:   "grpc inspect [--scalar m --batch m --key f --requests f --response-mode keyed --response-key f]",
		Run:     runGRPC,
	}
}

func runGRPC(_ context.Context, app *App, args []string) error {
	if len(args) == 0 || args[0] != "inspect" {
		return &CommandError{Code: ExitUsage, Message: "usage: grpc inspect [flags]"}
	}
	fs := flag.NewFlagSet("grpc inspect", flag.ContinueOnError)
	fs.SetOutput(app.Stderr())
	scalar := fs.String("scalar", "", "scalar method")
	batch := fs.String("batch", "", "batch method")
	key := fs.String("key", "", "request key field")
	requests := fs.String("requests", "requests", "batch requests field")
	mode := fs.String("response-mode", "keyed", "response mode: keyed or positional")
	respKey := fs.String("response-key", "", "response key field (keyed mode)")
	if err := fs.Parse(args[1:]); err != nil {
		return &CommandError{Code: ExitUsage}
	}
	w := app.Stdout()
	if *scalar != "" || *batch != "" {
		b := adapter.GRPCBinding{
			ScalarMethod: *scalar, BatchMethod: *batch, RequestKey: *key,
			BatchRequestsField: *requests, ResponseMode: adapter.GRPCResponseMode(*mode), ResponseKey: *respKey,
		}
		if rej := b.Validate(); rej != nil {
			fmt.Fprintf(w, "gRPC binding rejected\n\nDiagnostic:\n  %s\n\nReason:\n  %s\n", rej.Code, rej.Reason)
			return &CommandError{Code: ExitConfigInvalid}
		}
		fmt.Fprintf(w, "Scalar RPC:\n  %s\nBatch RPC:\n  %s\nRequest key:\n  %s\nResponse mode:\n  %s\n",
			b.ScalarMethod, b.BatchMethod, b.RequestKey, b.ResponseMode)
	}
	fmt.Fprintln(w, "\nMetadata partition policy (defaults):")
	for _, k := range []string{"authorization", "x-tenant-id", "traceparent", "content-type", "x-custom-header"} {
		fmt.Fprintf(w, "  %-18s %s\n", k+":", adapter.ClassifyMetadata(k))
	}
	return nil
}
