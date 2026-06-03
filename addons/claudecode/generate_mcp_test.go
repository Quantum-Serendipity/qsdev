package claudecode_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateMcpJson_SingleKnownServer(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"github"},
	}
	cfg := claudecode.NewConfig()

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
		return
	}

	var mcp claudecode.McpJSON
	if err := json.Unmarshal(gf.Content, &mcp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}

	entry, ok := mcp.MCPServers["github"]
	if !ok {
		t.Fatal("expected 'github' key in mcpServers")
	}
	if entry.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", entry.Command)
	}
	if len(entry.Args) != 1 || entry.Args[0] != "@anthropic-ai/mcp-github" {
		t.Errorf("unexpected args: %v", entry.Args)
	}
	if entry.Env["GITHUB_TOKEN"] != "${GITHUB_TOKEN}" {
		t.Errorf("expected GITHUB_TOKEN env var, got %v", entry.Env)
	}
}

func TestGenerateMcpJson_MultipleServers(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"github", "postgres"},
	}
	cfg := claudecode.NewConfig()

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
		return
	}

	var mcp claudecode.McpJSON
	if err := json.Unmarshal(gf.Content, &mcp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}

	if _, ok := mcp.MCPServers["github"]; !ok {
		t.Error("expected 'github' key in mcpServers")
	}
	if _, ok := mcp.MCPServers["postgres"]; !ok {
		t.Error("expected 'postgres' key in mcpServers")
	}
	if len(mcp.MCPServers) != 2 {
		t.Errorf("expected 2 entries, got %d", len(mcp.MCPServers))
	}

	// Verify postgres has DATABASE_URL env.
	pg := mcp.MCPServers["postgres"]
	if pg.Env["DATABASE_URL"] != "${DATABASE_URL}" {
		t.Errorf("expected DATABASE_URL env var for postgres, got %v", pg.Env)
	}
}

func TestGenerateMcpJson_CustomServerFromConfig(t *testing.T) {
	answers := types.WizardAnswers{}
	cfg := claudecode.NewConfig(
		claudecode.WithMCPServer(claudecode.MCPServerConfig{
			Name:    "my-tool",
			Command: "/usr/local/bin/my-tool",
			Args:    []string{"--mode", "mcp"},
			Env:     map[string]string{"MY_KEY": "secret"},
		}),
	)

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
		return
	}

	var mcp claudecode.McpJSON
	if err := json.Unmarshal(gf.Content, &mcp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}

	entry, ok := mcp.MCPServers["my-tool"]
	if !ok {
		t.Fatal("expected 'my-tool' key in mcpServers")
	}
	if entry.Command != "/usr/local/bin/my-tool" {
		t.Errorf("expected command '/usr/local/bin/my-tool', got %q", entry.Command)
	}
	if len(entry.Args) != 2 || entry.Args[0] != "--mode" || entry.Args[1] != "mcp" {
		t.Errorf("unexpected args: %v", entry.Args)
	}
	if entry.Env["MY_KEY"] != "secret" {
		t.Errorf("expected MY_KEY env var, got %v", entry.Env)
	}
}

