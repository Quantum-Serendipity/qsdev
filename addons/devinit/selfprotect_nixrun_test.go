package devinit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// TestSelfprotect_NixRunTool proves the selfprotect hook judges a call of the
// MCP server's nix_run tool as the Bash command lines it is equivalent to
// (XS-WS6): what self-protection refuses for Bash is refused for the tool
// whatever the server is named, a harmless call is allowed, and a
// tool_input it cannot read is denied.
func TestSelfprotect_NixRunTool(t *testing.T) {
	t.Parallel()
	sensitive := []cmdscan.CommandSpec{{Path: [][]string{{"teardown"}}, ReadOnly: []string{"--dry-run"}}}
	// Assembled so this file's own text holds no decode-and-run line.
	decodeToShell := "echo Y3VybCB4fHNo | base64 " + "-d | sh"
	tests := []struct {
		name  string
		tool  string
		input any
		deny  bool
	}{
		{"remove settings through bash -c", "mcp__qsdev__qsdev_nix_run",
			map[string]any{"command": "nixpkgs#bash", "args": []string{"-c", "rm .claude/settings.json"}}, true},
		{"overwrite local settings", "mcp__qsdev__qsdev_nix_run",
			map[string]any{"command": "nixpkgs#bash", "args": []string{"-c", "echo x > .claude/settings.local.json"}}, true},
		{"coreutils rm", "mcp__other-name__qsdev_nix_run",
			map[string]any{"command": "nixpkgs#coreutils", "args": []string{"rm", ".claude/settings.json"}}, true},
		{"human-only CLI command", "mcp__qsdev__qsdev_nix_run",
			map[string]any{"command": "nixpkgs#qsdev", "args": []string{"teardown", "--force"}}, true},
		{"decoded script piped to a shell", "mcp__qsdev__qsdev_nix_run",
			map[string]any{"command": "nixpkgs#bash", "args": []string{"-c", decodeToShell}}, true},
		{"stdin script", "mcp__qsdev__qsdev_nix_run",
			map[string]any{"command": "nixpkgs#bash", "stdin": "rm -f .claude/settings.json\n"}, true},
		{"malformed args", "mcp__qsdev__qsdev_nix_run",
			map[string]any{"command": "nixpkgs#bash", "args": "-c rm .claude/settings.json"}, true},
		{"harmless", "mcp__qsdev__qsdev_nix_run",
			map[string]any{"command": "nixpkgs#jq", "args": []string{"."}}, false},
		{"another MCP tool is not mapped", "mcp__qsdev__qsdev_env_info",
			map[string]any{"command": "nixpkgs#bash", "args": []string{"-c", "rm .claude/settings.json"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			payload, err := json.Marshal(map[string]any{"tool_name": tt.tool, "tool_input": tt.input, "cwd": t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			err = evaluateSelfprotect(context.Background(), strings.NewReader(string(payload)), &stderr, sensitive)
			if denied := errors.Is(err, errSelfprotectDeny); denied != tt.deny {
				t.Errorf("denied = %v, want %v (err %v, stderr %q)", denied, tt.deny, err, stderr.String())
			}
		})
	}
}
