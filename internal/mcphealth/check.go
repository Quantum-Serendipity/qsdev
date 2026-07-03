package mcphealth

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

var initializeParams = json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"qsdev-health","version":"1.0"}}`)

var toolsListParams = json.RawMessage(`{}`)

// maxHealthResponseBytes bounds how much of a health-probe response we read, so a
// hostile or misconfigured endpoint cannot stream unbounded data into the check.
// The JSON-RPC request/response shapes are defined in process.go and shared with
// the stdio probe.
const maxHealthResponseBytes = 1 << 20 // 1 MiB

// CheckServer probes a single MCP server and returns its health status.
// The provided context controls cancellation and timeout; callers should use
// context.WithTimeout to enforce a deadline.
func CheckServer(ctx context.Context, cfg ServerConfig) *ServerHealth {
	h := &ServerHealth{Name: cfg.Name}

	prereqs := checkPrerequisites(cfg)
	h.Prerequisites = prereqs

	unmet := false
	for _, p := range prereqs {
		if !p.Met {
			unmet = true
			break
		}
	}

	if unmet {
		h.Status = StatusDegraded
		h.Error = "one or more prerequisites not met"
		return h
	}

	start := time.Now()

	if cfg.URL != "" {
		return checkHTTPServer(ctx, cfg, h, start)
	}

	proc, err := startServer(cfg.Command, cfg.Args, cfg.Env)
	if err != nil {
		h.Status = StatusUnreachable
		h.Error = fmt.Sprintf("starting server: %s", err)
		return h
	}
	defer proc.Close()

	// The stdio transport's SendRequest cannot observe ctx cancellation, so run
	// the shared handshake in a goroutine and enforce the deadline via select.
	type probeResult struct {
		status    string
		err       string
		toolCount int
	}

	ch := make(chan probeResult, 1)
	go func() {
		status, errMsg, toolCount := handshake(proc)
		ch <- probeResult{status: status, err: errMsg, toolCount: toolCount}
	}()

	var r probeResult
	select {
	case r = <-ch:
	case <-ctx.Done():
		r = probeResult{status: StatusUnreachable, err: "health check timed out"}
	}

	h.Status = r.status
	h.Error = r.err
	h.ToolCount = r.toolCount

	h.ResponseMs = time.Since(start).Milliseconds()
	return h
}

// CheckAll probes all servers in parallel and returns an aggregated report.
// The provided context controls cancellation and timeout for each individual
// server check.
func CheckAll(ctx context.Context, servers map[string]ServerConfig) *HealthReport {
	report := &HealthReport{
		TotalCount: len(servers),
		CheckedAt:  time.Now(),
	}

	if len(servers) == 0 {
		report.Servers = []ServerHealth{}
		return report
	}

	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]ServerHealth, len(names))
	var wg sync.WaitGroup

	for i, name := range names {
		wg.Add(1)
		go func(idx int, cfg ServerConfig) {
			defer wg.Done()
			h := CheckServer(ctx, cfg)
			results[idx] = *h
		}(i, servers[name])
	}

	wg.Wait()

	report.Servers = results
	for _, s := range results {
		if s.Status == StatusHealthy {
			report.HealthyCount++
		}
	}

	return report
}

func checkPrerequisites(cfg ServerConfig) []PrerequisiteStatus {
	var prereqs []PrerequisiteStatus

	for _, envKey := range cfg.RequiredEnv {
		_, set := os.LookupEnv(envKey)
		p := PrerequisiteStatus{
			Name: envKey,
			Type: "env",
			Met:  set,
		}
		if !set {
			p.Detail = fmt.Sprintf("environment variable %s is not set", envKey)
		}
		prereqs = append(prereqs, p)
	}

	return prereqs
}

type toolsListResult struct {
	Tools []json.RawMessage `json:"tools"`
}

func countTools(result json.RawMessage) int {
	var tlr toolsListResult
	if err := json.Unmarshal(result, &tlr); err != nil {
		return 0
	}
	return len(tlr.Tools)
}

// transport is the minimal request/response surface a health probe needs. Both
// the stdio process (*MCPProcess) and the HTTP client (*httpTransport) implement
// it, letting a single handshake define "healthy" identically across transports.
type transport interface {
	SendRequest(id int, method string, params json.RawMessage) (json.RawMessage, error)
}

// handshake runs the shared MCP health probe over a transport: initialize, then
// tools/list, counting the advertised tools. Sharing it ensures the stdio and
// HTTP probes agree on what "healthy" means — a server whose initialize succeeds
// but whose tools/list fails is Unreachable on both transports, not just stdio.
func handshake(t transport) (status, errMsg string, toolCount int) {
	if _, err := t.SendRequest(1, "initialize", initializeParams); err != nil {
		return StatusUnreachable, fmt.Sprintf("initialize: %s", err), 0
	}

	result, err := t.SendRequest(2, "tools/list", toolsListParams)
	if err != nil {
		return StatusUnreachable, fmt.Sprintf("tools/list: %s", err), 0
	}

	return StatusHealthy, "", countTools(result)
}

func checkHTTPServer(ctx context.Context, cfg ServerConfig, h *ServerHealth, start time.Time) *ServerHealth {
	status, errMsg, toolCount := handshake(&httpTransport{ctx: ctx, client: http.DefaultClient, url: cfg.URL})

	h.Status = status
	h.Error = errMsg
	h.ToolCount = toolCount
	h.ResponseMs = time.Since(start).Milliseconds()
	return h
}

// httpTransport probes a Streamable-HTTP MCP server. Each SendRequest POSTs a
// JSON-RPC request and validates the reply, so the shared handshake behaves the
// same as it does over stdio.
type httpTransport struct {
	ctx    context.Context
	client *http.Client
	url    string
}

// SendRequest POSTs a real MCP JSON-RPC request rather than a bare GET. A bare
// GET only opens a server->client SSE stream; a spec-compliant Streamable-HTTP
// server that offers no such stream answers it with 405, which a status-only
// check misreads as unhealthy (false negative). POSTing and requiring a valid
// JSON-RPC result also rejects a plain non-MCP web server that merely returns
// 2xx (the symmetric false positive). It mirrors *MCPProcess.SendRequest: a
// JSON-RPC error reply becomes a Go error, otherwise the result is returned.
func (t *httpTransport) SendRequest(id int, method string, params json.RawMessage) (json.RawMessage, error) {
	reqBody, err := json.Marshal(jsonRPCRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req, err := http.NewRequestWithContext(t.ctx, http.MethodPost, t.url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("returned HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	rpc, err := decodeJSONRPCResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("endpoint did not return a valid MCP response: %w", err)
	}
	if rpc.Error != nil {
		return nil, fmt.Errorf("server error %d: %s", rpc.Error.Code, rpc.Error.Message)
	}

	return rpc.Result, nil
}

// decodeJSONRPCResponse extracts the JSON-RPC response from an MCP
// Streamable-HTTP initialize reply, which may be a direct application/json body
// or a single SSE `data:` event (text/event-stream). It returns an error when
// the body is not a JSON-RPC 2.0 message — i.e. the endpoint does not speak MCP.
func decodeJSONRPCResponse(resp *http.Response) (*jsonRPCResponse, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHealthResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	payload := body
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		payload = sseData(body)
		if payload == nil {
			return nil, fmt.Errorf("no SSE data event in response")
		}
	}
	var rpc jsonRPCResponse
	if err := json.Unmarshal(bytes.TrimSpace(payload), &rpc); err != nil {
		return nil, fmt.Errorf("response is not JSON-RPC: %w", err)
	}
	if rpc.JSONRPC != "2.0" || (rpc.Result == nil && rpc.Error == nil) {
		return nil, fmt.Errorf("response is not a JSON-RPC 2.0 result")
	}
	return &rpc, nil
}

// sseData returns the concatenated payload of the first SSE event's data: lines.
func sseData(body []byte) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), maxHealthResponseBytes)
	var data []byte
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimSpace(strings.TrimPrefix(line, "data:"))...)
		case line == "" && data != nil:
			return data // a blank line terminates the event
		}
	}
	return data
}
