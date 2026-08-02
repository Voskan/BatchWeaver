package jsonrpc

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// pipePair returns two connected in-memory streams.
func pipePair() (net.Conn, net.Conn) { return net.Pipe() }

func TestRequestResponse(t *testing.T) {
	a, b := pipePair()
	defer func() { _ = a.Close() }()
	defer func() { _ = b.Close() }()

	server := NewConn(b, b, func(_ context.Context, _ *Conn, req *Request) (any, *Error) {
		if req.Method != "echo" {
			return nil, NewError(CodeMethodNotFound, "no")
		}
		var s string
		_ = json.Unmarshal(req.Params, &s)
		return map[string]string{"echo": s}, nil
	})
	client := NewConn(a, a, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go server.Serve(ctx) //nolint:errcheck
	go client.Serve(ctx) //nolint:errcheck

	raw, err := client.Call(ctx, "echo", "hi")
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["echo"] != "hi" {
		t.Errorf("echo = %q, want hi", got["echo"])
	}
}

func TestMethodNotFound(t *testing.T) {
	a, b := pipePair()
	defer func() { _ = a.Close() }()
	defer func() { _ = b.Close() }()
	server := NewConn(b, b, func(_ context.Context, _ *Conn, _ *Request) (any, *Error) {
		return nil, NewError(CodeMethodNotFound, "nope")
	})
	client := NewConn(a, a, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go server.Serve(ctx) //nolint:errcheck
	go client.Serve(ctx) //nolint:errcheck
	if _, err := client.Call(ctx, "x", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestNotificationNoReply(t *testing.T) {
	a, b := pipePair()
	defer func() { _ = a.Close() }()
	defer func() { _ = b.Close() }()
	got := make(chan string, 1)
	server := NewConn(b, b, func(_ context.Context, _ *Conn, req *Request) (any, *Error) {
		got <- req.Method
		return nil, nil
	})
	client := NewConn(a, a, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go server.Serve(ctx) //nolint:errcheck
	go client.Serve(ctx) //nolint:errcheck
	if err := client.Notify(ctx, "ping", nil); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-got:
		if m != "ping" {
			t.Errorf("method = %q", m)
		}
	case <-ctx.Done():
		t.Fatal("notification not delivered")
	}
}

func TestIDMarshalRoundTrip(t *testing.T) {
	for _, id := range []ID{NewNumberID(7), NewStringID("abc")} {
		data, err := id.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		var back ID
		if err := back.UnmarshalJSON(data); err != nil {
			t.Fatal(err)
		}
		if back.String() != id.String() {
			t.Errorf("round trip %s != %s", back.String(), id.String())
		}
	}
}

func FuzzReadMessage(f *testing.F) {
	f.Add([]byte("Content-Length: 2\r\n\r\n{}"))
	f.Add([]byte("Content-Length: 0\r\n\r\n"))
	f.Add([]byte("garbage"))
	f.Add([]byte("Content-Length: 999999999999\r\n\r\n"))
	f.Fuzz(func(_ *testing.T, data []byte) {
		c := NewConn(readerOf(data), discard{}, nil)
		c.SetMaxMessageBytes(1 << 20)
		// Must never panic; an error is fine.
		_, _ = c.readMessage()
	})
}

type discard struct{}

func (discard) Write(b []byte) (int, error) { return len(b), nil }

func readerOf(b []byte) *byteReader { return &byteReader{b: b} }

type byteReader struct {
	b []byte
	i int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, errEOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

var errEOF = errorString("EOF")

type errorString string

func (e errorString) Error() string { return string(e) }
