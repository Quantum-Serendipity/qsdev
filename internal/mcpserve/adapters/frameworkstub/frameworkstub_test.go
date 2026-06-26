package frameworkstub_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cline"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/codex"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cursor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/frameworkstub"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/windsurf"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// frameworkCase pins the per-framework facts the shared stub must reproduce for
// each of the four real descriptors. The four cases together are the
// "four descriptors each produce an adapter with the right id/detection" check.
type frameworkCase struct {
	name        string
	adapter     *frameworkstub.Adapter
	wantID      aiframework.FrameworkID
	prefix      string // tool-name prefix, e.g. "qsdev_cursor_"
	marker      string // project-root-relative marker path
	markerIsDir bool
	clientYes   []string
	clientNo    []string
	capChecks   map[string]any // structured capability fields that must match
}

func cases() []frameworkCase {
	return []frameworkCase{
		{
			name:        "cursor",
			adapter:     cursor.New(),
			wantID:      aiframework.Cursor,
			prefix:      "qsdev_cursor_",
			marker:      ".cursor/rules",
			markerIsDir: true,
			clientYes:   []string{"Cursor", "cursor", "Cursor IDE 0.42"},
			clientNo:    []string{"claude-code", "windsurf", ""},
			capChecks: map[string]any{
				"tool_ceiling":      40,
				"tool_ceiling_kind": "client-side",
				"config_format":     "MDC",
				"enforcement_tier":  "advisory",
				"native_hooks":      false,
			},
		},
		{
			name:        "windsurf",
			adapter:     windsurf.New(),
			wantID:      aiframework.Windsurf,
			prefix:      "qsdev_windsurf_",
			marker:      ".windsurfrules",
			markerIsDir: false,
			clientYes:   []string{"Windsurf", "windsurf", "Cascade", "cascade-agent"},
			clientNo:    []string{"cursor", "claude-code", ""},
			capChecks: map[string]any{
				"tool_ceiling":      100,
				"tool_ceiling_kind": "client-side",
				"config_format":     "MDC",
				"context_model":     "Cascade",
				"enforcement_tier":  "advisory",
			},
		},
		{
			name:        "cline",
			adapter:     cline.New(),
			wantID:      aiframework.ContinueDev,
			prefix:      "qsdev_cline_",
			marker:      ".continue",
			markerIsDir: true,
			clientYes:   []string{"Cline", "cline", "Continue", "continue.dev"},
			clientNo:    []string{"cursor", "claude-code", ""},
			capChecks: map[string]any{
				"tool_ceiling_documented": false,
				"config_format":           "YAML",
				"family":                  "Continue.dev",
				"enforcement_tier":        "advisory",
			},
		},
		{
			name:        "codex",
			adapter:     codex.New(),
			wantID:      aiframework.Codex,
			prefix:      "qsdev_codex_",
			marker:      ".codex",
			markerIsDir: true,
			clientYes:   []string{"Codex", "codex", "Codex CLI"},
			clientNo:    []string{"cursor", "claude-code", ""},
			capChecks: map[string]any{
				"tool_ceiling_documented": false,
				"config_format":           "Starlark",
				"enforcement_tier":        "kernel",
				"native_sandbox":          true,
			},
		},
	}
}

func callCtx(root string) *spi.ToolCallContext {
	return &spi.ToolCallContext{ProjectRoot: root, ToolName: "test"}
}

func toolByName(t *testing.T, a *frameworkstub.Adapter, name string) spi.ToolRegistration {
	t.Helper()
	for _, reg := range a.Tools() {
		if reg.Name == name {
			return reg
		}
	}
	t.Fatalf("tool %q not registered", name)
	return spi.ToolRegistration{}
}

// TestIDAndRegistration proves each descriptor yields an adapter with the right
// FrameworkID and that its package init() self-registered it into the shared
// registry the server consumes.
func TestIDAndRegistration(t *testing.T) {
	t.Parallel()
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if id := tc.adapter.ID(); id != tc.wantID {
				t.Errorf("ID() = %q, want %q", id, tc.wantID)
			}
			found := false
			for _, a := range spi.DefaultRegistry().All() {
				if a.ID() == tc.wantID {
					found = true
				}
			}
			if !found {
				t.Errorf("%s adapter not present in spi.DefaultRegistry()", tc.name)
			}
		})
	}
}

// TestApplies proves marker detection honors each descriptor's directory-vs-file
// expectation and rejects an empty or marker-less root.
func TestApplies(t *testing.T) {
	t.Parallel()
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			if tc.adapter.Applies(ctx, "") {
				t.Error("Applies(empty) = true, want false")
			}
			if tc.adapter.Applies(ctx, t.TempDir()) {
				t.Error("Applies(no marker) = true, want false")
			}

			present := t.TempDir()
			markerPath := filepath.Join(present, filepath.FromSlash(tc.marker))
			if tc.markerIsDir {
				if err := os.MkdirAll(markerPath, 0o755); err != nil {
					t.Fatalf("creating marker dir: %v", err)
				}
			} else {
				if err := os.WriteFile(markerPath, []byte("# marker\n"), 0o644); err != nil {
					t.Fatalf("writing marker file: %v", err)
				}
			}
			if !tc.adapter.Applies(ctx, present) {
				t.Errorf("Applies(marker %q present) = false, want true", tc.marker)
			}
		})
	}
}

