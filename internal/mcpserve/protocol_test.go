package mcpserve

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

// TestServeStdioInitializeHandshake drives the real stdio transport over an
// io.Pipe (the same listenStdio code path production uses), performs an MCP
// initialize handshake, and asserts the negotiated protocol version and the
// advertised capabilities. It is intentionally UNTAGGED so it runs by default.
func TestServeStdioInitializeHandshake(t *testing.T) {
	t.Parallel()

	// Wire two pipes: srvIn carries client->server, srvOut carries server->client.
	srvInR, srvInW := io.Pipe()
	srvOutR, srvOutW := io.Pipe()

	srv := New(WithProjectRoot(t.TempDir()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.listenStdio(ctx, srvInR, srvOutW) }()

	// Send an initialize request requesting protocol 2025-11-25.
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "e2e-test-client",
				"version": "0.0.1",
			},
		},
	}
	line, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshaling initialize request: %v", err)
	}
	go func() {
		// Writing to an io.Pipe blocks until the server reads; do it off the
		// main goroutine so we can concurrently read the response.
		_, _ = srvInW.Write(append(line, '\n'))
	}()

	respLine := readLineWithTimeout(t, srvOutR, 5*time.Second)

	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			ProtocolVersion string                     `json:"protocolVersion"`
			Capabilities    map[string]json.RawMessage `json:"capabilities"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respLine, &resp); err != nil {
		t.Fatalf("unmarshaling initialize response %q: %v", respLine, err)
	}

	if resp.Error != nil {
		t.Fatalf("initialize returned JSON-RPC error: %+v", resp.Error)
	}
	if resp.Result.ProtocolVersion != "2025-11-25" {
		t.Errorf("protocolVersion = %q, want %q", resp.Result.ProtocolVersion, "2025-11-25")
	}
	if _, ok := resp.Result.Capabilities["tools"]; !ok {
		t.Errorf("expected tools capability advertised, got caps: %v", capKeys(resp.Result.Capabilities))
	}
	if _, ok := resp.Result.Capabilities["logging"]; !ok {
		t.Errorf("expected logging capability advertised, got caps: %v", capKeys(resp.Result.Capabilities))
	}
	if resp.Result.ServerInfo.Name == "" {
		t.Errorf("expected non-empty serverInfo.name")
	}

	// Clean shutdown: cancelling and closing the input lets Listen return.
	cancel()
	_ = srvInW.Close()
	_ = srvOutW.Close()
	select {
	case <-serveErr:
	case <-time.After(5 * time.Second):
		t.Fatal("listenStdio did not return after shutdown")
	}
}

// readLineWithTimeout reads a single newline-terminated line from r, failing the
// test if it does not arrive within d.
func readLineWithTimeout(t *testing.T, r io.Reader, d time.Duration) []byte {
	t.Helper()
	type result struct {
		line []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := bufio.NewReader(r).ReadBytes('\n')
		ch <- result{line, err}
	}()
	select {
	case res := <-ch:
		if res.err != nil && len(res.line) == 0 {
			t.Fatalf("reading response: %v", res.err)
		}
		return res.line
	case <-time.After(d):
		t.Fatal("timed out waiting for server response")
		return nil
	}
}

func capKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
