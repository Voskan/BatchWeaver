package adapter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	yaml "github.com/goccy/go-yaml"
)

// HTTP/OpenAPI diagnostic codes (BW73xx).
const (
	CodeHTTPEndpointUndeclared = "BW7301"
	CodeHTTPCorrelationMissing = "BW7302"
	CodeHTTPAuthPartition      = "BW7303"
	CodeHTTPResponseHeaders    = "BW7304"
	CodeOpenAPIUnsafeRef       = "BW7305"
	CodeOpenAPIInvalidExt      = "BW7306"
	CodeHTTPBatchSizeExceeded  = "BW7307"
)

// maxOpenAPIBytes bounds an OpenAPI document to guard against oversized inputs
// and alias-expansion bombs.
const maxOpenAPIBytes = 8 << 20

// XBatchweaver is the versioned BatchWeaver OpenAPI vendor extension declaring a
// scalar/batch relationship. It is validated strictly.
type XBatchweaver struct {
	ScalarOperationID string `json:"scalar-operation-id" yaml:"scalar-operation-id"`
	Mode              string `json:"mode" yaml:"mode"`
	RequestItemsPath  string `json:"request-items-path" yaml:"request-items-path"`
	RequestKeyPath    string `json:"request-key-path" yaml:"request-key-path"`
	ResponseItemsPath string `json:"response-items-path" yaml:"response-items-path"`
	ResponseKeyPath   string `json:"response-key-path" yaml:"response-key-path"`
	PerItemErrorPath  string `json:"per-item-error-path" yaml:"per-item-error-path"`
	MaximumItems      int    `json:"maximum-items" yaml:"maximum-items"`
}

type openAPIOp struct {
	OperationID  string        `json:"operationId" yaml:"operationId"`
	XBatchweaver *XBatchweaver `json:"x-batchweaver" yaml:"x-batchweaver"`
}

type openAPIDoc struct {
	OpenAPI string                          `json:"openapi" yaml:"openapi"`
	Paths   map[string]map[string]openAPIOp `json:"paths" yaml:"paths"`
}

// OpenAPIBatchBinding is a discovered explicit batch endpoint.
type OpenAPIBatchBinding struct {
	Path              string
	Method            string
	ScalarOperationID string
	Binding           HTTPBinding
	Extension         XBatchweaver
}

// httpMethods is the set of recognized OpenAPI operation keys.
var httpMethods = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"patch": true, "head": true, "options": true, "trace": true,
}

// LoadOpenAPI parses an OpenAPI 3.1+ document (JSON or YAML) and returns the
// explicit batch bindings declared via the x-batchweaver extension. It never
// resolves remote references and bounds document size. It never infers batch
// semantics from endpoint names.
func LoadOpenAPI(data []byte, isYAML bool) ([]OpenAPIBatchBinding, *Rejection) {
	if len(data) > maxOpenAPIBytes {
		return nil, &Rejection{Code: CodeHTTPBatchSizeExceeded, Reason: "OpenAPI document exceeds the size limit"}
	}
	var doc openAPIDoc
	var err error
	if isYAML {
		err = yaml.Unmarshal(data, &doc)
	} else {
		err = json.Unmarshal(data, &doc)
	}
	if err != nil {
		return nil, &Rejection{Code: CodeOpenAPIInvalidExt, Reason: "cannot parse OpenAPI document: " + err.Error()}
	}
	if !strings.HasPrefix(doc.OpenAPI, "3.") {
		return nil, &Rejection{Code: CodeOpenAPIInvalidExt, Reason: fmt.Sprintf("unsupported OpenAPI version %q (want 3.x)", doc.OpenAPI)}
	}

	var out []OpenAPIBatchBinding
	paths := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, path := range paths {
		methods := make([]string, 0, len(doc.Paths[path]))
		for m := range doc.Paths[path] {
			methods = append(methods, m)
		}
		sort.Strings(methods)
		for _, m := range methods {
			if !httpMethods[strings.ToLower(m)] {
				continue
			}
			op := doc.Paths[path][m]
			if op.XBatchweaver == nil {
				continue
			}
			binding, rej := extensionToBinding(path, m, *op.XBatchweaver)
			if rej != nil {
				return nil, rej
			}
			out = append(out, OpenAPIBatchBinding{
				Path: path, Method: strings.ToUpper(m),
				ScalarOperationID: op.XBatchweaver.ScalarOperationID,
				Binding:           binding, Extension: *op.XBatchweaver,
			})
		}
	}
	return out, nil
}

// extensionToBinding validates an x-batchweaver extension and maps it to an
// HTTP binding.
func extensionToBinding(path, method string, ext XBatchweaver) (HTTPBinding, *Rejection) {
	if ext.ScalarOperationID == "" {
		return HTTPBinding{}, &Rejection{Code: CodeOpenAPIInvalidExt, Reason: "x-batchweaver requires scalar-operation-id", Node: path}
	}
	var mode EnvelopeMode
	switch ext.Mode {
	case "keyed", "":
		mode = EnvelopeKeyed
		if ext.ResponseKeyPath == "" {
			return HTTPBinding{}, &Rejection{Code: CodeHTTPCorrelationMissing, Reason: "keyed mode requires response-key-path", Node: path}
		}
	case "positional":
		mode = EnvelopePositional
	default:
		return HTTPBinding{}, &Rejection{Code: CodeOpenAPIInvalidExt, Reason: fmt.Sprintf("unknown x-batchweaver mode %q", ext.Mode), Node: path}
	}
	return HTTPBinding{Method: strings.ToUpper(method), URL: path, Mode: mode, MaxItems: ext.MaximumItems}, nil
}
