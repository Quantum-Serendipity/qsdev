package mcpserve_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
	// The framework adapters these tests exercise are registered into
	// spi.DefaultRegistry() once by this package's TestMain
	// (adapters_register_test.go), mirroring cmd/qsdev/main.go's explicit wiring.
)

// protocolVersion is the MCP revision the universal server negotiates.
const protocolVersion = "2025-11-25"

// rpcError is a JSON-RPC error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// rpcMessage is a parsed JSON-RPC response (ID set) or notification (Method set).
type rpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id"`
	Method  string          `json:"method"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

// rawLine is one newline-delimited frame read off the server's stdout pipe, or a
// terminal read error.
type rawLine struct {
	data []byte
	err  error
}

// testClient drives a *mcpserve.Server over a pair of in-process io.Pipes using
// the real stdio transport (server.NewStdioServer(...).Listen) — the same code
// path production uses — and exchanges newline-delimited JSON-RPC with it.
//
// Framing: each request is a single JSON object followed by '\n'; the server
// replies with one '\n'-terminated JSON object per request. Two dedicated pump
// goroutines isolate the test goroutine from io.Pipe's synchronous blocking: a
// writer pump serializes outbound frames (preserving request ordering, e.g. the
// initialized notification before tools/list) and a reader pump continuously
// reads inbound frames into a channel so a single bufio.Reader is never read
// concurrently. Every read is bounded by a deadline; the client never sleeps as
// an assertion. close() performs a graceful, leak-free shutdown.
type testClient struct {
	t        *testing.T
	cancel   context.CancelFunc
	inW      *io.PipeWriter // client -> server (server stdin); closing yields EOF
	outW     *io.PipeWriter // server -> client (server stdout); closing drains reader pump
	send     chan []byte
	recv     chan rawLine
	serveErr chan error
	wg       sync.WaitGroup

	nextID int64

	mu          sync.Mutex
	closed      bool
	serveResult error
	serveGot    bool
}

// newTestClient wires the pipes, starts the real stdio transport for srv in a
// goroutine, and starts the reader/writer pumps. The server, pumps, and serve
// goroutine are all torn down by close(), which is registered as a cleanup.
func newTestClient(t *testing.T, srv *mcpserve.Server) *testClient {
	t.Helper()

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())

	c := &testClient{
		t:        t,
		cancel:   cancel,
		inW:      inW,
		outW:     outW,
		send:     make(chan []byte, 16),
		recv:     make(chan rawLine, 16),
		serveErr: make(chan error, 1),
	}

	stdio := server.NewStdioServer(srv.MCPServer())
	// Keep mcp-go's internal diagnostics out of the test output; correctness is
	// covered by the assertions, not by this logger.
	stdio.SetErrorLogger(log.New(io.Discard, "", 0))
	go func() { c.serveErr <- stdio.Listen(ctx, inR, outW) }()

	c.wg.Add(2)
	go c.writerPump()
	go c.readerPump(outR)

	t.Cleanup(func() { c.close() })
	return c
}

// writerPump serializes outbound frames onto the server's stdin pipe. A write
// error (e.g. after inW is closed during shutdown) terminates the pump.
func (c *testClient) writerPump() {
	defer c.wg.Done()
	for frame := range c.send {
		if _, err := c.inW.Write(frame); err != nil {
			return
		}
	}
}

// readerPump reads newline-delimited frames off the server's stdout pipe into
// recv until EOF/error, then exits. The terminal error is delivered
// non-blockingly so the pump can never wedge during shutdown.
func (c *testClient) readerPump(r io.Reader) {
	defer c.wg.Done()
	br := newLineReader(r)
	for {
		data, err := br.readLine()
		if len(data) > 0 {
			c.recv <- rawLine{data: data}
		}
		if err != nil {
			select {
			case c.recv <- rawLine{err: err}:
			default:
			}
			return
		}
	}
}

// close cancels the serve context (simulating a SIGTERM the way runServe's
// signal-context does), drains the server's lingering blocked reader by closing
// its stdin, waits for Listen to return, then drains the client pumps. It is
// idempotent and records Listen's return value for inspection.
func (c *testClient) close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	c.cancel()
	// Closing the stdin pipe makes the server's in-flight ReadString return EOF,
	// unblocking the readNextLine goroutine so Listen can return promptly.
	_ = c.inW.Close()

	select {
	case err := <-c.serveErr:
		c.mu.Lock()
		c.serveResult = err
		c.serveGot = true
		c.mu.Unlock()
	case <-time.After(5 * time.Second):
		c.t.Error("server Listen did not return within 5s of shutdown")
	}

	// Drain the client pumps: closing the server-output pipe writer unblocks the
	// reader pump's pending read; closing send ends the writer pump's range.
	_ = c.outW.Close()
	close(c.send)
	c.wg.Wait()
}

// serveError returns Listen's recorded return value and whether it was captured.
func (c *testClient) serveError() (error, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.serveResult, c.serveGot
}

// call sends a request and returns the matching response, skipping any
// interleaved notifications. It fails the test on timeout or read error.
func (c *testClient) call(method string, params any) rpcMessage {
	c.t.Helper()
	id := c.nextID
	c.nextID++
	c.write(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	return c.readResponse(id, method)
}

// notify sends a notification (no id, no response expected).
func (c *testClient) notify(method string, params any) {
	c.t.Helper()
	c.write(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (c *testClient) write(v any) {
	c.t.Helper()
	frame, err := json.Marshal(v)
	if err != nil {
		c.t.Fatalf("marshaling request: %v", err)
	}
	c.send <- append(frame, '\n')
}

// readResponse waits up to 5s for the response whose id matches, skipping
// notifications and any unrelated frames.
func (c *testClient) readResponse(id int64, method string) rpcMessage {
	c.t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ln := <-c.recv:
			if ln.err != nil {
				c.t.Fatalf("reading response to %q (id %d): %v", method, id, ln.err)
			}
			var m rpcMessage
			if err := json.Unmarshal(ln.data, &m); err != nil {
				c.t.Fatalf("unmarshaling %q response %q: %v", method, ln.data, err)
			}
			if m.ID != nil && *m.ID == id {
				return m
			}
			// Skip notifications (no id) and any non-matching frame.
		case <-deadline:
			c.t.Fatalf("timed out waiting for response to %q (id %d)", method, id)
			return rpcMessage{}
		}
	}
}

// initialize performs the MCP handshake as clientName, asserts the negotiated
// protocol version, returns the parsed capabilities, then sends the
// notifications/initialized notification per protocol.
func (c *testClient) initialize(clientName string) map[string]json.RawMessage {
	c.t.Helper()
	resp := c.call("initialize", map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": clientName, "version": "0.0.1"},
	})
	if resp.Error != nil {
		c.t.Fatalf("initialize returned error: %+v", resp.Error)
	}
	var out struct {
		ProtocolVersion string                     `json:"protocolVersion"`
		Capabilities    map[string]json.RawMessage `json:"capabilities"`
		ServerInfo      struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		c.t.Fatalf("decoding initialize result: %v", err)
	}
	if out.ProtocolVersion != protocolVersion {
		c.t.Errorf("protocolVersion = %q, want %q", out.ProtocolVersion, protocolVersion)
	}
	if out.ServerInfo.Name == "" {
		c.t.Errorf("serverInfo.name is empty")
	}
	c.notify("notifications/initialized", map[string]any{})
	return out.Capabilities
}

// listToolNames returns the set of tool names the server exposes to this client.
func (c *testClient) listToolNames() map[string]bool {
	c.t.Helper()
	resp := c.call("tools/list", map[string]any{})
	if resp.Error != nil {
		c.t.Fatalf("tools/list returned error: %+v", resp.Error)
	}
	var out struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		c.t.Fatalf("decoding tools/list result: %v", err)
	}
	names := make(map[string]bool, len(out.Tools))
	for _, tl := range out.Tools {
		names[tl.Name] = true
	}
	return names
}

// toolCallResult holds the salient fields of a tools/call response.
type toolCallResult struct {
	text       string
	isError    bool
	structured json.RawMessage
}

// callTool invokes name with args and returns the concatenated text content, the
// isError flag, and the raw structuredContent. It fails only on a JSON-RPC
// protocol error (a tool-level IsError result is still returned to the caller).
func (c *testClient) callTool(name string, args map[string]any) toolCallResult {
	c.t.Helper()
	if args == nil {
		args = map[string]any{}
	}
	resp := c.call("tools/call", map[string]any{"name": name, "arguments": args})
	if resp.Error != nil {
		c.t.Fatalf("tools/call %q returned protocol error: %+v", name, resp.Error)
	}
	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StructuredContent json.RawMessage `json:"structuredContent"`
		IsError           bool            `json:"isError"`
	}
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		c.t.Fatalf("decoding tools/call %q result: %v", name, err)
	}
	var b strings.Builder
	for _, ct := range out.Content {
		if ct.Type == "text" {
			b.WriteString(ct.Text)
		}
	}
	return toolCallResult{text: b.String(), isError: out.IsError, structured: out.StructuredContent}
}

// readResource reads uri and returns the first content block's mime type and text.
func (c *testClient) readResource(uri string) (mime, text string) {
	c.t.Helper()
	resp := c.call("resources/read", map[string]any{"uri": uri})
	if resp.Error != nil {
		c.t.Fatalf("resources/read %q returned error: %+v", uri, resp.Error)
	}
	var out struct {
		Contents []struct {
			URI      string `json:"uri"`
			MIMEType string `json:"mimeType"`
			Text     string `json:"text"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		c.t.Fatalf("decoding resources/read %q result: %v", uri, err)
	}
	if len(out.Contents) == 0 {
		c.t.Fatalf("resources/read %q returned no contents", uri)
	}
	return out.Contents[0].MIMEType, out.Contents[0].Text
}