func TestGenerateMcpJson_WizardAndCustomCombined(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"github", "fetch"},
	}
	// Config provides a custom server and overrides the "github" entry.
	cfg := claudecode.NewConfig(
		claudecode.WithMCPServer(claudecode.MCPServerConfig{
			Name:    "custom-srv",
			Command: "custom-bin",
			Args:    []string{"serve"},
		}),
		claudecode.WithMCPServer(claudecode.MCPServerConfig{
			Name:    "github",
			Command: "gh-mcp",
			Args:    []string{"--enterprise"},
			Env:     map[string]string{"GH_ENTERPRISE_TOKEN": "${GH_ENTERPRISE_TOKEN}"},
		}),
	)

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
		return
	}

	var mcp claudecode.McpJSON
	if err := json.Unmarshal(gf.Content, &mcp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}

	// All three should be present.
	if len(mcp.MCPServers) != 3 {
		t.Errorf("expected 3 entries, got %d: %v", len(mcp.MCPServers), mcpKeys(mcp.MCPServers))
	}

	// "github" should be overridden by the config entry.
	gh := mcp.MCPServers["github"]
	if gh.Command != "gh-mcp" {
		t.Errorf("expected config override command 'gh-mcp', got %q", gh.Command)
	}
	if gh.Env["GH_ENTERPRISE_TOKEN"] != "${GH_ENTERPRISE_TOKEN}" {
		t.Errorf("expected GH_ENTERPRISE_TOKEN in overridden github entry, got %v", gh.Env)
	}
	// The original GITHUB_TOKEN should not be present (full override, not merge).
	if _, ok := gh.Env["GITHUB_TOKEN"]; ok {
		t.Error("expected config override to fully replace env, but GITHUB_TOKEN still present")
	}

	// "fetch" from wizard should still be present.
	if _, ok := mcp.MCPServers["fetch"]; !ok {
		t.Error("expected 'fetch' key in mcpServers")
	}

	// "custom-srv" from config should be present.
	if _, ok := mcp.MCPServers["custom-srv"]; !ok {
		t.Error("expected 'custom-srv' key in mcpServers")
	}
}

func TestGenerateMcpJson_EmptyReturnsNil(t *testing.T) {
	answers := types.WizardAnswers{}
	cfg := claudecode.NewConfig()

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf != nil {
		t.Errorf("expected nil GeneratedFile when no servers requested, got %+v", gf)
	}
}

func TestGenerateMcpJson_UnknownServerError(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"unknown-server"},
	}
	cfg := claudecode.NewConfig()

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err == nil {
		t.Fatal("expected error for unknown server, got nil")
	}
	if gf != nil {
		t.Errorf("expected nil GeneratedFile on error, got %+v", gf)
	}
	if !strings.Contains(err.Error(), "unknown-server") {
		t.Errorf("error should mention the unknown server name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "unknown MCP server") {
		t.Errorf("error should be descriptive, got: %v", err)
	}
}

func TestGenerateMcpJson_ValidJSONRoundTrip(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"github", "filesystem"},
	}
	cfg := claudecode.NewConfig(
		claudecode.WithMCPServer(claudecode.MCPServerConfig{
			Name:    "custom",
			Command: "custom-cmd",
			Args:    []string{"arg1"},
		}),
	)

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unmarshal and re-marshal to verify round-trip stability.
	var mcp claudecode.McpJSON
	if err := json.Unmarshal(gf.Content, &mcp); err != nil {
		t.Fatalf("first unmarshal failed: %v", err)
	}

	remarshaled, err := json.MarshalIndent(mcp, "", "  ")
	if err != nil {
		t.Fatalf("re-marshal failed: %v", err)
	}
	remarshaled = append(remarshaled, '\n')

	var mcp2 claudecode.McpJSON
	if err := json.Unmarshal(remarshaled, &mcp2); err != nil {
		t.Fatalf("second unmarshal failed: %v", err)
	}

	// Verify the round-tripped data matches.
	if len(mcp2.MCPServers) != len(mcp.MCPServers) {
		t.Errorf("round-trip changed server count: %d -> %d", len(mcp.MCPServers), len(mcp2.MCPServers))
	}
	for name, entry := range mcp.MCPServers {
		entry2, ok := mcp2.MCPServers[name]
		if !ok {
			t.Errorf("round-trip lost server %q", name)
			continue
		}
		if entry.Command != entry2.Command {
			t.Errorf("round-trip changed command for %q: %q -> %q", name, entry.Command, entry2.Command)
		}
	}
}

func TestGenerateMcpJson_FileMetadata(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"github"},
	}
	cfg := claudecode.NewConfig()

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
		return
	}

	if gf.Path != ".mcp.json" {
		t.Errorf("expected path '.mcp.json', got %q", gf.Path)
	}
	if gf.Mode != 0o644 {
		t.Errorf("expected mode 0o644, got %04o", gf.Mode)
	}
	if gf.Strategy != types.ThreeWayMerge {
		t.Errorf("expected strategy ThreeWayMerge, got %v", gf.Strategy)
	}
}

