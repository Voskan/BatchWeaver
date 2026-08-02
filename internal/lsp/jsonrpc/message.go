package jsonrpc

import (
	"encoding/json"
	"fmt"
)

// Standard JSON-RPC 2.0 error codes plus the LSP-specific request-canceled and
// content-modified codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	// CodeRequestCancelled is the LSP code for a canceled request.
	CodeRequestCancelled = -32800
	// CodeContentModified is the LSP code for a result invalidated by newer content.
	CodeContentModified = -32801
)

// ID is a JSON-RPC request identifier, which may be a number or a string. The
// zero value is an absent ID (used for notifications).
type ID struct {
	num    int64
	str    string
	isStr  bool
	hasNum bool
}

// NewNumberID returns a numeric ID.
func NewNumberID(n int64) ID { return ID{num: n, hasNum: true} }

// NewStringID returns a string ID.
func NewStringID(s string) ID { return ID{str: s, isStr: true} }

// IsValid reports whether the ID is present.
func (id ID) IsValid() bool { return id.hasNum || id.isStr }

// String returns a stable string form used as a map key and in logs.
func (id ID) String() string {
	switch {
	case id.isStr:
		return "s:" + id.str
	case id.hasNum:
		return fmt.Sprintf("n:%d", id.num)
	default:
		return "<nil>"
	}
}

// MarshalJSON encodes the ID as a number or string, or null when absent.
func (id ID) MarshalJSON() ([]byte, error) {
	switch {
	case id.isStr:
		return json.Marshal(id.str)
	case id.hasNum:
		return json.Marshal(id.num)
	default:
		return []byte("null"), nil
	}
}

// UnmarshalJSON decodes a number or string ID.
func (id *ID) UnmarshalJSON(data []byte) error {
	*id = ID{}
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		if err := json.Unmarshal(data, &id.str); err != nil {
			return err
		}
		id.isStr = true
		return nil
	}
	if err := json.Unmarshal(data, &id.num); err != nil {
		return err
	}
	id.hasNum = true
	return nil
}

// wireMessage is the union of request, response, and notification as it appears
// on the wire. A message with an ID and a method is a request; with an ID and no
// method it is a response; with a method and no ID it is a notification.
type wireMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *ID             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Request is an incoming JSON-RPC request or notification. Notifications have an
// invalid (absent) ID.
type Request struct {
	ID     ID
	Method string
	Params json.RawMessage
}

// IsNotification reports whether the request is a notification (no ID).
func (r *Request) IsNotification() bool { return !r.ID.IsValid() }

// Error is a JSON-RPC error object.
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string { return fmt.Sprintf("jsonrpc: %d: %s", e.Code, e.Message) }

// NewError returns a JSON-RPC error with the given code and message.
func NewError(code int, msg string) *Error { return &Error{Code: code, Message: msg} }

// Errorf returns a JSON-RPC error with a formatted message.
func Errorf(code int, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}