// lineReader reads '\n'-delimited frames; thin wrapper to keep readerPump tidy.
type lineReader struct {
	r   io.Reader
	buf []byte
}

func newLineReader(r io.Reader) *lineReader { return &lineReader{r: r} }

func (lr *lineReader) readLine() ([]byte, error) {
	for {
		if i := indexByte(lr.buf, '\n'); i >= 0 {
			line := lr.buf[:i+1]
			lr.buf = lr.buf[i+1:]
			return line, nil
		}
		tmp := make([]byte, 4096)
		n, err := lr.r.Read(tmp)
		if n > 0 {
			lr.buf = append(lr.buf, tmp[:n]...)
		}
		if err != nil {
			if i := indexByte(lr.buf, '\n'); i >= 0 {
				line := lr.buf[:i+1]
				lr.buf = lr.buf[i+1:]
				return line, nil
			}
			rest := lr.buf
			lr.buf = nil
			return rest, err
		}
	}
}

func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// writeGoProject writes a minimal initialized Go project (go.mod + .qsdev.yaml)
// into dir and returns dir.
func writeGoProject(t *testing.T, dir string) string {
	t.Helper()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module example.com/x\n\ngo 1.21\n")
	mustWrite(t, filepath.Join(dir, ".qsdev.yaml"), "version: 1\n")
	return dir
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
}

