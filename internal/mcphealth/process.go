package mcphealth

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// MCPProcess is a stdio MCP server started for a health probe.
type MCPProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner

	// mu serialises SendRequest. Close and kill deliberately never take it:
	// SendRequest holds it across a blocking read, and only killing the
	// process can unblock that read.
	mu sync.Mutex

	closeOnce sync.Once
	closeErr  error
}

// closeGracePeriod is how long Close waits for the server to exit on its own
// after stdin is closed before killing it.
const closeGracePeriod = 3 * time.Second

// waitDelay bounds how long cmd.Wait may block on I/O after the process has
// exited, as a backstop should the process grow copy goroutines.
const waitDelay = time.Second

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonRPCResponse is any JSON-RPC message read from a server. Method is set
// only on server-to-client requests and notifications, which are not responses.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// answers reports whether the message is the response to request id, rather
// than a notification, a server-to-client request, or a reply to another id.
func (r *jsonRPCResponse) answers(id int) bool {
	return r.Method == "" && r.ID != nil && *r.ID == id
}

// outcome validates a response and returns its result, or its JSON-RPC error
// as a Go error. Both transports use it, so they agree on what a valid MCP
// response is: JSON-RPC 2.0 carrying exactly one of result or error.
func (r *jsonRPCResponse) outcome() (json.RawMessage, error) {
	if r.JSONRPC != "2.0" {
		return nil, fmt.Errorf("response is not JSON-RPC 2.0 (jsonrpc=%q)", r.JSONRPC)
	}
	hasResult := len(r.Result) > 0 && !bytes.Equal(r.Result, []byte("null"))
	if hasResult == (r.Error != nil) {
		return nil, errors.New("response must carry exactly one of result or error")
	}
	if r.Error != nil {
		return nil, fmt.Errorf("server error %d: %s", r.Error.Code, r.Error.Message)
	}
	return r.Result, nil
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func startServer(command string, args []string, env map[string]string) (*MCPProcess, error) {
	cmd := exec.Command(command, args...) //nolint:gosec // argv is an explicit array; no shell interpolation
	cmd.Env = buildProcessEnv(env)
	// Run the server in its own process group so launchers (npx, uvx, sh) and
	// the real server they spawn can be killed together.
	setProcessGroup(cmd)
	cmd.WaitDelay = waitDelay

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	// A nil Stderr connects the server to the null device. io.Discard would
	// make os/exec copy a stderr pipe in a goroutine that cmd.Wait joins, so
	// any grandchild still holding the pipe would keep Close blocked.
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("starting server %q: %w", command, err)
	}

	// MCP stdio messages are single lines; a tools/list response with full
	// schemas easily exceeds bufio.Scanner's 64 KiB default, so allow the same
	// size the HTTP probe does.
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), maxHealthResponseBytes)

	return &MCPProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: scanner,
	}, nil
}

func (p *MCPProcess) SendRequest(id int, method string, params json.RawMessage) (json.RawMessage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	data = append(data, '\n')
	if _, err := p.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("writing request: %w", err)
	}

	// Read lines until we get the response to this request. MCP servers may
	// emit notifications and server-to-client requests (which carry a method)
	// before it, and non-JSON lines are log noise; skip all of those.
	for {
		if !p.stdout.Scan() {
			if err := p.stdout.Err(); err != nil {
				return nil, fmt.Errorf("reading response: %w", err)
			}
			return nil, fmt.Errorf("reading response: unexpected EOF")
		}

		line := p.stdout.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		if !resp.answers(id) {
			continue
		}

		return resp.outcome()
	}
}

// Close shuts the server down: it closes stdin, waits closeGracePeriod for the
// server to exit, then kills its process group. It is idempotent and never
// blocks on an in-flight SendRequest.
func (p *MCPProcess) Close() error {
	return p.shutdown(closeGracePeriod)
}

// shutdown implements Close with a configurable grace period.
func (p *MCPProcess) shutdown(grace time.Duration) error {
	p.closeOnce.Do(func() {
		_ = p.stdin.Close()

		done := make(chan error, 1)
		go func() { done <- p.cmd.Wait() }()

		select {
		case p.closeErr = <-done:
			// The server exited; still kill its group so any children a
			// launcher left behind do not outlive the probe.
			p.kill()
		case <-time.After(grace):
			p.kill()
			<-done
			p.closeErr = fmt.Errorf("server did not exit within %s, killed", grace)
		}
	})
	return p.closeErr
}

// kill immediately terminates the server and every process in its group.
// Killing closes the server's stdout, which unblocks a SendRequest waiting on
// a response. It is safe to call concurrently with SendRequest and Close.
func (p *MCPProcess) kill() {
	killProcessGroup(p.cmd.Process)
}

func buildProcessEnv(env map[string]string) []string {
	osEnv := os.Environ()
	result := make([]string, len(osEnv), len(osEnv)+len(env))
	copy(result, osEnv)

	for k, v := range env {
		result = append(result, k+"="+expandVars(v, os.LookupEnv))
	}

	return result
}
