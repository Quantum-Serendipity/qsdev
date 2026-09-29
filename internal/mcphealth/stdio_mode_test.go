package mcphealth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

// fakeServerModeEnv selects the behaviour of the fake stdio MCP server that the
// test binary runs as when re-executed by fakeModeServerConfig.
const fakeServerModeEnv = "QSDEV_FAKE_MCP_SERVER_MODE"

// fakeModeServerConfig returns a ServerConfig that runs this test binary as a fake
// stdio MCP server in the given mode.
func fakeModeServerConfig(t *testing.T, mode string) ServerConfig {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return ServerConfig{
		Name:    "fake-" + mode,
		Command: exe,
		Args:    []string{"-test.run=^TestFakeMCPServerProcess$"},
		Env:     map[string]string{fakeServerModeEnv: mode},
	}
}

// TestFakeMCPServerProcess is not a real test: when re-executed with
// fakeServerModeEnv set, it acts as a stdio MCP server and exits.
//
// Modes:
//   - healthy: emits a notification before each response and answers
//     initialize and tools/list (two tools).
//   - hang: reads requests but never replies, like a server stuck downloading
//     or waiting on auth.
func TestFakeMCPServerProcess(t *testing.T) {
	mode := os.Getenv(fakeServerModeEnv)
	if mode == "" {
		t.Skip("helper process for stdio health-check tests")
	}
	runFakeMCPServer(mode)
	os.Exit(0)
}

func runFakeMCPServer(mode string) {
	if mode == "hang" {
		time.Sleep(30 * time.Second)
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req struct {
			ID     *int   `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(scanner.Bytes(), &req) != nil || req.ID == nil {
			continue // notifications get no response
		}
		fmt.Println(`{"jsonrpc":"2.0","method":"notifications/message","params":{"level":"info"}}`)
		switch req.Method {
		case "initialize":
			fmt.Printf(`{"jsonrpc":"2.0","id":%d,"result":{"protocolVersion":"2025-03-26","capabilities":{"tools":{}}}}`+"\n", *req.ID)
		case "tools/list":
			fmt.Printf(`{"jsonrpc":"2.0","id":%d,"result":{"tools":[{"name":"a"},{"name":"b"}]}}`+"\n", *req.ID)
		default:
			fmt.Printf(`{"jsonrpc":"2.0","id":%d,"error":{"code":-32601,"message":"method not found"}}`+"\n", *req.ID)
		}
	}
}

func TestCheckServer_StdioHealthy(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h := CheckServer(ctx, fakeModeServerConfig(t, "healthy"))

	if h.Status != StatusHealthy {
		t.Fatalf("status = %q, want %q (error=%q)", h.Status, StatusHealthy, h.Error)
	}
	if h.ToolCount != 2 {
		t.Errorf("tool count = %d, want 2", h.ToolCount)
	}
}

// TestCheckServer_StdioTimeoutHolds is the F535 regression: after the deadline
// CheckServer's deferred Close blocked on the lock held by the handshake
// goroutine, which was itself blocked reading a reply that never came, so a
// server that never answered hung the probe until the server exited.
func TestCheckServer_StdioTimeoutHolds(t *testing.T) {
	t.Parallel()

	const deadline = 300 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	done := make(chan *ServerHealth, 1)
	start := time.Now()
	go func() { done <- CheckServer(ctx, fakeModeServerConfig(t, "hang")) }()

	select {
	case h := <-done:
		if elapsed := time.Since(start); elapsed > deadline+2*time.Second {
			t.Errorf("CheckServer took %v, want about the %v deadline", elapsed, deadline)
		}
		if h.Status != StatusUnreachable {
			t.Errorf("status = %q, want %q", h.Status, StatusUnreachable)
		}
		if h.Error != "health check timed out" {
			t.Errorf("error = %q, want the timeout message", h.Error)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("CheckServer did not return after its deadline")
	}
}