// detectedLanguages decodes the languages array from a project-info structured
// payload.
func detectedLanguages(t *testing.T, structured json.RawMessage) []string {
	t.Helper()
	var s struct {
		Languages []string `json:"languages"`
	}
	if err := json.Unmarshal(structured, &s); err != nil {
		t.Fatalf("decoding project-info structured content: %v", err)
	}
	return s.Languages
}

func hasLangPrefix(langs []string, prefix string) bool {
	for _, l := range langs {
		if l == prefix || strings.HasPrefix(l, prefix+" ") {
			return true
		}
	}
	return false
}

// TestFullProtocolFlow drives a complete session — initialize, tools/list,
// tools/call, resources/read, shutdown — over the real stdio transport against a
// real server with the generic project-context and security/devenv tools mounted
// and the Claude Code adapter applied. It is intentionally UNTAGGED.
//
// NOTE: not parallel. The stdio transport uses a process-global session
// singleton (mcp-go's stdioSessionInstance), so concurrent Listen calls would
// race; the non-parallel tests in this package run to completion before the
// parallel protocol_test resumes.
func TestFullProtocolFlow(t *testing.T) {
	dir := writeGoProject(t, t.TempDir())
	mustMkdir(t, filepath.Join(dir, ".claude"))
	mustWrite(t, filepath.Join(dir, "CLAUDE.md"), "# project\n")

	pc, err := projectctx.NewProjectContext(dir)
	if err != nil {
		t.Fatalf("NewProjectContext: %v", err)
	}

	srv := mcpserve.New(mcpserve.WithProjectRoot(dir))
	srv.MountProjectContext(pc)
	srv.MountTools(tools.All(dir))

	c := newTestClient(t, srv)

	// A Claude Code client so the tool filter reveals the qsdev_cc_* family.
	caps := c.initialize("Claude Code")
	if _, ok := caps["tools"]; !ok {
		t.Errorf("initialize did not advertise tools capability; caps=%v", capKeyNames(caps))
	}
	if _, ok := caps["resources"]; !ok {
		t.Errorf("initialize did not advertise resources capability; caps=%v", capKeyNames(caps))
	}

	names := c.listToolNames()
	if !names["qsdev_project_info"] {
		t.Errorf("generic tool qsdev_project_info missing from tools/list; got %v", sortedKeys(names))
	}
	ccTools := []string{"qsdev_cc_permissions", "qsdev_cc_hooks", "qsdev_cc_context_budget"}
	for _, want := range ccTools {
		if !names[want] {
			t.Errorf("claude-code tool %q missing from tools/list; got %v", want, sortedKeys(names))
		}
	}

	res := c.callTool("qsdev_project_info", nil)
	if res.isError {
		t.Errorf("qsdev_project_info returned an error result: %q", res.text)
	}
	if !strings.Contains(strings.ToLower(res.text), "languages: go") {
		t.Errorf("qsdev_project_info text does not mention Go detection: %q", res.text)
	}
	if langs := detectedLanguages(t, res.structured); !hasLangPrefix(langs, "go") {
		t.Errorf("qsdev_project_info structured languages = %v, want a go entry", langs)
	}

	mime, body := c.readResource("qsdev://project/detection")
	if mime != "application/json" {
		t.Errorf("detection resource mime = %q, want application/json", mime)
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(body), &probe); err != nil {
		t.Fatalf("detection resource is not valid JSON: %v\nbody=%s", err, body)
	}
	if len(probe) == 0 {
		t.Errorf("detection resource JSON is empty")
	}

	// Explicit clean shutdown; cleanup also calls close() (idempotent).
	c.close()
	if err, ok := c.serveError(); ok && !isCleanShutdown(err) {
		t.Errorf("Listen returned unexpected error on shutdown: %v", err)
	}
}

