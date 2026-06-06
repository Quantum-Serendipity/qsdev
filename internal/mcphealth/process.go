package mcphealth

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type MCPProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func startServer(command string, args []string, env map[string]string) (*MCPProcess, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = buildProcessEnv(env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("starting server %q: %w", command, err)
	}

	return &MCPProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
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

	// Read lines until we get a response with a matching id.
	// MCP servers may emit notifications (no id field) before the response.
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

		// Skip notifications (no id field).
		if resp.ID == nil {
			continue
		}

		if *resp.ID != id {
			continue
		}

		if resp.Error != nil {
			return nil, fmt.Errorf("server error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		return resp.Result, nil
	}
}

func (p *MCPProcess) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stdin.Close()

	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		p.cmd.Process.Kill()
		<-done
		return fmt.Errorf("server did not exit within 3s, killed")
	}
}

func buildProcessEnv(env map[string]string) []string {
	osEnv := os.Environ()
	osEnvMap := make(map[string]string, len(osEnv))
	for _, entry := range osEnv {
		if k, v, ok := strings.Cut(entry, "="); ok {
			osEnvMap[k] = v
		}
	}

	result := make([]string, len(osEnv))
	copy(result, osEnv)

	for k, v := range env {
		resolved := envRefPattern.ReplaceAllStringFunc(v, func(match string) string {
			varName := match[2 : len(match)-1]
			if val, ok := osEnvMap[varName]; ok {
				return val
			}
			return match
		})
		result = append(result, k+"="+resolved)
	}

	return result
}
