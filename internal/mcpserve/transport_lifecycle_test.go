package mcpserve

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// freeLoopbackAddr returns a currently unused 127.0.0.1 address for a server
// under test to bind.
func freeLoopbackAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving a loopback port: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("releasing the loopback port: %v", err)
	}
	return addr
}

// waitForHTTP polls url until the server answers, so a test does not race the
// listener goroutine.
func waitForHTTP(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		resp, err := http.Get(url) //nolint:gosec,noctx // loopback test server
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("server at %s never came up: %v", url, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestServeHTTPShutdownWithOpenStream is the regression test for shutdown
// stalling on a held notification stream: with a client attached to the GET
// (SSE) stream, cancelling the serve context must return promptly and cleanly
// (context.Canceled, which the serve command treats as a normal stop) instead
// of waiting out the shutdown timeout and reporting DeadlineExceeded.
func TestServeHTTPShutdownWithOpenStream(t *testing.T) {
	t.Parallel()
	srv := New(WithProjectRoot(t.TempDir()))
	addr := freeLoopbackAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ServeHTTP(ctx, addr, nil) }()

	url := "http://" + addr + mcpEndpointPath
	waitForHTTP(t, url)
	client := &http.Client{Timeout: 10 * time.Second}
	session := mcpInitialize(t, client, url)

	req, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx // closed via resp.Body below
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Mcp-Session-Id", session)
	streamClient := &http.Client{} // no timeout: the stream is meant to stay open
	resp, err := streamClient.Do(req)
	if err != nil {
		t.Fatalf("opening the GET stream: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET stream status = %d, want 200", resp.StatusCode)
	}

	start := time.Now()
	cancel()
	select {
	case err := <-serveErr:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("ServeHTTP returned %v, want context.Canceled", err)
		}
		if elapsed := time.Since(start); elapsed >= httpShutdownTimeout {
			t.Errorf("shutdown took %s; the open stream held it for the full timeout", elapsed)
		}
	case <-time.After(httpShutdownTimeout + 5*time.Second):
		t.Fatal("ServeHTTP did not return after cancellation")
	}
}

// TestHTTPHandlerCapsRequestBody verifies the handler stack bounds request
// bodies, so an oversized POST cannot be read fully into memory.
func TestHTTPHandlerCapsRequestBody(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"at limit", maxHTTPRequestBytes, false},
		{"over limit", maxHTTPRequestBytes + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var readErr error
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, readErr = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodPost, mcpEndpointPath, bytes.NewReader(make([]byte, tt.size)))
			req.Host = "127.0.0.1:8765"
			httpHandler(inner, nil).ServeHTTP(httptest.NewRecorder(), req)

			var maxErr *http.MaxBytesError
			if got := errors.As(readErr, &maxErr); got != tt.wantErr {
				t.Errorf("body of %d bytes: read error = %v, want MaxBytesError: %v", tt.size, readErr, tt.wantErr)
			}
		})
	}
}

// TestStreamableHTTPReclaimsIdleSessions is the regression test for sessions
// accumulating forever: a client that initializes and then abandons its session
// without a DELETE must have it reclaimed once it has been idle past the TTL.
func TestStreamableHTTPReclaimsIdleSessions(t *testing.T) {
	t.Parallel()
	srv := New(WithProjectRoot(t.TempDir()))
	streamable := newStreamableHTTP(srv.MCPServer(), 500*time.Millisecond)
	defer func() { _ = streamable.Shutdown(context.Background()) }()
	mux := http.NewServeMux()
	mux.Handle(mcpEndpointPath, streamable)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	session := mcpInitialize(t, ts.Client(), ts.URL+mcpEndpointPath)
	notify := func() error {
		return srv.MCPServer().SendNotificationToSpecificClient(session, "notifications/message", nil)
	}
	if err := notify(); errors.Is(err, server.ErrSessionNotFound) {
		t.Fatal("session was not registered after initialize")
	}

	deadline := time.Now().Add(10 * time.Second)
	for !errors.Is(notify(), server.ErrSessionNotFound) {
		if time.Now().After(deadline) {
			t.Fatal("abandoned session was never reclaimed")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