// TestMultiFrameworkMilestone is the Phase 32 milestone: a single universal
// server in multi-adapter mode simultaneously exposes the Claude Code, Cursor,
// and Cline tool families alongside the generic and security/devenv tools, and
// one tool from each framework family executes and returns a result. UNTAGGED.
func TestMultiFrameworkMilestone(t *testing.T) {
	dir := writeGoProject(t, t.TempDir())
	mustMkdir(t, filepath.Join(dir, ".claude"))
	mustMkdir(t, filepath.Join(dir, ".cursor", "rules"))
	mustMkdir(t, filepath.Join(dir, ".continue"))

	pc, err := projectctx.NewProjectContext(dir)
	if err != nil {
		t.Fatalf("NewProjectContext: %v", err)
	}

	srv := mcpserve.New(mcpserve.WithProjectRoot(dir), mcpserve.WithMultiAdapter(true))
	srv.MountProjectContext(pc)
	srv.MountTools(tools.All(dir))

	c := newTestClient(t, srv)
	c.initialize("integration-milestone-client")

	names := c.listToolNames()

	families := map[string][]string{
		"generic":     {"qsdev_project_info", "qsdev_detect"},
		"security":    {"qsdev_security_scan", "qsdev_policy_check"},
		"devenv":      {"qsdev_devenv_doctor", "qsdev_env_info"},
		"claude-code": {"qsdev_cc_permissions", "qsdev_cc_context_budget"},
		"cursor":      {"qsdev_cursor_info", "qsdev_cursor_config"},
		"cline":       {"qsdev_cline_info", "qsdev_cline_config"},
	}
	for family, want := range families {
		for _, name := range want {
			if !names[name] {
				t.Errorf("family %s: tool %q absent from the simultaneous catalog; got %v",
					family, name, sortedKeys(names))
			}
		}
	}

	// Execute one tool from each framework family and assert a usable result.
	for family, name := range map[string]string{
		"claude-code": "qsdev_cc_permissions",
		"cursor":      "qsdev_cursor_info",
		"cline":       "qsdev_cline_info",
	} {
		res := c.callTool(name, nil)
		if res.text == "" && len(res.structured) == 0 {
			t.Errorf("family %s: tool %q returned no content", family, name)
		}
	}
}

