package jsonrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultMaxMessageBytes bounds a single incoming message to defend against
// hostile or runaway payloads.
const DefaultMaxMessageBytes = 64 << 20 // 64 MiB

// Handler processes an incoming request or notification. For a request it
// returns a result to marshal, or a non-nil *Error. For a notification the
// return values are ignored. The context is canceled if the peer cancels the
// request or the connection closes.
type Handler func(ctx context.Context, conn *Conn, req *Request) (any, *Error)

// Conn is a bidirectional JSON-RPC 2.0 connection over an LSP-framed stream. It
// is safe for concurrent use: writes are serialized, incoming requests are
// handled concurrently, and outgoing calls are matched to responses by ID.
type Conn struct {
	r        *bufio.Reader
	w        io.Writer
	maxBytes int64

	writeMu sync.Mutex

	handler Handler

	seq atomic.Int64

	pendingMu sync.Mutex
	pending   map[string]chan *response // outgoing calls awaiting a response

	cancelMu sync.Mutex
	cancels  map[string]context.CancelFunc // in-flight incoming requests

	closeOnce sync.Once
	closed    chan struct{}
	closeErr  error

	// inflight tracks incoming request/notification handler goroutines so Serve
	// can flush their replies before returning on EOF.
	inflight sync.WaitGroup
}

// drainTimeout bounds how long Serve waits for in-flight handlers on shutdown.
const drainTimeout = 5 * 1000 * 1000 * 1000 // 5s in nanoseconds

type response struct {
	result json.RawMessage
	err    *Error
}

// NewConn returns a connection reading from r and writing to w, dispatching
// incoming requests to handler (which may be nil for a pure client).
func NewConn(r io.Reader, w io.Writer, handler Handler) *Conn {
	return &Conn{
		r:        bufio.NewReader(r),
		w:        w,
		maxBytes: DefaultMaxMessageBytes,
		handler:  handler,
		pending:  make(map[string]chan *response),
		cancels:  make(map[string]context.CancelFunc),
		closed:   make(chan struct{}),
	}
}

// SetMaxMessageBytes overrides the incoming message size bound.
func (c *Conn) SetMaxMessageBytes(n int64) {
	if n > 0 {
		c.maxBytes = n
	}
}

// Done returns a channel closed when the connection stops serving.
func (c *Conn) Done() <-chan struct{} { return c.closed }

// Err returns the error that stopped the connection, if any.
func (c *Conn) Err() error { return c.closeErr }

// Serve reads and dispatches messages until EOF, a fatal read error, or ctx
// cancellation. It returns the terminating error (nil on clean EOF).
func (c *Conn) Serve(ctx context.Context) error {
	defer c.shutdown(nil)
	for {
		if err := ctx.Err(); err != nil {
			c.shutdown(err)
			return err
		}
		msg, err := c.readMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.drain()
				c.shutdown(nil)
				return nil
			}
			c.shutdown(err)
			return err
		}
		c.dispatch(ctx, msg)
	}
}

// dispatch routes a decoded message to a response waiter or the handler.
func (c *Conn) dispatch(ctx context.Context, msg *wireMessage) {
	// Response to one of our outgoing calls.
	if msg.Method == "" && msg.ID != nil {
		c.deliverResponse(*msg.ID, &response{result: msg.Result, err: msg.Error})
		return
	}
	req := &Request{Method: msg.Method, Params: msg.Params}
	if msg.ID != nil {
		req.ID = *msg.ID
	}
	// Built-in cancellation notification.
	if req.Method == "$/cancelRequest" {
		c.handleCancel(req.Params)
		return
	}
	if req.IsNotification() {
		if c.handler != nil {
			c.inflight.Add(1)
			go func() {
				defer c.inflight.Done()
				_, _ = c.handler(ctx, c, req) // notification result ignored
			}()
		}
		return
	}
	c.handleRequest(ctx, req)
}

// handleRequest runs the handler for an incoming request on its own goroutine
// with a cancellable context registered by ID.
func (c *Conn) handleRequest(parent context.Context, req *Request) {
	if c.handler == nil {
		_ = c.Reply(req.ID, nil, NewError(CodeMethodNotFound, "no handler"))
		return
	}
	ctx, cancel := context.WithCancel(parent)
	key := req.ID.String()
	c.cancelMu.Lock()
	c.cancels[key] = cancel
	c.cancelMu.Unlock()
	c.inflight.Add(1)
	go func() {
		defer c.inflight.Done()
		defer func() {
			c.cancelMu.Lock()
			delete(c.cancels, key)
			c.cancelMu.Unlock()
			cancel()
		}()
		result, rpcErr := c.handler(ctx, c, req)
		if ctx.Err() != nil && rpcErr == nil {
			rpcErr = NewError(CodeRequestCancelled, "request canceled")
		}
		_ = c.Reply(req.ID, result, rpcErr)
	}()
}

