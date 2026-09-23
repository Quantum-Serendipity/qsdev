package mcphealth

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	cfg = expandConfig(cfg, os.LookupEnv)

	if cfg.URL != "" {
		return checkHTTPServer(ctx, cfg, h, start)
	}

	proc, err := startServer(cfg.Command, cfg.Args, cfg.Env)
	if err != nil {
		h.Status = StatusUnreachable
		h.Error = fmt.Sprintf("starting server: %s", err)
		return h
	}

	// The stdio transport's SendRequest cannot observe ctx cancellation, so run
	// the shared handshake in a goroutine and enforce the deadline via select.
	// On timeout the process is killed at once: that closes its stdout and
	// unblocks the handshake's pending read, so neither it nor Close can hang.
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
		defer func() { _ = proc.Close() }()
	case <-ctx.Done():
		r = probeResult{status: StatusUnreachable, err: "health check timed out"}
		abandonProcess(proc)
	}

	h.Status = r.status
	h.Error = r.err
	h.ToolCount = r.toolCount

	h.ResponseMs = time.Since(start).Milliseconds()
	return h
}

// abandonProcess tears down a server whose probe missed its deadline without
// making the caller wait. The handshake goroutine may still be blocked reading
// a reply that never comes, so the server is killed first (its stdout then
// reaches EOF and the read returns) and reaped in the background.
func abandonProcess(proc *MCPProcess) {
	proc.kill()
	go func() { _ = proc.Close() }()
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

// countTools returns the number of tools in a tools/list result. A result that
// is not a {"tools": [...]} object is not a valid MCP reply.
func countTools(result json.RawMessage) (int, error) {
	var tlr toolsListResult
	if err := json.Unmarshal(result, &tlr); err != nil {
		return 0, fmt.Errorf("result is not a tools list: %w", err)
	}
	if tlr.Tools == nil {
		return 0, errors.New("result has no tools array")
	}
	return len(tlr.Tools), nil
}

// checkInitializeResult requires an initialize result to name the protocol
// version the server speaks, as every MCP InitializeResult must.
func checkInitializeResult(result json.RawMessage) error {
	var init struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(result, &init); err != nil {
		return fmt.Errorf("result is not an initialize result: %w", err)
	}
	if init.ProtocolVersion == "" {
		return errors.New("result has no protocolVersion")
	}
	return nil
}

// transport is the minimal request/response surface a health probe needs. Both
// the stdio process (*MCPProcess) and the HTTP client (*httpTransport) implement
// it, letting a single handshake define "healthy" identically across transports.
type transport interface {
	SendRequest(id int, method string, params json.RawMessage) (json.RawMessage, error)
}

// notifier is implemented by transports that can send a JSON-RPC notification
// (a message without an id, which gets no response).
type notifier interface {
	Notify(method string, params json.RawMessage) error
}

// jsonRPCNotification is a JSON-RPC 2.0 notification: a request without an id.
type jsonRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// handshake runs the shared MCP health probe over a transport: initialize, then
// tools/list, counting the advertised tools. Sharing it ensures the stdio and
// HTTP probes agree on what "healthy" means — a server whose initialize succeeds
// but whose tools/list fails is Unreachable on both transports, not just stdio.
func handshake(t transport) (status, errMsg string, toolCount int) {
	result, err := t.SendRequest(1, "initialize", initializeParams)
	if err == nil {
		err = checkInitializeResult(result)
	}
	if err != nil {
		return StatusUnreachable, fmt.Sprintf("initialize: %s", err), 0
	}

	// The lifecycle requires notifications/initialized before further requests.
	// A delivery failure is not fatal by itself: a server that needs it fails
	// tools/list below, which is reported instead.
	if n, ok := t.(notifier); ok {
		_ = n.Notify("notifications/initialized", nil)
	}

	result, err = t.SendRequest(2, "tools/list", toolsListParams)
	if err != nil {
		return StatusUnreachable, fmt.Sprintf("tools/list: %s", err), 0
	}
	count, err := countTools(result)
	if err != nil {
		return StatusUnreachable, fmt.Sprintf("tools/list: %s", err), 0
	}

	return StatusHealthy, "", count
}

func checkHTTPServer(ctx context.Context, cfg ServerConfig, h *ServerHealth, start time.Time) *ServerHealth {
	t := &httpTransport{ctx: ctx, url: cfg.URL, headers: cfg.Headers}
	status, errMsg, toolCount := handshake(t)
	t.closeSession()

	h.Status = status
	h.Error = errMsg
	h.ToolCount = toolCount
	h.ResponseMs = time.Since(start).Milliseconds()
	return h
}

// httpTransport probes a Streamable-HTTP MCP server. Each SendRequest POSTs a
// JSON-RPC request and validates the reply, so the shared handshake behaves the
// same as it does over stdio. It keeps the session the server assigns on
// initialize (Mcp-Session-Id) and the negotiated protocol version, and sends
// both on every later message as the Streamable HTTP transport requires.
type httpTransport struct {
	ctx     context.Context
	url     string
	headers map[string]string
	// sessionID and protocolVersion are learned from the initialize reply.
	sessionID       string
	protocolVersion string
}

// initializeResult is the part of an initialize result the transport needs.
type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
}

