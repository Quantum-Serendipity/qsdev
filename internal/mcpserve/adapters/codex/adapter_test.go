package codex

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// compile-time assertions, also asserted at runtime below.
var (
	_ spi.FrameworkAdapter = (*Adapter)(nil)
	_ spi.ClientMatcher    = (*Adapter)(nil)
)

func callCtx(root string) *spi.ToolCallContext {
	return &spi.ToolCallContext{ProjectRoot: root, ToolName: "test"}
}

func toolByName(t *testing.T, a *Adapter, name string) spi.ToolRegistration {
	t.Helper()
	for _, reg := range a.Tools() {
		if reg.Name == name {
			return reg
		}
	}
	t.Fatalf("tool %q not registered", name)
	return spi.ToolRegistration{}
}

func TestID(t *testing.T) {
	t.Parallel()
	if id := New().ID(); id != aiframework.Codex {
		t.Errorf("ID() = %q, want %q", id, aiframework.Codex)
	}
}

// TestRegisteredInDefaultRegistry proves the package init() self-registered the
// singleton into the shared registry the server consumes.
func TestRegisteredInDefaultRegistry(t *testing.T) {
	t.Parallel()
	found := false
	for _, a := range spi.DefaultRegistry().All() {
		if a.ID() == aiframework.Codex {
			found = true
		}
	}
	if !found {
		t.Fatal("codex adapter not present in spi.DefaultRegistry()")
	}
}

func TestMatchesClient(t *testing.T) {
	t.Parallel()
	a := New()
	tests := []struct {
		name string
		want bool
	}{
		{"Codex", true},
		{"codex", true},
		{"Codex CLI", true},
		{"cursor", false},
		{"claude-code", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := a.MatchesClient(spi.ClientInfo{Name: tt.name}); got != tt.want {
			t.Errorf("MatchesClient(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestApplies(t *testing.T) {
	t.Parallel()
	a := New()

	if a.Applies(context.Background(), "") {
		t.Error("Applies(empty) = true, want false")
	}

	absent := t.TempDir()
	if a.Applies(context.Background(), absent) {
		t.Error("Applies(no marker) = true, want false")
	}

	present := t.TempDir()
	if err := os.MkdirAll(filepath.Join(present, ".codex"), 0o755); err != nil {
		t.Fatalf("creating .codex: %v", err)
	}
	if !a.Applies(context.Background(), present) {
		t.Error("Applies(.codex present) = false, want true")
	}
}

func TestToolsSurface(t *testing.T) {
	t.Parallel()
	a := New()
	tools := a.Tools()
	if len(tools) != 3 {
		t.Fatalf("Tools() returned %d tools, want 3", len(tools))
	}
	want := map[string]bool{toolInfo: false, toolConfig: false, toolCapabilities: false}
	for _, reg := range tools {
		if !strings.HasPrefix(reg.Name, "qsdev_codex_") {
			t.Errorf("tool %q lacks qsdev_codex_ prefix", reg.Name)
		}
		if _, ok := want[reg.Name]; !ok {
			t.Errorf("unexpected tool %q", reg.Name)
		}
		want[reg.Name] = true
		if reg.Name == "" || reg.Description == "" {
			t.Errorf("tool %q has empty name or description", reg.Name)
		}
		if reg.Handler == nil {
			t.Errorf("tool %q has nil handler", reg.Name)
		}
		if reg.Category == "" {
			t.Errorf("tool %q has empty category", reg.Name)
		}
		if len(reg.InputSchema) == 0 {
			t.Errorf("tool %q has empty input schema", reg.Name)
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("missing tool %q", name)
		}
	}
}

func TestCapabilitiesTool(t *testing.T) {
	t.Parallel()
	a := New()
	reg := toolByName(t, a, toolCapabilities)
	res, err := reg.Handler(context.Background(), callCtx(t.TempDir()), &spi.ToolRequest{Name: toolCapabilities})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("capabilities reported IsError: %s", res.Text)
	}
	caps, ok := res.Structured.(map[string]any)
	if !ok {
		t.Fatalf("structured result is not a map: %T", res.Structured)
	}
	if caps["tool_ceiling_documented"].(bool) {
		t.Error("tool_ceiling_documented = true, want false (no documented ceiling)")
	}
	if caps["config_format"].(string) != "Starlark" {
		t.Errorf("config_format = %v, want Starlark", caps["config_format"])
	}
	if caps["enforcement_tier"].(string) != "kernel" {
		t.Errorf("enforcement_tier = %v, want kernel", caps["enforcement_tier"])
	}
	if !caps["native_sandbox"].(bool) {
		t.Error("native_sandbox = false, want true")
	}
	// Kernel enforcement must be qualified as a client-side sandbox guarantee.
	if scope, _ := caps["enforcement_scope"].(string); !strings.Contains(scope, "client-side") || !strings.Contains(scope, "not an MCP-protocol") {
		t.Errorf("enforcement_scope must note client-side, non-MCP-protocol nature: %q", scope)
	}
}

// TestConfigToolGated proves the config tool returns an honest research_gated
// structured result, not a fake success and not an empty no-op.
func TestConfigToolGated(t *testing.T) {
	t.Parallel()
	a := New()
	reg := toolByName(t, a, toolConfig)
	res, err := reg.Handler(context.Background(), callCtx(t.TempDir()), &spi.ToolRequest{Name: toolConfig})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !res.IsError {
		t.Fatal("gated config tool did not report IsError")
	}
	if res.Text == "" {
		t.Fatal("gated config tool returned empty text (no-op)")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(res.Text), &payload); err != nil {
		t.Fatalf("gated result is not valid JSON: %v (%s)", err, res.Text)
	}
	if payload["status"] != "research_gated" {
		t.Errorf("status = %v, want research_gated", payload["status"])
	}
	if payload["framework"] != string(aiframework.Codex) {
		t.Errorf("framework = %v, want codex", payload["framework"])
	}
	if payload["reason"] == nil || payload["reason"] == "" {
		t.Error("gated result missing reason")
	}
}

func TestInfoTool(t *testing.T) {
	t.Parallel()
	a := New()
	reg := toolByName(t, a, toolInfo)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/x\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	res, err := reg.Handler(context.Background(), callCtx(root), &spi.ToolRequest{Name: toolInfo})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("info reported IsError: %s", res.Text)
	}
	structured, ok := res.Structured.(map[string]any)
	if !ok {
		t.Fatalf("structured result is not a map: %T", res.Structured)
	}
	if structured["framework"].(string) != string(aiframework.Codex) {
		t.Errorf("framework = %v, want codex", structured["framework"])
	}
	if structured["project_root"].(string) != root {
		t.Errorf("project_root = %v, want %v", structured["project_root"], root)
	}

	// Empty project root degrades gracefully.
	empty, err := reg.Handler(context.Background(), callCtx(""), &spi.ToolRequest{Name: toolInfo})
	if err != nil {
		t.Fatalf("handler error on empty root: %v", err)
	}
	if !empty.IsError {
		t.Error("expected not_configured for empty project root")
	}
}