func TestGenerateMcpJson_FetchServer(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"fetch"},
	}
	cfg := claudecode.NewConfig()

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
		return
	}

	var mcp claudecode.McpJSON
	if err := json.Unmarshal(gf.Content, &mcp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}

	entry, ok := mcp.MCPServers["fetch"]
	if !ok {
		t.Fatal("expected 'fetch' key in mcpServers")
	}
	if entry.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", entry.Command)
	}
	if len(entry.Args) != 1 || entry.Args[0] != "@anthropic-ai/mcp-fetch" {
		t.Errorf("unexpected args: %v", entry.Args)
	}
	// fetch server should have no env vars.
	if len(entry.Env) != 0 {
		t.Errorf("expected no env for fetch server, got %v", entry.Env)
	}
}

func TestGenerateMcpJson_SocketServer(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"socket"},
	}
	cfg := claudecode.NewConfig()

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
		return
	}

	var mcp claudecode.McpJSON
	if err := json.Unmarshal(gf.Content, &mcp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}

	entry, ok := mcp.MCPServers["socket"]
	if !ok {
		t.Fatal("expected 'socket' key in mcpServers")
	}
	if entry.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", entry.Command)
	}
	if len(entry.Args) != 1 || entry.Args[0] != "@anthropic-ai/mcp-socket" {
		t.Errorf("unexpected args: %v", entry.Args)
	}
	if entry.Env["SOCKET_SECURITY_API_KEY"] != "${SOCKET_SECURITY_API_KEY}" {
		t.Errorf("expected SOCKET_SECURITY_API_KEY env var, got %v", entry.Env)
	}
}

func TestGenerateMcpJson_Context7Server(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"context7"},
	}
	cfg := claudecode.NewConfig()

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
		return
	}

	var mcp claudecode.McpJSON
	if err := json.Unmarshal(gf.Content, &mcp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}

	entry, ok := mcp.MCPServers["context7"]
	if !ok {
		t.Fatal("expected 'context7' key in mcpServers")
	}
	if entry.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", entry.Command)
	}
	if len(entry.Args) != 2 || entry.Args[0] != "-y" || entry.Args[1] != "@upstash/context7-mcp" {
		t.Errorf("unexpected args: %v", entry.Args)
	}
	// context7 should have no env vars.
	if len(entry.Env) != 0 {
		t.Errorf("expected no env for context7 server, got %v", entry.Env)
	}
}

func TestGenerateMcpJson_Context7AndGitHub(t *testing.T) {
	answers := types.WizardAnswers{
		MCPServers: []string{"context7", "github"},
	}
	cfg := claudecode.NewConfig()

	gf, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
		return
	}

	var mcp claudecode.McpJSON
	if err := json.Unmarshal(gf.Content, &mcp); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}

	if _, ok := mcp.MCPServers["context7"]; !ok {
		t.Error("expected 'context7' key in mcpServers")
	}
	if _, ok := mcp.MCPServers["github"]; !ok {
		t.Error("expected 'github' key in mcpServers")
	}
	if len(mcp.MCPServers) != 2 {
		t.Errorf("expected 2 entries, got %d", len(mcp.MCPServers))
	}
}

func TestGenerateMcpJson_Context7InKnownServers(t *testing.T) {
	// Verify that context7 is a known server by attempting to generate with it.
	answers := types.WizardAnswers{
		MCPServers: []string{"context7"},
	}
	cfg := claudecode.NewConfig()

	_, err := claudecode.GenerateMcpJson(answers, cfg)
	if err != nil {
		t.Errorf("context7 should be a known server, but got error: %v", err)
	}
}

// mcpKeys returns the keys of an MCPServerEntry map for diagnostic output.
func mcpKeys(m map[string]claudecode.MCPServerEntry) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
