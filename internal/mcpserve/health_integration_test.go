//go:build integration

// Package mcpserve_test's integration-tagged suite spawns the real qsdev binary
// as a child process and health-checks it over stdio. It is gated behind the
// `integration` build tag because it (1) builds the whole qsdev program and
// (2) executes the resulting binary — both of which may be unavailable in a
// sandboxed CI step. Run it explicitly with:
//
//	go test -tags integration -run TestHealthCheckIntegration ./internal/mcpserve/...
//
// When the binary cannot be built or executed in the current environment the
// test self-skips with a clear message rather than failing; the untagged suite
// (integration_test.go) is the must-pass gate.
package mcpserve_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

const universalImportPath = "github.com/Quantum-Serendipity/qsdev/cmd/qsdev"

// TestHealthCheckIntegration builds the qsdev binary, then drives the universal
// MCP server through mcphealth.CheckServer as a real subprocess over stdio and
// asserts it is healthy with a non-empty tool catalog. It additionally performs
// a direct initialize handshake to assert the negotiated protocol revision,
// since ServerHealth does not carry the protocol version field.
func TestHealthCheckIntegration(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go toolchain not on PATH; cannot build the qsdev binary: %v", err)
	}

	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "qsdev")

	buildCtx, cancelBuild := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelBuild()
	build := exec.CommandContext(buildCtx, goBin, "build", "-o", binPath, universalImportPath)
	if out, berr := build.CombinedOutput(); berr != nil {
		t.Skipf("could not build qsdev binary (environment may forbid it): %v\n%s", berr, out)
	}

	// Pre-flight: confirm the environment actually permits executing the built
	// binary. A sandbox that blocks running binaries fails here, and we skip.
	probeCtx, cancelProbe := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelProbe()
	if perr := exec.CommandContext(probeCtx, binPath, "--help").Run(); perr != nil {
		t.Skipf("built qsdev binary cannot be executed in this environment: %v", perr)
	}

	// A minimal initialized project for the server to root at.
	projDir := t.TempDir()
	writeFile(t, filepath.Join(projDir, "go.mod"), "module example.com/x\n\ngo 1.21\n")
	writeFile(t, filepath.Join(projDir, ".qsdev.yaml"), "version: 1\n")

	cfg := mcphealth.ServerConfig{
		Name:    "qsdev-universal",
		Command: binPath,
		Args:    []string{"mcp", "serve", "--project-root", projDir},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h := mcphealth.CheckServer(ctx, cfg)
	if h.Status != mcphealth.StatusHealthy {
		t.Fatalf("CheckServer status = %q (error=%q), want %q",
			h.Status, h.Error, mcphealth.StatusHealthy)
	}
	if h.ToolCount <= 0 {
		t.Errorf("CheckServer ToolCount = %d, want > 0", h.ToolCount)
	}

	// ServerHealth does not expose the negotiated protocol version, so assert it
	// with a direct handshake against the same binary.
	got := negotiatedProtocol(t, binPath, projDir)
	if got != "2025-11-25" {
		t.Errorf("negotiated protocolVersion = %q, want %q", got, "2025-11-25")
	}
}

// negotiatedProtocol launches the built server, performs a single initialize
// requesting revision 2025-11-25, and returns the protocolVersion the server
// echoes back.
func negotiatedProtocol(t *testing.T, binPath, projDir string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "mcp", "serve", "--project-root", projDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "qsdev-health-itest", "version": "0.0.1"},
		},
	}
	line, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshaling initialize: %v", err)
	}
	if _, err := stdin.Write(append(line, '\n')); err != nil {
		t.Fatalf("writing initialize: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var resp struct {
			ID     *int `json:"id"`
			Result struct {
				ProtocolVersion string `json:"protocolVersion"`
			} `json:"result"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			continue
		}
		if resp.ID == nil {
			continue // skip notifications
		}
		return resp.Result.ProtocolVersion
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading initialize response: %v", err)
	}
	t.Fatal("server closed stdout before answering initialize")
	return ""
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