// TestMatchesClient proves each descriptor's client hints match the expected
// client names and reject the rest.
func TestMatchesClient(t *testing.T) {
	t.Parallel()
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, name := range tc.clientYes {
				if !tc.adapter.MatchesClient(spi.ClientInfo{Name: name}) {
					t.Errorf("MatchesClient(%q) = false, want true", name)
				}
			}
			for _, name := range tc.clientNo {
				if tc.adapter.MatchesClient(spi.ClientInfo{Name: name}) {
					t.Errorf("MatchesClient(%q) = true, want false", name)
				}
			}
		})
	}
}

// TestToolsSurface proves each adapter exposes exactly the three prefixed tools
// with non-empty descriptions, schemas, categories, and handlers.
func TestToolsSurface(t *testing.T) {
	t.Parallel()
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tools := tc.adapter.Tools()
			if len(tools) != 3 {
				t.Fatalf("Tools() returned %d tools, want 3", len(tools))
			}
			want := map[string]bool{
				tc.prefix + "info":         false,
				tc.prefix + "config":       false,
				tc.prefix + "capabilities": false,
			}
			for _, reg := range tools {
				if !strings.HasPrefix(reg.Name, tc.prefix) {
					t.Errorf("tool %q lacks %q prefix", reg.Name, tc.prefix)
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
		})
	}
}

// TestCapabilitiesTool proves the capabilities tool returns the descriptor's
// validated, non-error structured profile.
func TestCapabilitiesTool(t *testing.T) {
	t.Parallel()
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reg := toolByName(t, tc.adapter, tc.prefix+"capabilities")
			res, err := reg.Handler(context.Background(), callCtx(t.TempDir()), &spi.ToolRequest{Name: reg.Name})
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
			if caps["framework"] != string(tc.wantID) {
				t.Errorf("framework = %v, want %v", caps["framework"], tc.wantID)
			}
			for key, wantVal := range tc.capChecks {
				if got := caps[key]; got != wantVal {
					t.Errorf("caps[%q] = %v (%T), want %v (%T)", key, got, got, wantVal, wantVal)
				}
			}
		})
	}
}

// TestConfigToolGated proves the config tool returns an honest research_gated
// structured result (valid JSON Text, IsError set), not a fake success or no-op.
func TestConfigToolGated(t *testing.T) {
	t.Parallel()
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reg := toolByName(t, tc.adapter, tc.prefix+"config")
			res, err := reg.Handler(context.Background(), callCtx(t.TempDir()), &spi.ToolRequest{Name: reg.Name})
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
			if payload["framework"] != string(tc.wantID) {
				t.Errorf("framework = %v, want %v", payload["framework"], tc.wantID)
			}
			if payload["reason"] == nil || payload["reason"] == "" {
				t.Error("gated result missing reason")
			}
		})
	}
}

// TestInfoTool proves the info tool reports detected languages for a real project
// and degrades to a structured not_configured result for an empty project root.
func TestInfoTool(t *testing.T) {
	t.Parallel()
	for _, tc := range cases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reg := toolByName(t, tc.adapter, tc.prefix+"info")

			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/x\n\ngo 1.21\n"), 0o644); err != nil {
				t.Fatalf("writing go.mod: %v", err)
			}
			res, err := reg.Handler(context.Background(), callCtx(root), &spi.ToolRequest{Name: reg.Name})
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
			if structured["framework"] != string(tc.wantID) {
				t.Errorf("framework = %v, want %v", structured["framework"], tc.wantID)
			}
			if structured["project_root"] != root {
				t.Errorf("project_root = %v, want %v", structured["project_root"], root)
			}
			langs, _ := structured["languages"].([]string)
			if !containsPrefix(langs, "go") {
				t.Errorf("languages %v should include go", langs)
			}

			// Empty project root degrades gracefully to a structured not_configured.
			empty, err := reg.Handler(context.Background(), callCtx(""), &spi.ToolRequest{Name: reg.Name})
			if err != nil {
				t.Fatalf("handler error on empty root: %v", err)
			}
			if !empty.IsError {
				t.Error("expected not_configured for empty project root")
			}
			payload, ok := empty.Structured.(map[string]any)
			if !ok || payload["status"] != "not_configured" {
				t.Errorf("empty-root result should be structured not_configured, got %T %v", empty.Structured, empty.Structured)
			}
		})
	}
}

func containsPrefix(s []string, prefix string) bool {
	for _, v := range s {
		if v == prefix || strings.HasPrefix(v, prefix+" ") {
			return true
		}
	}
	return false
}
