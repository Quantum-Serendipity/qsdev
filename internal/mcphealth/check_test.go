package mcphealth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckServer_MissingBinary(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Name:    "nonexistent",
		Command: "this-binary-does-not-exist-xyz-999",
		Args:    []string{"--stdio"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h := CheckServer(ctx, cfg)

	if h.Name != "nonexistent" {
		t.Errorf("name = %q, want %q", h.Name, "nonexistent")
	}
	if h.Status != StatusUnreachable {
		t.Errorf("status = %q, want %q", h.Status, StatusUnreachable)
	}
	if h.Error == "" {
		t.Error("expected non-empty error for missing binary")
	}
}

func TestCheckServer_Prerequisites(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Name:        "needs-secret",
		Command:     "bash",
		Args:        []string{"-c", "echo hello"},
		RequiredEnv: []string{"QSDEV_TEST_MISSING_SECRET_XYZ_999"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h := CheckServer(ctx, cfg)

	if h.Status != StatusDegraded {
		t.Errorf("status = %q, want %q", h.Status, StatusDegraded)
	}

	if len(h.Prerequisites) == 0 {
		t.Fatal("expected at least one prerequisite status")
	}

	p := h.Prerequisites[0]
	if p.Met {
		t.Error("expected prerequisite to be unmet")
	}
	if p.Name != "QSDEV_TEST_MISSING_SECRET_XYZ_999" {
		t.Errorf("prereq name = %q, want QSDEV_TEST_MISSING_SECRET_XYZ_999", p.Name)
	}
}

func TestCheckAll_EmptyServers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	report := CheckAll(ctx, map[string]ServerConfig{})

	if report.TotalCount != 0 {
		t.Errorf("total = %d, want 0", report.TotalCount)
	}
	if report.HealthyCount != 0 {
		t.Errorf("healthy = %d, want 0", report.HealthyCount)
	}
	if len(report.Servers) != 0 {
		t.Errorf("servers = %d, want 0", len(report.Servers))
	}
	if report.CheckedAt.IsZero() {
		t.Error("expected CheckedAt to be set")
	}
}

func TestCheckAll_MixedResults(t *testing.T) {
	t.Parallel()

	servers := map[string]ServerConfig{
		"missing-binary": {
			Name:    "missing-binary",
			Command: "this-binary-does-not-exist-xyz-999",
		},
		"missing-prereq": {
			Name:        "missing-prereq",
			Command:     "bash",
			RequiredEnv: []string{"QSDEV_TEST_MISSING_MIX_12345"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	report := CheckAll(ctx, servers)

	if report.TotalCount != 2 {
		t.Errorf("total = %d, want 2", report.TotalCount)
	}
	if report.HealthyCount != 0 {
		t.Errorf("healthy = %d, want 0", report.HealthyCount)
	}
	if len(report.Servers) != 2 {
		t.Fatalf("servers = %d, want 2", len(report.Servers))
	}

	statuses := map[string]string{}
	for _, s := range report.Servers {
		statuses[s.Name] = s.Status
	}

	if statuses["missing-binary"] != StatusUnreachable {
		t.Errorf("missing-binary status = %q, want %q", statuses["missing-binary"], StatusUnreachable)
	}
	if statuses["missing-prereq"] != StatusDegraded {
		t.Errorf("missing-prereq status = %q, want %q", statuses["missing-prereq"], StatusDegraded)
	}
}

// mcpInitResult writes a minimal JSON-RPC initialize result and, for a bare GET
// (the SSE-stream open), replies 405 exactly like a spec-compliant Streamable-
// HTTP MCP server that offers no server-initiated stream.
func mcpInitResult(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","capabilities":{}}}`))
}

// TestCheckServer_HTTPProbesMCP is the M7 regression. The old check did a bare
// GET and treated only 2xx as healthy, so it (a) reported a spec-compliant MCP
// server that answers a GET with 405 as unhealthy (false negative) and (b)
// reported any non-MCP web server returning 2xx as healthy (false positive).
// The probe now POSTs a real MCP initialize and validates a JSON-RPC result.
func TestCheckServer_HTTPProbesMCP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{
			// A real MCP server: 405 to GET, valid JSON-RPC result to the POST.
			name:    "mcp server answering 405 to GET is healthy",
			handler: mcpInitResult,
			want:    StatusHealthy,
		},
		{
			name: "mcp server replying over SSE is healthy",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\n\n"))
			},
			want: StatusHealthy,
		},
		{
			name: "plain non-MCP web server returning 200 is unreachable",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte("<html>hello</html>"))
			},
			want: StatusUnreachable,
		},
		{
			name: "jsonrpc error response is unreachable",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"bad"}}`))
			},
			want: StatusUnreachable,
		},
		{
			name:    "not found is unreachable",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
			want:    StatusUnreachable,
		},
		{
			name:    "server error is unreachable",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			want:    StatusUnreachable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			h := CheckServer(ctx, ServerConfig{Name: "http-server", URL: srv.URL})

			if h.Status != tt.want {
				t.Errorf("status = %q, want %q (error=%q)", h.Status, tt.want, h.Error)
			}
			if tt.want != StatusHealthy && h.Error == "" {
				t.Error("expected a non-empty error for an unhealthy status")
			}
		})
	}
}

// mcpRequestMethod reads the JSON-RPC method from an MCP request body so a test
// handler can respond differently to initialize versus tools/list.
func mcpRequestMethod(r *http.Request) string {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Method string `json:"method"`
	}
	_ = json.Unmarshal(body, &req)
	return req.Method
}

// TestCheckServer_HTTPToolsListError covers bug #13: previously the HTTP probe
// stopped after initialize, so a server whose initialize succeeds but whose
// tools/list fails was reported Healthy — while the identical failure over stdio
// reports Unreachable. The shared handshake now sends tools/list over HTTP too,
// so a tools/list error yields Unreachable, matching the stdio transport.
func TestCheckServer_HTTPToolsListError(t *testing.T) {
	t.Parallel()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch mcpRequestMethod(r) {
		case "initialize":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","capabilities":{}}}`))
		default: // tools/list
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"error":{"code":-32601,"message":"method not found"}}`))
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h := CheckServer(ctx, ServerConfig{Name: "http-server", URL: srv.URL})

	if h.Status != StatusUnreachable {
		t.Errorf("status = %q, want %q (error=%q)", h.Status, StatusUnreachable, h.Error)
	}
	if h.Error == "" {
		t.Error("expected a non-empty error when tools/list fails")
	}
	if h.ToolCount != 0 {
		t.Errorf("tool count = %d, want 0", h.ToolCount)
	}
}

// TestCheckServer_HTTPReportsToolCount covers the other half of bug #13: the old
// HTTP probe never called tools/list, so it always reported ToolCount==0. The
// shared handshake now counts the advertised tools over HTTP, matching stdio.
func TestCheckServer_HTTPReportsToolCount(t *testing.T) {
	t.Parallel()

	const wantTools = 3
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch mcpRequestMethod(r) {
		case "initialize":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","capabilities":{}}}`))
		default: // tools/list
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"a"},{"name":"b"},{"name":"c"}]}}`))
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h := CheckServer(ctx, ServerConfig{Name: "http-server", URL: srv.URL})

	if h.Status != StatusHealthy {
		t.Errorf("status = %q, want %q (error=%q)", h.Status, StatusHealthy, h.Error)
	}
	if h.ToolCount != wantTools {
		t.Errorf("tool count = %d, want %d", h.ToolCount, wantTools)
	}
}

func TestCountTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "two tools",
			input: `{"tools":[{"name":"a"},{"name":"b"}]}`,
			want:  2,
		},
		{
			name:  "empty tools list",
			input: `{"tools":[]}`,
			want:  0,
		},
		{
			name:  "invalid json",
			input: `not json`,
			want:  0,
		},
		{
			name:  "missing tools key",
			input: `{"other":"value"}`,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := countTools([]byte(tt.input))
			if got != tt.want {
				t.Errorf("countTools(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