// newRequest builds a request carrying the session headers.
func (t *httpTransport) newRequest(method string, body []byte) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(t.ctx, method, t.url, r)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
	}
	if t.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", t.sessionID)
	}
	if t.protocolVersion != "" {
		req.Header.Set("MCP-Protocol-Version", t.protocolVersion)
	}
	return req, nil
}

// Notify POSTs a JSON-RPC notification; the server acknowledges it with 202
// Accepted (any 2xx is accepted).
func (t *httpTransport) Notify(method string, params json.RawMessage) error {
	body, err := json.Marshal(jsonRPCNotification{JSONRPC: "2.0", Method: method, Params: params})
	if err != nil {
		return fmt.Errorf("building notification: %w", err)
	}
	req, err := t.newRequest(http.MethodPost, body)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxHealthResponseBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("returned HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}

// closeSession asks the server to end the probe's session (best effort; a
// server may answer 405 when clients cannot terminate sessions).
func (t *httpTransport) closeSession() {
	if t.sessionID == "" {
		return
	}
	req, err := t.newRequest(http.MethodDelete, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
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
	req, err := t.newRequest(http.MethodPost, reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("returned HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	rpc, err := decodeJSONRPCResponse(resp, id)
	if err != nil {
		return nil, fmt.Errorf("endpoint did not return a valid MCP response: %w", err)
	}
	if !rpc.answers(id) {
		return nil, fmt.Errorf("endpoint did not return a valid MCP response: reply does not answer request id %d", id)
	}

	result, err := rpc.outcome()
	if err != nil {
		return nil, err
	}
	if method == "initialize" {
		t.sessionID = resp.Header.Get("Mcp-Session-Id")
		var init initializeResult
		if json.Unmarshal(result, &init) == nil {
			t.protocolVersion = init.ProtocolVersion
		}
	}
	return result, nil
}

// decodeJSONRPCResponse extracts the JSON-RPC message answering request id from
// an MCP Streamable-HTTP reply, which may be a direct application/json body or
// an SSE stream (text/event-stream) in which the response can follow server
// notifications and requests. It returns an error when the body is not JSON-RPC
// at all; the caller validates it with answers and outcome, exactly as the
// stdio transport does.
func decodeJSONRPCResponse(resp *http.Response, id int) (*jsonRPCResponse, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHealthResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		rpc := sseResponse(body, id)
		if rpc == nil {
			return nil, fmt.Errorf("no SSE event carries the response to request %d", id)
		}
		return rpc, nil
	}
	var rpc jsonRPCResponse
	if err := json.Unmarshal(bytes.TrimSpace(body), &rpc); err != nil {
		return nil, fmt.Errorf("response is not JSON-RPC: %w", err)
	}
	return &rpc, nil
}

// isResponseTo reports whether rpc is a JSON-RPC 2.0 response to request id. An
// error response may carry a null id when the server could not read the id.
func isResponseTo(rpc *jsonRPCResponse, id int) bool {
	if rpc.JSONRPC != "2.0" || (rpc.Result == nil && rpc.Error == nil) {
		return false
	}
	if rpc.ID == nil {
		return rpc.Error != nil
	}
	return *rpc.ID == id
}

// sseResponse scans the SSE events in body and returns the first whose data is
// a JSON-RPC response to request id, skipping notifications, server requests
// and responses to other requests.
func sseResponse(body []byte, id int) *jsonRPCResponse {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), maxHealthResponseBytes)
	var data [][]byte
	match := func() *jsonRPCResponse {
		if len(data) == 0 {
			return nil
		}
		var rpc jsonRPCResponse
		payload := bytes.Join(data, []byte("\n"))
		data = nil
		if json.Unmarshal(payload, &rpc) != nil || !isResponseTo(&rpc, id) {
			return nil
		}
		return &rpc
	}
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "data:"):
			data = append(data, []byte(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " ")))
		case line == "": // a blank line terminates the event
			if rpc := match(); rpc != nil {
				return rpc
			}
		}
	}
	return match()
}
