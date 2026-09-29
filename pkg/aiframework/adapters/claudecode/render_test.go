package claudecode_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// renderFile renders input and returns the file at path, failing if absent.
func renderFile(t *testing.T, input *aiframework.PolicyInput, path string) types.GeneratedFile {
	t.Helper()
	files, err := newAdapter().Render(context.Background(), input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	for _, f := range files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("Render() produced no %s (files: %d)", path, len(files))
	return types.GeneratedFile{}
}

func renderSettings(t *testing.T, input *aiframework.PolicyInput) ccaddon.SettingsJSON {
	t.Helper()
	f := renderFile(t, input, ".claude/settings.json")
	var settings ccaddon.SettingsJSON
	if err := json.Unmarshal(f.Content, &settings); err != nil {
		t.Fatalf("rendered settings.json is invalid: %v", err)
	}
	return settings
}

func hookCommands(settings ccaddon.SettingsJSON) []string {
	var cmds []string
	for _, matchers := range settings.Hooks {
		for _, m := range matchers {
			for _, h := range m.Hooks {
				cmds = append(cmds, h.Command)
			}
		}
	}
	return cmds
}

func containsSubstring(items []string, sub string) bool {
	return slices.ContainsFunc(items, func(s string) bool { return strings.Contains(s, sub) })
}

// TestRender_AlwaysIncludesSelfProtection guards the invariant that a
// rendered Claude Code settings.json always carries the self-protection hook,
// whatever hooks the policy lists.
func TestRender_AlwaysIncludesSelfProtection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		hooks           *aiframework.HookConfiguration
		wantPackageHook bool
	}{
		{"no hooks declared", nil, true},
		{"explicit self-protection and package guard", &aiframework.HookConfiguration{Hooks: []aiframework.HookSpec{
			{Command: string(aiframework.LogicAgentSelfProtection)},
			{Command: string(aiframework.LogicPackageGuard)},
		}}, true},
		{"only credential scan declared", &aiframework.HookConfiguration{Hooks: []aiframework.HookSpec{
			{Command: string(aiframework.LogicCredentialScan)},
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			settings := renderSettings(t, &aiframework.PolicyInput{
				ProjectRoot: t.TempDir(),
				Permissions: &aiframework.PermissionPolicy{Preset: "standard"},
				Hooks:       tt.hooks,
			})
			cmds := hookCommands(settings)
			if !containsSubstring(cmds, " selfprotect") {
				t.Errorf("rendered hooks lack self-protection: %v", cmds)
			}
			if tt.wantPackageHook && !containsSubstring(cmds, "package-guard") {
				t.Errorf("rendered hooks lack package-guard: %v", cmds)
			}
		})
	}
}

func TestRender_AskRules(t *testing.T) {
	t.Parallel()
	settings := renderSettings(t, &aiframework.PolicyInput{
		ProjectRoot: t.TempDir(),
		Permissions: &aiframework.PermissionPolicy{
			Preset:   "standard",
			AskRules: []aiframework.PermissionRule{{Pattern: "Bash(git push --unique-ask-marker *)"}},
		},
	})
	if !slices.Contains(settings.Permissions.Ask, "Bash(git push --unique-ask-marker *)") {
		t.Errorf("ask rules = %v, missing policy ask rule", settings.Permissions.Ask)
	}
}

func TestRender_SandboxPolicy(t *testing.T) {
	t.Parallel()
	settings := renderSettings(t, &aiframework.PolicyInput{
		ProjectRoot: t.TempDir(),
		Permissions: &aiframework.PermissionPolicy{Preset: "standard"},
		Sandbox: &aiframework.SandboxPolicy{
			WritablePaths:  []string{"./build"},
			ReadOnlyPaths:  []string{"./docs"},
			DeniedPaths:    []string{"/srv/secret-marker"},
			NetworkAllowed: []string{"registry.example.com"},
		},
	})
	sb := settings.Sandbox
	if sb == nil || sb.Filesystem == nil || sb.Network == nil {
		t.Fatalf("rendered settings.json lacks sandbox filesystem/network blocks: %+v", sb)
	}
	checks := []struct {
		field string
		list  []string
		want  string
	}{
		{"filesystem.allowWrite", sb.Filesystem.AllowWrite, "./build"},
		{"filesystem.denyWrite", sb.Filesystem.DenyWrite, "./docs"},
		{"filesystem.denyWrite", sb.Filesystem.DenyWrite, "/srv/secret-marker"},
		{"filesystem.denyRead", sb.Filesystem.DenyRead, "/srv/secret-marker"},
		{"network.allowedDomains", sb.Network.AllowedDomains, "registry.example.com"},
	}
	for _, c := range checks {
		if !slices.Contains(c.list, c.want) {
			t.Errorf("sandbox %s = %v, missing %q", c.field, c.list, c.want)
		}
	}
}

func TestRender_SandboxNetworkDenyIsRejected(t *testing.T) {
	t.Parallel()
	_, err := newAdapter().Render(context.Background(), &aiframework.PolicyInput{
		ProjectRoot: t.TempDir(),
		Sandbox:     &aiframework.SandboxPolicy{NetworkDenied: []string{"evil.example.com"}},
	})
	if err == nil {
		t.Fatal("Render() accepted a network deny list it cannot express")
	}
}

func TestRender_RemoteMCPServers(t *testing.T) {
	t.Parallel()
	f := renderFile(t, &aiframework.PolicyInput{
		ProjectRoot: t.TempDir(),
		MCPServers: []aiframework.MCPServerSpec{
			{Name: "remote-x", URL: "https://example.com/mcp", Transport: aiframework.TransportStreamableHTTP},
			{Name: "remote-sse", URL: "https://example.com/sse", Transport: aiframework.TransportSSE},
		},
	}, ".mcp.json")

	var doc ccaddon.McpJSON
	if err := json.Unmarshal(f.Content, &doc); err != nil {
		t.Fatalf("invalid .mcp.json: %v", err)
	}
	tests := []struct {
		name, wantType, wantURL string
	}{
		{"remote-x", "http", "https://example.com/mcp"},
		{"remote-sse", "sse", "https://example.com/sse"},
	}
	for _, tt := range tests {
		got, ok := doc.MCPServers[tt.name]
		if !ok {
			t.Errorf(".mcp.json missing %q: %+v", tt.name, doc.MCPServers)
			continue
		}
		if got.Type != tt.wantType || got.URL != tt.wantURL {
			t.Errorf("%s = %+v, want type %q url %q", tt.name, got, tt.wantType, tt.wantURL)
		}
	}
}

func TestReportGaps_NilPolicy(t *testing.T) {
	t.Parallel()
	if gaps := newAdapter().ReportGaps(context.Background(), nil); gaps != nil {
		t.Errorf("ReportGaps(nil) = %v, want nil", gaps)
	}
}
