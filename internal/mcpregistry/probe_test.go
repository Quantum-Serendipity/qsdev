package mcpregistry

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

// TestProbeSkipReason covers the launch-safety classification health probes use
// to avoid downloading and running packages or spawning qsdev's own server.
func TestProbeSkipReason(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      mcphealth.ServerConfig
		wantSkip bool
	}{
		{"npx launcher", mcphealth.ServerConfig{Command: "npx", Args: []string{"-y", "pkg"}}, true},
		{"absolute uvx", mcphealth.ServerConfig{Command: "/nix/store/x-uv/bin/uvx", Args: []string{"pkg"}}, true},
		{"pipx", mcphealth.ServerConfig{Command: "pipx", Args: []string{"run", "pkg"}}, true},
		{"windows npx shim", mcphealth.ServerConfig{Command: `C:\Program Files\nodejs\npx.cmd`, Args: []string{"-y", "pkg"}}, true},
		{"cmd /c wrapper", mcphealth.ServerConfig{Command: "cmd", Args: []string{"/c", "npx", "-y", "pkg"}}, true},
		{"sh -c wrapper", mcphealth.ServerConfig{Command: "sh", Args: []string{"-c", "uvx --from semble[mcp] semble"}}, true},
		{"env wrapper", mcphealth.ServerConfig{Command: "/usr/bin/env", Args: []string{"FOO=1", "bunx", "pkg"}}, true},
		{"self server", mcphealth.ServerConfig{Command: "qsdev", Args: []string{"mcp", "serve", "--stdio"}}, true},
		{"self server absolute", mcphealth.ServerConfig{Command: "/run/current-system/sw/bin/qsdev", Args: []string{"mcp", "serve"}}, true},
		{"empty command", mcphealth.ServerConfig{}, true},
		{"other qsdev subcommand", mcphealth.ServerConfig{Command: "qsdev", Args: []string{"mcp", "health"}}, false},
		{"single-module server", mcphealth.ServerConfig{Command: "qsdev", Args: []string{"mcp", "serve", "--module", "agent-postmortem"}}, false},
		{"single-module server, joined flag", mcphealth.ServerConfig{Command: "qsdev", Args: []string{"mcp", "serve", "--module=version-sentinel"}}, false},
		{"local binary", mcphealth.ServerConfig{Command: "/opt/bin/server", Args: []string{"--port", "0"}}, false},
		{"launcher name as substring", mcphealth.ServerConfig{Command: "/opt/bin/npx-free-server"}, false},
		{"remote url", mcphealth.ServerConfig{URL: "https://example.test/mcp"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ProbeSkipReason(tt.cfg) != ""; got != tt.wantSkip {
				t.Errorf("ProbeSkipReason(%+v) skip = %v, want %v", tt.cfg, got, tt.wantSkip)
			}
		})
	}
}