// handleCancel cancels an in-flight incoming request by ID.
func (c *Conn) handleCancel(params json.RawMessage) {
	var p struct {
		ID ID `json:"id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}
	c.cancelMu.Lock()
	if cancel, ok := c.cancels[p.ID.String()]; ok {
		cancel()
	}
	c.cancelMu.Unlock()
}

// Call sends a request and waits for the matching response or ctx cancellation.
func (c *Conn) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := NewNumberID(c.seq.Add(1))
	ch := make(chan *response, 1)
	key := id.String()
	c.pendingMu.Lock()
	c.pending[key] = ch
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, key)
		c.pendingMu.Unlock()
	}()

	if err := c.write(&wireMessage{JSONRPC: "2.0", ID: &id, Method: method, Params: mustRaw(params)}); err != nil {
		return nil, err
	}
	select {
	case resp := <-ch:
		if resp.err != nil {
			return nil, resp.err
		}
		return resp.result, nil
	case <-ctx.Done():
		// Best-effort cancellation notification to the peer.
		_ = c.Notify(context.Background(), "$/cancelRequest", map[string]any{"id": id})
		return nil, ctx.Err()
	case <-c.closed:
		return nil, fmt.Errorf("jsonrpc: connection closed: %w", c.closeErr)
	}
}

// Notify sends a notification (no response expected).
func (c *Conn) Notify(_ context.Context, method string, params any) error {
	return c.write(&wireMessage{JSONRPC: "2.0", Method: method, Params: mustRaw(params)})
}

// Reply writes a response for an incoming request.
func (c *Conn) Reply(id ID, result any, rpcErr *Error) error {
	msg := &wireMessage{JSONRPC: "2.0", ID: &id}
	if rpcErr != nil {
		msg.Error = rpcErr
	} else {
		msg.Result = mustRaw(result)
	}
	return c.write(msg)
}

// deliverResponse hands a response to the waiting caller, if any.
func (c *Conn) deliverResponse(id ID, resp *response) {
	c.pendingMu.Lock()
	ch, ok := c.pending[id.String()]
	c.pendingMu.Unlock()
	if ok {
		ch <- resp
	}
}

// write serializes and frames a message with an ordered, exclusive writer.
func (c *Conn) write(msg *wireMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := io.WriteString(c.w, "Content-Length: "+strconv.Itoa(len(body))+"\r\n\r\n"); err != nil {
		return err
	}
	_, err = c.w.Write(body)
	return err
}

// readMessage reads one framed message, enforcing the size bound.
func (c *Conn) readMessage() (*wireMessage, error) {
	var contentLength int64 = -1
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break // end of headers
		}
		if k, v, ok := strings.Cut(trimmed, ":"); ok && strings.EqualFold(strings.TrimSpace(k), "Content-Length") {
			n, perr := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if perr != nil {
				return nil, fmt.Errorf("jsonrpc: invalid Content-Length: %w", perr)
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, errors.New("jsonrpc: missing Content-Length header")
	}
	if contentLength > c.maxBytes {
		return nil, fmt.Errorf("jsonrpc: message of %d bytes exceeds bound %d", contentLength, c.maxBytes)
	}
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(c.r, buf); err != nil {
		return nil, err
	}
	var msg wireMessage
	if err := json.Unmarshal(buf, &msg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParse, err)
	}
	return &msg, nil
}

// ErrParse indicates a message body that is not valid JSON.
var ErrParse = errors.New("jsonrpc: parse error")

// drain waits, up to drainTimeout, for in-flight incoming handlers to finish so
// their replies are flushed before the connection closes.
func (c *Conn) drain() {
	done := make(chan struct{})
	go func() {
		c.inflight.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(drainTimeout):
	}
}

// shutdown records the terminating error once and wakes waiters.
func (c *Conn) shutdown(err error) {
	c.closeOnce.Do(func() {
		c.closeErr = err
		close(c.closed)
	})
}

// mustRaw marshals params to raw JSON, returning nil for a nil params.
func mustRaw(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	if raw, ok := v.(json.RawMessage); ok {
		return raw
	}
	data, err := json.Marshal(v)
	if err != nil {
		// A marshal failure here is a programming error; encode null so the peer
		// still receives a well-formed message rather than a truncated frame.
		return json.RawMessage("null")
	}
	return data
}
