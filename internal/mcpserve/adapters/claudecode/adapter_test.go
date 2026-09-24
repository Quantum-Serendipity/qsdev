package claudecode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// presentRoot returns a temp dir populated with Claude Code markers and a
// minimal .qsdev.yaml declaring the standard preset.
func presentRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatalf("creating .claude dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("writing CLAUDE.md: %v", err)
	}
	yaml := "version: 1\nclaude_code:\n  permission_level: standard\n"
	if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("writing .qsdev.yaml: %v", err)
	}
	return root
}

// callCtx builds a minimal ToolCallContext rooted at root.
func callCtx(root string) *spi.ToolCallContext {
	return &spi.ToolCallContext{ProjectRoot: root, ToolName: "test"}
}

// toolByName returns the registration with the given name, failing the test when
// it is absent.
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
	if id := New().ID(); id != aiframework.ClaudeCode {
		t.Errorf("ID() = %q, want %q", id, aiframework.ClaudeCode)
	}
}

// TestRegisters proves the adapter registers cleanly and exposes the Claude Code
// FrameworkID. Production wiring into spi.DefaultRegistry() now happens explicitly
// from cmd/qsdev/main.go (no longer via package init()); that wiring is covered by
// TestRegisterFrameworkAdapters in package main.
func TestRegisters(t *testing.T) {
	t.Parallel()
	reg := spi.NewAdapterRegistry()
	if err := reg.Register(New()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	found := false
	for _, a := range reg.All() {
		if a.ID() == aiframework.ClaudeCode {
			found = true
		}
	}
	if !found {
		t.Fatal("claude code adapter not present after Register()")
	}
}

func TestApplies(t *testing.T) {
	t.Parallel()
	a := New()
	tests := []struct {
		name string
		root func(t *testing.T) string
		want bool
	}{
		{"present", presentRoot, true},
		{"absent", func(t *testing.T) string { t.Helper(); return t.TempDir() }, false},
		{"empty_root", func(t *testing.T) string { t.Helper(); return "" }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := a.Applies(context.Background(), tt.root(t)); got != tt.want {
				t.Errorf("Applies() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestToolsSurface checks the five tools are registered with handlers, schemas,
// and categories.
func TestToolsSurface(t *testing.T) {
	t.Parallel()
	a := New()
	want := []string{
		toolPermissions, toolHooks, toolContextBudget, toolConfigRender, toolEnforcementGaps,
	}
	tools := a.Tools()
	if len(tools) != len(want) {
		t.Fatalf("Tools() returned %d tools, want %d", len(tools), len(want))
	}
	got := make(map[string]spi.ToolRegistration, len(tools))
	for _, reg := range tools {
		got[reg.Name] = reg
	}
	for _, name := range want {
		reg, ok := got[name]
		if !ok {
			t.Errorf("missing tool %q", name)
			continue
		}
		if reg.Handler == nil {
			t.Errorf("tool %q has nil handler", name)
		}
		if reg.Category == "" {
			t.Errorf("tool %q has empty category", name)
		}
		if len(reg.InputSchema) == 0 {
			t.Errorf("tool %q has empty input schema", name)
		}
	}
}

// TestPermissionsTool checks the standard preset yields non-empty deny rules and
// that the category filter narrows the result to a single targeted tool.
func TestPermissionsTool(t *testing.T) {
	t.Parallel()
	a := New()
	reg := toolByName(t, a, toolPermissions)
	root := presentRoot(t)

	res, err := reg.Handler(context.Background(), callCtx(root), &spi.ToolRequest{Name: toolPermissions, Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected not_configured result: %s", res.Text)
	}
	structured, ok := res.Structured.(map[string]any)
	if !ok {
		t.Fatalf("structured result is not a map: %T", res.Structured)
	}
	if dc := structured["deny_count"].(int); dc == 0 {
		t.Errorf("expected non-empty deny rules for standard preset, got %d", dc)
	}
	rules, ok := structured["rules"].([]map[string]any)
	if !ok || len(rules) == 0 {
		t.Fatalf("expected non-empty rules, got %v", structured["rules"])
	}

	// Derive a real targeted tool from the rendered rules, then assert the
	// category filter returns only rules for that tool.
	want := rules[0]["tool"].(string)
	filtered, err := reg.Handler(context.Background(), callCtx(root),
		&spi.ToolRequest{Name: toolPermissions, Arguments: map[string]any{"category": want}})
	if err != nil {
		t.Fatalf("filtered handler error: %v", err)
	}
	frules := filtered.Structured.(map[string]any)["rules"].([]map[string]any)
	if len(frules) == 0 {
		t.Fatalf("expected at least one %q rule", want)
	}
	for _, r := range frules {
		if r["tool"].(string) != want {
			t.Errorf("category filter %q leaked rule for %q: %v", want, r["tool"], r)
		}
	}
}

// TestEnforcementGapsTool checks that the active deny policy surfaces gaps with a
// required kernel tier against Claude Code's actual hook tier.
func TestEnforcementGapsTool(t *testing.T) {
	t.Parallel()
	a := New()
	reg := toolByName(t, a, toolEnforcementGaps)
	root := presentRoot(t)

	res, err := reg.Handler(context.Background(), callCtx(root), &spi.ToolRequest{Name: toolEnforcementGaps, Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected not_configured result: %s", res.Text)
	}
	structured := res.Structured.(map[string]any)
	if gc := structured["gap_count"].(int); gc == 0 {
		t.Fatal("expected at least one enforcement gap")
	}
	gaps := structured["gaps"].([]map[string]any)
	for _, g := range gaps {
		if g["required_tier"].(string) != "kernel" {
			t.Errorf("gap required_tier = %v, want kernel", g["required_tier"])
		}
		if g["actual_tier"].(string) != "hook" {
			t.Errorf("gap actual_tier = %v, want hook", g["actual_tier"])
		}
		if g["mitigation"].(string) == "" {
			t.Errorf("gap missing mitigation: %v", g)
		}
	}
}

// TestConfigRenderTool checks the render produces valid JSON files as a dry-run
// preview without writing anything, and advertises itself as read-only.
func TestConfigRenderTool(t *testing.T) {
	t.Parallel()
	a := New()
	reg := toolByName(t, a, toolConfigRender)
	root := presentRoot(t)

	if reg.Annotations.ReadOnly == nil || !*reg.Annotations.ReadOnly {
		t.Error("config render must be annotated read-only")
	}
	if props, _ := reg.InputSchema["properties"].(map[string]any); len(props) != 0 {
		t.Errorf("config render input schema = %v, want no arguments", props)
	}

	res, err := reg.Handler(context.Background(), callCtx(root), &spi.ToolRequest{Name: toolConfigRender, Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected not_configured result: %s", res.Text)
	}
	structured := res.Structured.(map[string]any)
	if dry, _ := structured["dry_run"].(bool); !dry {
		t.Error("render result does not report dry_run=true")
	}
	if structured["apply_with"] != configApplyCommand {
		t.Errorf("apply_with = %v, want %q", structured["apply_with"], configApplyCommand)
	}
	files := structured["files"].([]map[string]any)
	if len(files) == 0 {
		t.Fatal("render produced no files")
	}
	var sawSettings bool
	for _, f := range files {
		content := f["content"].(string)
		if filepath.Ext(f["path"].(string)) == ".json" && !json.Valid([]byte(content)) {
			t.Errorf("rendered %q is not valid JSON", f["path"])
		}
		if filepath.Base(f["path"].(string)) == "settings.json" {
			sawSettings = true
		}
	}
	if !sawSettings {
		t.Error("render did not produce settings.json")
	}
	for _, rel := range []string{".claude/settings.json", ".mcp.json"} {
		if _, statErr := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(statErr) {
			t.Errorf("dry-run wrote %s to disk", rel)
		}
	}
}

// TestConfigRenderTool_RefusesWrite is the regression test for F224: an agent
// that switched .qsdev.yaml to the permissive preset could call the render tool
// with write=true to rewrite its own .claude/settings.json allow list outside
// selfprotect. Every write request is refused and leaves the guardrail files
// byte-for-byte untouched; write=false is an ordinary dry-run.
func TestConfigRenderTool_RefusesWrite(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		args      map[string]any
		wantError bool
	}{
		{"write=true refused", map[string]any{"write": true}, true},
		{"write=false is a dry-run", map[string]any{"write": false}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := presentRoot(t)
			permissive := "version: 1\nclaude_code:\n  permission_level: permissive\n"
			if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte(permissive), 0o644); err != nil {
				t.Fatalf("writing .qsdev.yaml: %v", err)
			}
			settingsPath := filepath.Join(root, ".claude", "settings.json")
			existing := []byte(`{"permissions":{"allow":[],"deny":[]}}`)
			if err := os.WriteFile(settingsPath, existing, 0o644); err != nil {
				t.Fatalf("seeding settings.json: %v", err)
			}

			reg := toolByName(t, New(), toolConfigRender)
			res, err := reg.Handler(context.Background(), callCtx(root),
				&spi.ToolRequest{Name: toolConfigRender, Arguments: tt.args})
			if err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if res.IsError != tt.wantError {
				t.Fatalf("IsError = %v, want %v: %s", res.IsError, tt.wantError, res.Text)
			}
			if tt.wantError && !strings.Contains(res.Text, configApplyCommand) {
				t.Errorf("refusal %q does not point at %q", res.Text, configApplyCommand)
			}

			got, err := os.ReadFile(settingsPath)
			if err != nil {
				t.Fatalf("reading settings.json: %v", err)
			}
			if string(got) != string(existing) {
				t.Errorf("settings.json changed:\n%s", got)
			}
			if _, statErr := os.Stat(filepath.Join(root, ".mcp.json")); !os.IsNotExist(statErr) {
				t.Error(".mcp.json was written")
			}
		})
	}
}

// TestContextBudgetTool checks the default-model budget for a small project is
// within budget and reports the resolved model.
func TestContextBudgetTool(t *testing.T) {
	t.Parallel()
	a := New()
	reg := toolByName(t, a, toolContextBudget)
	root := presentRoot(t)

	res, err := reg.Handler(context.Background(), callCtx(root), &spi.ToolRequest{Name: toolContextBudget, Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected not_configured result: %s", res.Text)
	}
	structured := res.Structured.(map[string]any)
	if structured["model"].(string) != "sonnet" {
		t.Errorf("default model = %v, want sonnet", structured["model"])
	}
	if !structured["within_budget"].(bool) {
		t.Errorf("small project reported over budget: %v", structured)
	}
	if structured["max_tokens"].(int) != 200_000 {
		t.Errorf("sonnet max_tokens = %v, want 200000", structured["max_tokens"])
	}
}

// TestHooksTool checks deployed hook scripts are enumerated and cross-referenced
// with the settings.json wiring, and that a missing hooks dir degrades.
func TestHooksTool(t *testing.T) {
	t.Parallel()
	a := New()
	reg := toolByName(t, a, toolHooks)

	// Missing hooks dir → not_configured.
	bare := presentRoot(t)
	missing, err := reg.Handler(context.Background(), callCtx(bare), &spi.ToolRequest{Name: toolHooks, Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !missing.IsError {
		t.Error("expected not_configured for project without hooks directory")
	}

	// Deploy a hook plus a settings.json wiring it to PreToolUse/Bash.
	root := presentRoot(t)
	hooksDir := filepath.Join(root, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("creating hooks dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "package-guard.py"), []byte("#!/usr/bin/env python3\n"), 0o755); err != nil {
		t.Fatalf("writing hook script: %v", err)
	}
	settings := `{"permissions":{"allow":[],"deny":[]},"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/package-guard.py","timeout":30}]}]}}`
	if err := os.WriteFile(filepath.Join(root, ".claude", "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatalf("writing settings.json: %v", err)
	}

	res, err := reg.Handler(context.Background(), callCtx(root), &spi.ToolRequest{Name: toolHooks, Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected not_configured: %s", res.Text)
	}
	structured := res.Structured.(map[string]any)
	if structured["count"].(int) != 1 {
		t.Fatalf("expected 1 deployed hook, got %v", structured["count"])
	}
	hook := structured["hooks"].([]map[string]any)[0]
	if hook["name"].(string) != "package-guard.py" {
		t.Errorf("hook name = %v", hook["name"])
	}
	if !hook["wired"].(bool) {
		t.Error("expected hook to be wired in settings.json")
	}
	bindings := hook["bindings"].([]map[string]any)
	if len(bindings) != 1 || bindings[0]["event"].(string) != "PreToolUse" || bindings[0]["enforcement"].(string) != "blocking" {
		t.Errorf("unexpected hook binding: %v", bindings)
	}
}

// TestResources checks both resources advertise application/json and return
// valid JSON content.
func TestResources(t *testing.T) {
	t.Parallel()
	a := New()
	resources := a.Resources()
	if len(resources) != 2 {
		t.Fatalf("Resources() returned %d, want 2", len(resources))
	}
	for _, r := range resources {
		if r.MIMEType != mimeJSON {
			t.Errorf("resource %q MIMEType = %q, want %q", r.URI, r.MIMEType, mimeJSON)
		}
	}

	root := presentRoot(t)
	for _, r := range resources {
		res, err := r.Handler(context.Background(), callCtx(root), &spi.ResourceRequest{URI: r.URI})
		if err != nil {
			t.Fatalf("resource %q handler error: %v", r.URI, err)
		}
		if len(res.Contents) != 1 {
			t.Fatalf("resource %q returned %d contents, want 1", r.URI, len(res.Contents))
		}
		c := res.Contents[0]
		if c.MIMEType != mimeJSON {
			t.Errorf("resource %q content MIMEType = %q, want %q", r.URI, c.MIMEType, mimeJSON)
		}
		if !json.Valid([]byte(c.Text)) {
			t.Errorf("resource %q content is not valid JSON: %s", r.URI, c.Text)
		}
	}
}

func TestToolForPattern(t *testing.T) {
	t.Parallel()
	tests := []struct{ pattern, want string }{
		{"Bash(curl *)", "Bash"},
		{"Read(./secrets)", "Read"},
		{"WebFetch", "WebFetch"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := toolForPattern(tt.pattern); got != tt.want {
			t.Errorf("toolForPattern(%q) = %q, want %q", tt.pattern, got, tt.want)
		}
	}
}

func TestHookScriptName(t *testing.T) {
	t.Parallel()
	tests := []struct{ command, want string }{
		{`"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`, "package-guard.py"},
		{`"${CLAUDE_PROJECT_DIR}"/.claude/hooks/soc2-audit-log.py session_start`, "soc2-audit-log.py"},
		{"qsdev selfprotect", ""},
	}
	for _, tt := range tests {
		if got := hookScriptName(tt.command); got != tt.want {
			t.Errorf("hookScriptName(%q) = %q, want %q", tt.command, got, tt.want)
		}
	}
}

// renderedSettings runs the config-render tool (dry-run) against root and
// returns its structured payload and the rendered settings.json content.
func renderedSettings(t *testing.T, root string) (map[string]any, string) {
	t.Helper()
	reg := toolByName(t, New(), toolConfigRender)
	res, err := reg.Handler(context.Background(), callCtx(root), &spi.ToolRequest{Name: toolConfigRender, Arguments: map[string]any{}})
	if err != nil || res.IsError {
		t.Fatalf("render: err=%v result=%+v", err, res)
	}
	structured := res.Structured.(map[string]any)
	for _, f := range structured["files"].([]map[string]any) {
		if filepath.Base(f["path"].(string)) == "settings.json" {
			return structured, f["content"].(string)
		}
	}
	t.Fatal("no settings.json rendered")
	return nil, ""
}

// TestConfigRenderIncludesSecurityHooks is the regression test for a render
// whose PolicyInput carried no Hooks: the preview (and a fresh write) must carry
// the always-on security hooks `qsdev init` generates — the package guard
// (safety block) and self-protection — and must name any enabled hook choice the
// framework-agnostic render cannot express instead of silently dropping it.
func TestConfigRenderIncludesSecurityHooks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		qsdevYAML      string
		wantUnrendered []string
	}{
		{"standard preset", "version: 1\nclaude_code:\n  permission_level: standard\n", nil},
		{"strict security level", "version: 1\nsecurity:\n  level: strict\n", []string{"auto_format", "pre_commit", "audit_log"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := presentRoot(t)
			if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte(tt.qsdevYAML), 0o644); err != nil {
				t.Fatalf("writing .qsdev.yaml: %v", err)
			}
			structured, content := renderedSettings(t, root)
			for _, want := range []string{"package-guard.py", "selfprotect"} {
				if !strings.Contains(content, want) {
					t.Errorf("rendered settings.json lacks the %q hook:\n%s", want, content)
				}
			}
			got, _ := structured["unrendered_hooks"].([]string)
			if !slices.Equal(got, tt.wantUnrendered) {
				t.Errorf("unrendered_hooks = %v, want %v", got, tt.wantUnrendered)
			}
		})
	}
}

// TestUnparseableConfigIsNotConfigured is the regression test for silently
// reporting the default preset when .qsdev.yaml is present but broken: every
// policy-derived tool must degrade to not_configured carrying the parse error.
func TestUnparseableConfigIsNotConfigured(t *testing.T) {
	t.Parallel()
	root := presentRoot(t)
	if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte("version: 1\nclaude_codee:\n  permission_level: standard\n"), 0o644); err != nil {
		t.Fatalf("writing .qsdev.yaml: %v", err)
	}
	a := New()
	for _, name := range []string{toolPermissions, toolEnforcementGaps, toolConfigRender} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			reg := toolByName(t, a, name)
			res, err := reg.Handler(context.Background(), callCtx(root), &spi.ToolRequest{Name: name, Arguments: map[string]any{}})
			if err != nil {
				t.Fatalf("handler Go error: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected IsError for a broken config, got %+v", res.Structured)
			}
			m := res.Structured.(map[string]any)
			if m["status"] != "not_configured" || m["error"] == nil {
				t.Errorf("structured = %v, want not_configured with the parse error", m)
			}
		})
	}
}