// TestCWDDetection proves the server's project root drives detection: two servers
// rooted at a Go project and a Python project respectively report their own
// language independently. UNTAGGED.
func TestCWDDetection(t *testing.T) {
	goDir := writeGoProject(t, t.TempDir())

	pyDir := t.TempDir()
	mustWrite(t, filepath.Join(pyDir, "pyproject.toml"), "[project]\nname = \"x\"\nversion = \"0\"\n")
	mustWrite(t, filepath.Join(pyDir, ".qsdev.yaml"), "version: 1\n")

	goLangs := projectInfoLanguages(t, goDir)
	if !hasLangPrefix(goLangs, "go") {
		t.Errorf("Go project: languages = %v, want a go entry", goLangs)
	}
	if hasLangPrefix(goLangs, "python") {
		t.Errorf("Go project: unexpectedly detected python: %v", goLangs)
	}

	pyLangs := projectInfoLanguages(t, pyDir)
	if !hasLangPrefix(pyLangs, "python") {
		t.Errorf("Python project: languages = %v, want a python entry", pyLangs)
	}
	if hasLangPrefix(pyLangs, "go") {
		t.Errorf("Python project: unexpectedly detected go: %v", pyLangs)
	}
}

// projectInfoLanguages spins up a server rooted at dir, drives a real session,
// and returns the languages qsdev_project_info detects.
func projectInfoLanguages(t *testing.T, dir string) []string {
	t.Helper()
	pc, err := projectctx.NewProjectContext(dir)
	if err != nil {
		t.Fatalf("NewProjectContext(%s): %v", dir, err)
	}
	srv := mcpserve.New(mcpserve.WithProjectRoot(dir))
	srv.MountProjectContext(pc)

	c := newTestClient(t, srv)
	c.initialize("integration-detect-client")
	res := c.callTool("qsdev_project_info", nil)
	if res.isError {
		t.Fatalf("qsdev_project_info(%s) errored: %q", dir, res.text)
	}
	langs := detectedLanguages(t, res.structured)
	c.close()
	return langs
}

// TestGracefulShutdown cancels the serve context (simulating the SIGTERM that
// runServe wires through signal.NotifyContext) and asserts Listen returns
// cleanly with no goroutine leak. It captures runtime.NumGoroutine() before and
// after, polling with a settle loop rather than asserting on a fixed sleep, and
// adds no third-party leak detector. UNTAGGED.
func TestGracefulShutdown(t *testing.T) {
	dir := writeGoProject(t, t.TempDir())

	// Settle any goroutines from prior subtests, then record the baseline.
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	pc, err := projectctx.NewProjectContext(dir)
	if err != nil {
		t.Fatalf("NewProjectContext: %v", err)
	}
	srv := mcpserve.New(mcpserve.WithProjectRoot(dir))
	srv.MountProjectContext(pc)

	c := newTestClient(t, srv)
	c.initialize("integration-shutdown-client")
	// Make sure the server is fully live (a request round-tripped) before cancel.
	if got := c.listToolNames(); !got["qsdev_project_info"] {
		t.Fatalf("server not serving tools before shutdown; got %v", sortedKeys(got))
	}

	// Cancel + drain everything (close cancels the context and waits for Listen).
	c.close()

	serveErr, ok := c.serveError()
	if !ok {
		t.Fatal("Listen did not return after context cancellation")
	}
	if !isCleanShutdown(serveErr) {
		t.Errorf("Listen returned a non-clean error on cancellation: %v", serveErr)
	}

	// Poll for the goroutine count to settle back to the baseline. All of the
	// transport's goroutines (input reader, notification handler, worker pool)
	// plus the test pumps must have exited.
	if !waitForGoroutines(baseline, 3*time.Second) {
		t.Errorf("goroutine leak after shutdown: baseline=%d, now=%d", baseline, runtime.NumGoroutine())
	}
}

// waitForGoroutines polls until runtime.NumGoroutine() falls back to at most
// baseline, or the timeout elapses. Returns true if it settled in time.
func waitForGoroutines(baseline int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		runtime.GC()
		if runtime.NumGoroutine() <= baseline {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// isCleanShutdown reports whether err is the expected outcome of a cancelled or
// EOF-closed stdio session: nil (EOF) or context.Canceled. Any other error is a
// genuine transport failure.
func isCleanShutdown(err error) bool {
	return err == nil || errors.Is(err, context.Canceled)
}

func capKeyNames(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
