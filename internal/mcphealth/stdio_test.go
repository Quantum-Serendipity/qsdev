package mcphealth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// helperModeEnv selects the fake MCP server behaviour when the test binary is
// re-executed as a stdio server by fakeServerConfig.
const helperModeEnv = "QSDEV_MCPHEALTH_FAKE_SERVER"

// TestHelperFakeMCPServer is not a real test: when helperModeEnv is set, the
// test binary acts as a stdio MCP server so the probe can be exercised against
// controlled, cross-platform server behaviour.
func TestHelperFakeMCPServer(t *testing.T) {
	mode := os.Getenv(helperModeEnv)
	if mode == "" {
		return
	}
	runFakeServer(mode)
	os.Exit(0)
}

// fakeServerConfig returns a ServerConfig that launches this test binary as a
// fake stdio MCP server in the given mode.
func fakeServerConfig(t *testing.T, mode string) ServerConfig {
	t.Helper()
	return ServerConfig{
		Name:    "fake-" + mode,
		Command: os.Args[0],
		Args:    []string{"-test.run=^TestHelperFakeMCPServer$"},
		Env:     map[string]string{helperModeEnv: mode},
	}
}

func runFakeServer(mode string) {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 1<<20)
	out := bufio.NewWriter(os.Stdout)
	send := func(v any) {
		b, _ := json.Marshal(v)
		_, _ = out.Write(append(b, '\n'))
		_ = out.Flush()
	}

	for in.Scan() {
		line := in.Bytes()
		if mode == "echo" {
			_, _ = out.Write(append(append([]byte{}, line...), '\n'))
			_ = out.Flush()
			continue
		}
		if mode == "silent" {
			continue
		}

		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		if mode == "chatty" {
			// A notification and a server-to-client request that reuses the
			// pending id must both be skipped by the probe.
			send(map[string]any{"jsonrpc": "2.0", "method": "notifications/message", "params": map[string]any{}})
			send(map[string]any{"jsonrpc": "2.0", "id": req.ID, "method": "roots/list"})
		}

		send(fakeResponse(mode, req.ID, req.Method))
	}
}

func fakeResponse(mode string, id int, method string) map[string]any {
	resp := map[string]any{"jsonrpc": "2.0", "id": id}
	switch mode {
	case "noresult":
		return resp
	case "badversion":
		resp["jsonrpc"] = "1.0"
	}

	if method == "initialize" {
		result := map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"serverInfo":      map[string]any{"name": "fake", "version": "0"},
		}
		if mode == "noprotocol" {
			delete(result, "protocolVersion")
		}
		resp["result"] = result
		return resp
	}

	if mode == "badtools" {
		resp["result"] = map[string]any{"other": 1}
		return resp
	}

	n := 2
	desc := "a tool"
	if mode == "large" {
		// ~200 KiB on a single line: well past bufio.Scanner's 64 KiB default.
		n = 800
		desc = strings.Repeat("x", 256)
	}
	tools := make([]map[string]any, n)
	for i := range tools {
		tools[i] = map[string]any{"name": fmt.Sprintf("tool-%d", i), "description": desc}
	}
	resp["result"] = map[string]any{"tools": tools}
	return resp
}

// checkWithin runs CheckServer and fails the test if it does not return within
// limit, instead of hanging the whole test binary.
func checkWithin(t *testing.T, ctx context.Context, cfg ServerConfig, limit time.Duration) *ServerHealth {
	t.Helper()
	done := make(chan *ServerHealth, 1)
	go func() { done <- CheckServer(ctx, cfg) }()
	select {
	case h := <-done:
		return h
	case <-time.After(limit):
		t.Fatalf("CheckServer did not return within %s", limit)
		return nil
	}
}

// TestCheckServer_StdioHandshake covers the stdio probe's notion of "healthy":
// it must accept large single-line responses (F264/F536), must skip server
// notifications and server-to-client requests, and must reject anything that is
// not a well-formed MCP JSON-RPC 2.0 response (F265) — an echo server like `cat`
// is not healthy.
func TestCheckServer_StdioHandshake(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode      string
		want      string
		wantTools int
		wantErr   string
	}{
		{mode: "ok", want: StatusHealthy, wantTools: 2},
		{mode: "chatty", want: StatusHealthy, wantTools: 2},
		{mode: "large", want: StatusHealthy, wantTools: 800},
		{mode: "echo", want: StatusUnreachable, wantErr: "timed out"},
		{mode: "noresult", want: StatusUnreachable, wantErr: "initialize"},
		{mode: "badversion", want: StatusUnreachable, wantErr: "initialize"},
		{mode: "noprotocol", want: StatusUnreachable, wantErr: "protocolVersion"},
		{mode: "badtools", want: StatusUnreachable, wantErr: "tools/list"},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			t.Parallel()

			timeout := 20 * time.Second
			if tt.mode == "echo" {
				timeout = 2 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			h := checkWithin(t, ctx, fakeServerConfig(t, tt.mode), timeout+10*time.Second)

			if h.Status != tt.want {
				t.Fatalf("status = %q, want %q (error=%q)", h.Status, tt.want, h.Error)
			}
			if h.ToolCount != tt.wantTools {
				t.Errorf("tool count = %d, want %d", h.ToolCount, tt.wantTools)
			}
			if tt.wantErr != "" && !strings.Contains(h.Error, tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", h.Error, tt.wantErr)
			}
		})
	}
}

// TestCheckServer_StdioTimeoutReturnsPromptly is the F255 regression: a server
// that never answers initialize used to deadlock the probe, because Close
// blocked on the mutex held by the in-flight read. The deadline must now be
// enforced: CheckServer returns shortly after ctx expires.
func TestCheckServer_StdioTimeoutReturnsPromptly(t *testing.T) {
	t.Parallel()

	const deadline = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	start := time.Now()
	h := checkWithin(t, ctx, fakeServerConfig(t, "silent"), 10*time.Second)
	elapsed := time.Since(start)

	if h.Status != StatusUnreachable || !strings.Contains(h.Error, "timed out") {
		t.Errorf("status = %q error = %q, want unreachable/timed out", h.Status, h.Error)
	}
	// Killing on timeout must not wait out the graceful-close grace period.
	if elapsed > deadline+closeGracePeriod {
		t.Errorf("CheckServer took %s after a %s deadline", elapsed, deadline)
	}
}
